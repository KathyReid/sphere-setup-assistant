package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-wireless/iwlib"
	"github.com/paypal/gatt"
)

type WifiNetwork struct {
	SSID string `json:"name"`
}

type WifiCredentials struct {
	SSID string `json:"ssid"`
	Key  string `json:"key"`
}

const WLANInterfaceTemplate = "iface wlan0 inet dhcp\n"

// This is ugly... but for some reason go-ninja was only delivering the progress to one of the
// listeners, so rpc and http need to share.
var lastUpdateProgress map[string]interface{}

func GetSetupRPCRouter(conn *ninja.Connection, wifi_manager *WifiManager, srv *gatt.Server, pairing_ui ConsolePairingUI) *JSONRPCRouter {

	rpc_router := &JSONRPCRouter{}
	rpc_router.Init()
	rpc_router.AddHandler("sphere.setup.ping", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		pong := JSONRPCResponse{"2.0", request.Id, 1234, nil}
		resp <- pong

		return resp
	})

	rpc_router.AddHandler("sphere.setup.get_visible_wifi_networks", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		pairing_ui.DisplayIcon("wifi-searching.gif")

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

	rpc_router.AddHandler("sphere.setup.connect_wifi_network", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		pairing_ui.DisplayIcon("wifi-connecting.gif")

		wifi_creds := new(WifiCredentials)
		b, _ := json.Marshal(request.Params[0])
		json.Unmarshal(b, wifi_creds)

		logger.Debugf("Got wifi credentials %v", wifi_creds)

		go func() {
			success := wifi_manager.SetCredentials(wifi_creds)
			if success {
				var err error
				var path string
				var serial_number string

				pairing_ui.DisplayIcon("wifi-connected.gif")

				path, err = exec.LookPath("sphere-serial")
				if err == nil {
					serial_number, err = exec.Command("sphere-serial").Output()
				}
				if err == nil {
					pong := JSONRPCResponse{"2.0", request.Id, string(serial_number), nil}
					resp <- pong
				} else {
					logger.Errorf("failed to obtain serial number: %v", err)
					resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, "Failed to obtain serial number", nil}}
				}

			} else {
				pairing_ui.DisplayIcon("wifi-failed.gif")
				resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, "Could not connect to specified WiFi network, is the key correct?", nil}}
			}
		}()

		return resp
	})

	rpc_router.AddHandler("sphere.setup.acknowledge_wifi_connected", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)
		go func() {
			wifi_manager.ConnectionAcknowledged()
			logger.Infof("Received acknowledgement of wifi connected from app.")
			pairing_ui.DisplayIcon("wifi-connected.gif")
			resp <- JSONRPCResponse{"2.0", request.Id, nil, nil}
		}()

		return resp
	})

	if !factoryReset {

		updateService := conn.GetServiceClient("$node/" + config.Serial() + "/updates")
		ledService := conn.GetServiceClient("$node/" + config.Serial() + "/led-controller")

		rpc_router.AddHandler("sphere.setup.start_update", func(request JSONRPCRequest) chan JSONRPCResponse {
			resp := make(chan JSONRPCResponse, 1)

			var response bool

			logger.Infof("Starting update id: %d. Waiting for response....", request.Id)

			err := updateService.Call("start", nil, &response, time.Second*10)

			if response {
				lastUpdateProgress = nil
			}

			logger.Infof("Got update start response: %d. %v", request.Id, response)

			if err == nil {
				resp <- JSONRPCResponse{"2.0", request.Id, &response, nil}
			} else {
				resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, fmt.Sprintf("%s", err), nil}}
			}

			return resp
		})

		rpc_router.AddHandler("sphere.setup.get_update_progress", func(request JSONRPCRequest) chan JSONRPCResponse {
			resp := make(chan JSONRPCResponse, 1)

			logger.Infof("Requesting update progress id:%s", request.Id)

			resp <- JSONRPCResponse{"2.0", request.Id, lastUpdateProgress, nil}

			logger.Infof("Sent progress %v", lastUpdateProgress)

			running, ok := lastUpdateProgress["running"]

			// If we have sent back a progress that shows the update is finished... shut down the ble after 5 seconds.
			if ok && running.(bool) == false { /*running:*/
				logger.Infof("Update is finished!")
				go func() {
					time.Sleep(time.Second * 5)
					srv.Close()
				}()
			}

			return resp
		})

		rpc_router.AddHandler("sphere.setup.display_drawing", func(request JSONRPCRequest) chan JSONRPCResponse {
			resp := make(chan JSONRPCResponse, 1)

			var response json.RawMessage

			err := ledService.Call("displayDrawing", request.Params, &response, time.Second*10)

			if err == nil {
				resp <- JSONRPCResponse{"2.0", request.Id, &response, nil}
			} else {
				resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, fmt.Sprintf("%s", err), nil}}
			}

			return resp
		})

		rpc_router.AddHandler("sphere.setup.draw", func(request JSONRPCRequest) chan JSONRPCResponse {
			resp := make(chan JSONRPCResponse, 1)

			var response json.RawMessage

			err := ledService.Call("draw", request.Params, &response, time.Second*10)

			if err == nil {
				resp <- JSONRPCResponse{"2.0", request.Id, &response, nil}
			} else {
				resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, fmt.Sprintf("%s", err), nil}}
			}

			return resp
		})
	}

	return rpc_router
}
