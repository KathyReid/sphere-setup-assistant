package main

import (
	"github.com/paypal/gatt"
	"log"
	"encoding/json"
	"github.com/theojulienne/go-wireless/iwlib"
	"os"
	"os/exec"
	"io"
	"strings"
)

type WifiNetwork struct {
	SSID string `json:"name"`
}

type WifiCredentials struct {
	SSID string `json:"ssid"`
	Key string `json:"key"`
}

const WPASupplicantTemplate = `
ctrl_interface=/var/run/wpa_supplicant
update_config=1
p2p_disabled=1
 
network={
	ssid="{{ssid}}"
	scan_ssid=1
	psk="{{key}}"
	key_mgmt=WPA-PSK
}
`

const WLANInterfaceTemplate = `
auto wlan0
iface wlan0 inet dhcp
	pre-up /usr/local/sbin/wpa_supplicant -B -D nl80211 -i wlan0 -c /etc/wpa_supplicant.conf
	post-down /usr/bin/killall -q wpa_supplicant
`

func WriteToFile(filename string, contents string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.WriteString(f, contents)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// start by registering the RPC functions that will be accessible
	// once the client has authenticated
	rpc_router := JSONRPCRouter{}
	rpc_router.Init()
	rpc_router.AddHandler("sphere.setup.ping", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		pong := JSONRPCResponse{"2.0", request.Id, 1234, nil}
		resp <- pong

		return resp
	})
	rpc_router.AddHandler("sphere.setup.get_visible_wifi_networks", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)
		
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

		s := strings.Replace(WPASupplicantTemplate,"{{ssid}}",wifi_creds.SSID,-1)
		s = strings.Replace(s,"{{key}}",wifi_creds.Key,-1)

		go func() {
			WriteToFile("/etc/wpa_supplicant.conf", s)
			WriteToFile("/etc/network/interfaces.d/wlan0", WLANInterfaceTemplate)
			
			cmd := exec.Command("ifup", "wlan0")
			cmd.Start()
			cmd.Wait() // shit will break badly if this fails :/
			
			serial_number, err := exec.Command("/opt/ninjablocks/bin/sphere-serial").Output()
			if err != nil {
				// ow ow ow
			}
			
			pong := JSONRPCResponse{"2.0", request.Id, serial_number, nil}
			resp <- pong
		}()

		return resp
	})

	srv := &gatt.Server{Name: "ninjasphere"}

	auth_handler := new(OneTimeAuthHandler)
	auth_handler.Init("spheramid")

	RegisterSecuredRPCService(srv, rpc_router, auth_handler)

	// Start the server
	log.Println("Starting setup assistant...");
	log.Fatal(srv.AdvertiseAndServe())
}
