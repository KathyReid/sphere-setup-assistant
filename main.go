package main

import (
	"flag"
	"log"
	"time"

	nlog "github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/sphere-go-led-controller/model"
	"github.com/paypal/gatt"
)

const WirelessNetworkInterface = "wlan0"

// consider the wifi to be invalid after this timeout
const WirelessStaleTimeout = time.Second * 30 // FIXME: INCREASE THIS. a few minutes at least when not in testing.

var firewallHook = flag.Bool("firewall-hook", false, "Sets up the firewall based on configuration options, and nothing else.")
var factoryReset = false

// factoryReset - we can't use anything that requires MQTT in this mode

var logger = nlog.GetLogger("sphere-setup")

func main() {
	// ap0 adhoc/hostap management
	flag.BoolVar(&factoryReset, "factory-reset", false, "Run in factory reset mode.")

	config := LoadConfig("/etc/opt/ninja/setup-assistant.conf")
	apManager := NewAccessPointManager(config)

	flag.Parse()
	if *firewallHook {
		logger.Debugf("Setting ip firewall rules...")
		apManager.SetupFirewall()
		return
	}
	var pairing_ui ConsolePairingUI
	var controlChecker *ControlChecker

	restartHeartbeat := false

	startResetMonitor(func(m *model.ResetMode) {
		if pairing_ui == nil || controlChecker == nil {
			return
		}
		if m.Mode == "none" {
			if restartHeartbeat {
				// only restart the heartbeat if we stopped it previously
				controlChecker.StartHeartbeat()
			}
		} else {
			restartHeartbeat = controlChecker.StopHeartbeat()
			pairing_ui.DisplayResetMode(m)
		}
	})

	apManager.WriteAPConfig()
	if config.Wireless_Host.Always_Active {
		apManager.StartHostAP()
	} else {
		apManager.StopHostAP()
	}

	// wlan0 client management
	wifi_manager, err := NewWifiManager(WirelessNetworkInterface)
	if err != nil {
		log.Fatal("Could not setup manager for wlan0, does the interface exist?")
	}
	defer wifi_manager.Cleanup()

	pairing_ui, err = NewConsolePairingUI()
	if err != nil {
		log.Fatal("Could not setup ninja connection")
	}

	// This name is sent in the BLE advertising packet,
	// and is used by the phone to see that this is a
	// sphere, and if its in factory reset mode.
	serviceName := "ninjasphere"
	if factoryReset {
		serviceName = "ninjasphere-reset"
	}

	srv := &gatt.Server{
		Name: serviceName,
		Connect: func(c gatt.Conn) {
			logger.Infof("BLE Connect")
			pairing_ui.DisplayIcon("ble-connected.gif")
		},
		Disconnect: func(c gatt.Conn) {
			logger.Infof("BLE Disconnect")
			pairing_ui.DisplayIcon("ble-disconnected.gif")
		},
		StateChange: func(state string) {
			logger.Infof("BLE State Change: %s", state)
		},
	}

	// start by registering the RPC functions that will be accessible
	// once the client has authenticated
	// We pass in the ble server so that we can close the connection once the updates are installed
	// (THIS SHOULD HAPPEN OVER WIFI INSTEAD!)
	rpc_router := GetSetupRPCRouter(wifi_manager, srv, pairing_ui)

	auth_handler := new(OneTimeAuthHandler)
	auth_handler.Init("spheramid")

	controlChecker = NewControlChecker(pairing_ui)

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
		logger.Warningf("Wireless is stale! Invalid SSID, router down, or not in range.")

		if !is_serving_pairer {
			is_serving_pairer = true
			logger.Infof("Launching BLE pairing assistant...")
			go srv.AdvertiseAndServe()

			// and if the hostap isn't normally active, make it active
			if !config.Wireless_Host.Always_Active {
				logger.Infof("Launching AdHoc pairing assistant...")
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
		logger.Infof("State: %v", state)

		switch state {
		case WifiStateConnected:
			if wireless_stale != nil {
				wireless_stale.Stop()
			}
			wireless_stale = nil
			logger.Infof("Connected and attempting to get IP.")

			/*if !config.Wireless_Host.Enables_Control {
				// if the wireless AP mode hasn't already enabled normal control, then enable it now that wifi works
				controlChecker.StartHeartbeat()
			}*/

			if is_serving_pairer {
				is_serving_pairer = false

				// We need to keep the server open for now, as we are sending update progress to it, and accepting
				// led drawing messages. Later, this will be over wifi and we can close it here.
				//srv.Close()

				// and if the hostap isn't normally active, turn it off again
				if !config.Wireless_Host.Always_Active {
					logger.Infof("Terminating AdHoc pairing assistant.")
					apManager.StopHostAP()
				}
			}

		case WifiStateDisconnected:
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

				logger.Warningf("Wireless key is invalid!")
			}
		}
	}
}
