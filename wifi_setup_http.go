package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/ninjasphere/gatt"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-wireless/iwlib"
)

func isPaired() bool {
	return config.HasString("siteId") && config.HasString("token") && config.HasString("userId") && config.HasString("nodeId")
}

func StartHTTPServer(conn *ninja.Connection, wifi_manager *WifiManager, srv *gatt.Server, pairing_ui ConsolePairingUI) {

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {

		config.MustRefresh()

		data := map[string]interface{}{
			"paired": isPaired(),
			"nodeId": config.MustString("nodeId"),
		}

		if ip, ipErr := GetWlanAddress(); ipErr == nil {
			data["wlanIp"] = ip
		}

		if isPaired() {
			data["siteId"] = config.MustString("siteId")
			data["userId"] = config.MustString("userId")
		}

		out, _ := json.Marshal(data)

		io.WriteString(w, string(out))
	})

	http.HandleFunc("/get_visible_wifi_networks", func(w http.ResponseWriter, r *http.Request) {

		pairing_ui.DisplayIcon("wifi-searching.gif")

		// Before we search for wifi networks, disable any that are try-fail-ing
		wifi_manager.DisableAllNetworks()

		networks, err := iwlib.GetWirelessNetworks("wlan0")

		if err == nil {
			wifi_networks := make([]WifiNetwork, len(networks))
			for i, network := range networks {
				wifi_networks[i].SSID = network.SSID
			}

			out, err := json.Marshal(wifi_networks)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.WriteString(w, string(out))
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	})

	http.HandleFunc("/connect_wifi_network", func(w http.ResponseWriter, r *http.Request) {

		pairing_ui.DisplayIcon("wifi-connecting.gif")

		body, err := ioutil.ReadAll(r.Body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var wifi_creds WifiCredentials

		json.Unmarshal(body, &wifi_creds)

		logger.Infof("Got wifi credentials %v", wifi_creds)

		done := make(chan bool, 2)

		go func() {
			done <- wifi_manager.SetCredentials(&wifi_creds)
		}()

		select {
		case success := <-done:

			logger.Infof("Wifi success? %t", success)

			if success {
				pairing_ui.DisplayIcon("wifi-connected.gif")
				serial_number, err := exec.Command("/opt/ninjablocks/bin/sphere-serial").Output()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				io.WriteString(w, "\""+string(serial_number)+"\"")
			} else {
				pairing_ui.DisplayIcon("wifi-failed.gif")
				http.Error(w, "Could not connect to specified WiFi network, is the key correct?", http.StatusBadRequest)
			}

		case <-time.After(time.Second * 15):
			pairing_ui.DisplayIcon("wifi-failed.gif")
			http.Error(w, "Could not connect to specified WiFi network, is it in range?", http.StatusBadRequest)
		}
	})

	http.HandleFunc("/close_ble_central", func(w http.ResponseWriter, r *http.Request) {
		err := srv.Close()

		if err == nil {
			io.WriteString(w, "true")
		} else {
			io.WriteString(w, "false")
		}
	})

	if !factoryReset {

		updateService := conn.GetServiceClient("$node/" + config.Serial() + "/updates")

		updateService.OnEvent("progress", func(progress *map[string]interface{}, topicKeys map[string]string) bool {
			lastUpdateProgress = *progress

			logger.Infof("Got update progress: %v", *progress)
			return true
		})

		http.HandleFunc("/start_update", func(w http.ResponseWriter, r *http.Request) {
			var response bool

			err := updateService.Call("start", nil, &response, time.Second*10)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if response {
				lastUpdateProgress = nil
			}

			out, err := json.Marshal(&response)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.WriteString(w, string(out))
		})

		http.HandleFunc("/get_update_progress", func(w http.ResponseWriter, r *http.Request) {

			out, err := json.Marshal(&lastUpdateProgress)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.WriteString(w, string(out))
		})

		http.HandleFunc("/get_wifi_ip", func(w http.ResponseWriter, r *http.Request) {

			ip, err := GetWlanAddress()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			out, err := json.Marshal(&ip)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			io.WriteString(w, string(out))
		})

	}

	go func() {
		logger.Infof("Starting http server on port 8888")
		logger.Fatalf("Web server failed: %s", http.ListenAndServe(":8888", nil))
	}()

}
