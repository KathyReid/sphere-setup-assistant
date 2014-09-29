package main

import (
	"encoding/json"
	"github.com/theojulienne/go-wireless/iwlib"
	"os/exec"
	"log"
)

type WifiNetwork struct {
	SSID string `json:"name"`
}

type WifiCredentials struct {
	SSID string `json:"ssid"`
	Key string `json:"key"`
}

const WLANInterfaceTemplate = "iface wlan0 inet dhcp\n"

func GetSetupRPCRouter(wifi_manager *WifiManager) *JSONRPCRouter {
	rpc_router := &JSONRPCRouter{}
	rpc_router.Init()
	rpc_router.AddHandler("sphere.setup.ping", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		pong := JSONRPCResponse{"2.0", request.Id, 1234, nil}
		resp <- pong

		return resp
	})

	rpc_router.AddHandler("sphere.setup.get_visible_wifi_networks", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		// Before we search for wifi networks, disable any that are try-fail-ing
		wifi_manager.DisableAllNetworks()
		
		networks, err := iwlib.GetWirelessNetworks("wlan0")
		if err == nil {
			wifi_networks := make([]WifiNetwork, len(networks))
			for i, network := range networks {
				wifi_networks[i].SSID = network.SSID
			}

			resp <- JSONRPCResponse{"2.0", request.Id, wifi_networks, nil}
		} else {
			resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, "Could not retrieve WiFi networks", nil}}
		}

		return resp
	})

	rpc_router.AddHandler("sphere.setup.connect_wifi_network", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		wifi_creds := new(WifiCredentials)
		b, _ := json.Marshal(request.Params[0])
		json.Unmarshal(b, wifi_creds)

		log.Println("Got wifi credentials", wifi_creds)

		go func() {
			WriteToFile("/etc/network/interfaces.d/wlan0", WLANInterfaceTemplate)
			
			states := wifi_manager.WatchState()

			wifi_manager.AddStandardNetwork(wifi_creds.SSID, wifi_creds.Key)
			wifi_manager.Controller.ReloadConfiguration()
			
			success := true
			for {
				state := <- states
				if state == WifiStateConnected {
					success = true
					break
				} else if state == WifiStateInvalidKey {
					success = false
					break
				}
			}

			wifi_manager.UnwatchState(states)
			
			if success {
				serial_number, err := exec.Command("/opt/ninjablocks/bin/sphere-serial").Output()
				if err != nil {
					// ow ow ow
				}
				
				pong := JSONRPCResponse{"2.0", request.Id, string(serial_number), nil}
				resp <- pong
			} else {
				resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, "Could not connect to specified WiFi network, is the key correct?", nil}}
			}
		}()

		return resp
	})

	return rpc_router
}
