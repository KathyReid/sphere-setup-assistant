package main

import (
	"encoding/json"
	"fmt"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/ninjasphere/go-ninja/config"
)

type ConsolePairingUI struct {
	client *mqtt.MqttClient
	serial string
}

func NewConsolePairingUI() (*ConsolePairingUI, error) {

	mqttURL := fmt.Sprintf("tcp://%s:%d", config.MustString("mqtt", "host"), config.MustInt("mqtt", "port"))

	opts := mqtt.NewClientOptions().AddBroker(mqttURL).SetClientId("sphere-setup-assistant").SetCleanSession(true)
	client := mqtt.NewClient(opts)

	if _, err := client.Start(); err != nil {
		return nil, err
	}

	serial := config.Serial()

	return &ConsolePairingUI{
		client: client,
		serial: serial,
	}, nil
}

func (ui *ConsolePairingUI) DisplayColorHint(color string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"color":"#FF0000"}],"jsonrpc": "2.0","method":"displayColor","time":132123123}' -t '$node/:node/led-controller'

	err := ui.sendRpcRequest("displayColor", map[string]string{
		"color": color,
	})

	if err != nil {
		return err
	}

	fmt.Printf(" *** COLOR HINT: %s ***\n", color)
	return nil

}

func (ui *ConsolePairingUI) DisplayPairingCode(code string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"code":"1234"}],"jsonrpc": "2.0","method":"displayPairingCode","time":132123123}' -t '$node/:node/led-controller'

	//

	err := ui.sendRpcRequest("displayPairingCode", map[string]string{
		"code": code,
	})

	if err != nil {
		return err
	}

	fmt.Printf(" *** PAIRING CODE: %s ***\n", code)

	return nil
}

func (ui *ConsolePairingUI) sendRpcRequest(method string, payload map[string]string) error {

	topic := fmt.Sprintf("$node/%s/led-controller", ui.serial)

	data := JSONRPCRequest{"2.0", string(time.Now().Unix()), method, []interface{}{payload}}

	msg, err := json.Marshal(&data)

	if err != nil {
		return err
	}

	ui.client.Publish(0, topic, msg)

	return nil

}
