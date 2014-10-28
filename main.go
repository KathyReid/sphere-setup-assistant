package main

import (
	"flag"
	"log"
	"time"

	"github.com/juju/loggo"
	"github.com/paypal/gatt"
)

const WirelessNetworkInterface = "wlan0"

// consider the wifi to be invalid after this timeout
const WirelessStaleTimeout = time.Second * 10 // FIXME: INCREASE THIS. a few minutes at least when not in testing.

var firewallHook = flag.Bool("firewall-hook", false, "Sets up the firewall based on configuration options, and nothing else.")

var logger = loggo.GetLogger("sphere-setup")

func main() {
	// ap0 adhoc/hostap management
	config := LoadConfig("/etc/opt/ninja/setup-assistant.conf")
	apManager := NewAccessPointManager(config)

	flag.Parse()
	if *firewallHook {
		log.Println("Setting ip firewall rules...")
		apManager.SetupFirewall()
		return
	}

	apManager.WriteAPConfig()
	if config.Wireless_Host.Always_Active {
		apManager.StartHostAP()
	} else {
		apManager.StopHostAP()
	}

	// wlan0 client management
	iman := NewInterfaceManager(WirelessNetworkInterface)
	wifi_manager, err := NewWifiManager(WirelessNetworkInterface)
	if err != nil {
		log.Fatal("Could not setup manager for wlan0, does the interface exist?")
	}
	defer wifi_manager.Cleanup()

	// start by registering the RPC functions that will be accessible
	// once the client has authenticated
	rpc_router := GetSetupRPCRouter(wifi_manager)

	srv := &gatt.Server{Name: "ninjasphere"}

	auth_handler := new(OneTimeAuthHandler)
	auth_handler.Init("spheramid")

	pairing_ui, err := NewConsolePairingUI()

	if err != nil {
		log.Fatal("Could not setup ninja connection")
	}

	controlChecker := NewControlChecker(pairing_ui)

	RegisterSecuredRPCService(srv, rpc_router, auth_handler, pairing_ui)

	// Start the server
	//log.Println("Starting setup assistant...");
	//log.Fatal(srv.AdvertiseAndServe())

	states := wifi_manager.WatchState()

	//wifi_manager.WifiConfigured()

	var wireless_stale *time.Timer

	is_serving_pairer := false

	// start by forcing the state to Disconnected.
	// reloading the configuration in wpa_supplicant will also force this,
	// but we need to do it here in case we are already disconnected
	states <- WifiStateDisconnected
	wifi_manager.Controller.ReloadConfiguration()

	handleBadWireless := func() {
		log.Println("Wireless is stale! Invalid SSID, router down, or not in range.")

		if !is_serving_pairer {
			is_serving_pairer = true
			log.Println("Launching BLE pairing assistant...")
			go srv.AdvertiseAndServe()

			// and if the hostap isn't normally active, make it active
			if !config.Wireless_Host.Always_Active {
				log.Println("Launching AdHoc pairing assistant...")
				apManager.StartHostAP()
			}
		}
	}

	wifi_configured, _ := wifi_manager.WifiConfigured()
	if !wifi_configured {
		// when wireless isn't configured at all, automatically start doing this, don't wait for staleness
		handleBadWireless()
	}

	if config.Wireless_Host.Enables_Control {
		// the wireless AP causes control to be enabled, so we just start the heartbeat immediately
		controlChecker.StartHeartbeat()
	}

	for {
		state := <-states
		log.Println("State:", state)

		switch state {
		case WifiStateConnected:
			if wireless_stale != nil {
				wireless_stale.Stop()
			}
			wireless_stale = nil
			iman.Up()
			log.Println("Connected and attempting to get IP.")

			if !config.Wireless_Host.Enables_Control {
				// if the wireless AP mode hasn't already enabled normal control, then enable it now that wifi works
				controlChecker.StartHeartbeat()
			}

			if is_serving_pairer {
				is_serving_pairer = false
				srv.Close()

				// and if the hostap isn't normally active, turn it off again
				if !config.Wireless_Host.Always_Active {
					log.Println("Terminating AdHoc pairing assistant.")
					apManager.StopHostAP()
				}
			}

		case WifiStateDisconnected:
			iman.Down()
			if wireless_stale == nil {
				wireless_stale = time.AfterFunc(WirelessStaleTimeout, handleBadWireless)
			}

		case WifiStateInvalidKey:
			if wireless_stale == nil {
				wireless_stale = time.AfterFunc(WirelessStaleTimeout, handleBadWireless)
			}
			wifi_configured, _ = wifi_manager.WifiConfigured()
			if wifi_configured {
				// not stale, we actually know the key is wrong
				// FIXME: report back to the user! for now we're just going to let staleness timeout
				/*if wireless_stale != nil {
					wireless_stale.Stop()
				}
				wireless_stale = nil*/

				log.Println("Wireless key is invalid!")
			}
		}
	}
}
