package main

import (
	"log"
	"time"
	"flag"

	"github.com/paypal/gatt"
)

const WirelessNetworkInterface = "wlan0"

// consider the wifi to be invalid after this timeout
const WirelessStaleTimeout = time.Second * 10 // FIXME: INCREASE THIS. a few minutes at least when not in testing.

var firewallHook = flag.Bool("firewall-hook", false, "Sets up the firewall based on configuration options, and nothing else.")

func main() {
	// ap0 adhoc/hostap management
	config := LoadConfig("/etc/opt/ninjablocks/setup-assistant.conf")
	apManager := NewAccessPointManager(config)
	
	flag.Parse()
	if (*firewallHook) {
		log.Println("Setting ip firewall rules...")
		apManager.SetupFirewall()
		return
	}

	apManager.WriteAPConfig()

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
		}
	}

	wifi_configured, _ := wifi_manager.WifiConfigured()
	if !wifi_configured {
		// when wireless isn't configured at all, automatically start doing this, don't wait for staleness
		handleBadWireless()
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

			controlChecker.StartHeartbeat()

			if is_serving_pairer {
				is_serving_pairer = false
				srv.Close()
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
