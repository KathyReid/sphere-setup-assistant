package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/elliots/go-wireless/iwlib"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
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

func GetSetupRPCRouter(wifi_manager *WifiManager, srv *gatt.Server, pairing_ui ConsolePairingUI) *JSONRPCRouter {

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
			WriteToFile("/etc/network/interfaces.d/wlan0", WLANInterfaceTemplate)

			states := wifi_manager.WatchState()

			wifi_manager.AddStandardNetwork(wifi_creds.SSID, wifi_creds.Key)
			wifi_manager.Controller.ReloadConfiguration()

			success := true
			for {
				state := <-states
				if state == WifiStateConnected {
					pairing_ui.DisplayIcon("wifi-connected.gif")
					success = true
					break
				} else if state == WifiStateInvalidKey {
					pairing_ui.DisplayIcon("wifi-failed.gif")
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

	conn, err := ninja.Connect("sphere-setup-assistant-updates")

	if err != nil {
		logger.FatalErrorf(err, "Failed to connect to mqtt")
	}

	updateService := conn.GetServiceClient("$node/" + config.Serial() + "/updates")
	ledService := conn.GetServiceClient("$node/" + config.Serial() + "/led-controller")

	rpc_router.AddHandler("sphere.setup.start_update", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		var response json.RawMessage

		logger.Infof("Starting update id: %d. Waiting for response....", request.Id)

		err := updateService.Call("start", nil, &response, time.Second*10)

		logger.Infof("Got update start response: %d. %v", request.Id, response)

		if err == nil {
			resp <- JSONRPCResponse{"2.0", request.Id, &response, nil}
		} else {
			resp <- JSONRPCResponse{"2.0", request.Id, nil, &JSONRPCError{500, fmt.Sprintf("%s", err), nil}}
		}

		return resp
	})

	var lastProgress map[string]interface{}

	updateService.OnEvent("progress", func(progress *map[string]interface{}, topicKeys map[string]string) bool {
		lastProgress = *progress

		logger.Infof("Got update progress: %v", *progress)
		return true
	})

	rpc_router.AddHandler("sphere.setup.get_update_progress", func(request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		logger.Infof("Requesting update progress id:%s", request.Id)

		resp <- JSONRPCResponse{"2.0", request.Id, lastProgress, nil}

		logger.Infof("Sent progress %v", lastProgress)

		running, ok := lastProgress["running"]

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

	return rpc_router
}
