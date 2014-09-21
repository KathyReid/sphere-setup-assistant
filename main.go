package main

import (
	"github.com/paypal/gatt"
	"log"
	"encoding/json"
)

type WifiNetwork struct {
	SSID string `json:"name"`
}

type WifiCredentials struct {
	SSID string `json:"ssid"`
	Key string `json:"key"`
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

		wifi_networks := []WifiNetwork{
			{"SuperNinja"},
			{"MagicNet"},
		}

		pong := JSONRPCResponse{"2.0", request.Id, wifi_networks, nil}
		resp <- pong

		return resp
	})
	rpc_router.AddHandler("sphere.setup.connect_wifi_network", func (request JSONRPCRequest) chan JSONRPCResponse {
		resp := make(chan JSONRPCResponse, 1)

		wifi_creds := new(WifiCredentials)
		b, _ := json.Marshal(request.Params[0])
		json.Unmarshal(b, wifi_creds)

		log.Println("Got wifi credentials", wifi_creds)

		pong := JSONRPCResponse{"2.0", request.Id, 1, nil}
		resp <- pong

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