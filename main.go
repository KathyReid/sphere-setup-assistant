package main

import (
	"github.com/paypal/gatt"
	"log"
	"time"
)

const WirelessNetworkInterface = "wlan0"

// consider the wifi to be invalid after this timeout
const WirelessStaleTimeout = time.Second * 10

func main() {
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

	RegisterSecuredRPCService(srv, rpc_router, auth_handler)

	// Start the server
	//log.Println("Starting setup assistant...");
	//log.Fatal(srv.AdvertiseAndServe())

	states := wifi_manager.WatchState()

	wifi_manager.WifiConfigured()
	
	var wireless_stale *time.Timer

	// start by forcing the state to Disconnected.
	// reloading the configuration in wpa_supplicant will also force this,
	// but we need to do it here in case we are already disconnected
	states <- WifiStateDisconnected
	wifi_manager.Controller.ReloadConfiguration()

	for {
		state := <- states
		log.Println("State:", state)

		switch state {
		case WifiStateConnected:
			if wireless_stale != nil {
				wireless_stale.Stop()
			}
			wireless_stale = nil
			iman.Up()
			log.Println("Connected and attempting to get IP.")

		case WifiStateDisconnected:
			iman.Down()
			if wireless_stale == nil {
				wireless_stale = time.AfterFunc(WirelessStaleTimeout, func() {
					log.Println("Wireless is stale! Invalid SSID, router down, or not in range.")
				})
			}

		case WifiStateInvalidKey:
			// not stale, we actually know the key is wrong
			if wireless_stale != nil {
				wireless_stale.Stop()
			}
			wireless_stale = nil

			log.Println("Wireless key is invalid!")

		}
	}
}
