package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
)

type ConsolePairingUI struct {
	conn   *ninja.Connection
	serial string
}

func NewConsolePairingUI() (*ConsolePairingUI, error) {

	conn, err := ninja.Connect("sphere-setup-assistant")

	if err != nil {
		log.Fatalf("Failed to connect to mqtt: %s", err)
	}

	return &ConsolePairingUI{
		conn:   conn,
		serial: config.Serial(),
	}, nil
}

// DisplayPairingCode makes an rpc call to the led-controller for displaying a color hint
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

// DisplayPairingCode makes an rpc call to the led-controller for displaying the pairing code
func (ui *ConsolePairingUI) DisplayPairingCode(code string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"code":"1234"}],"jsonrpc": "2.0","method":"displayPairingCode","time":132123123}' -t '$node/:node/led-controller'

	err := ui.sendRpcRequest("displayPairingCode", map[string]string{
		"code": code,
	})

	if err != nil {
		return err
	}

	fmt.Printf(" *** PAIRING CODE: %s ***\n", code)

	return nil
}

// EnableControl once paired we need to led-controller to enable control
func (ui *ConsolePairingUI) EnableControl() error {
	// mosquitto_pub -m '{"id":123, "params": [],"jsonrpc": "2.0","method":"enableControl","time":132123123}' -t '$node/:node/led-controller'

	err := ui.sendRpcRequest("enableControl", make(map[string]string))

	if err != nil {
		return err
	}

	fmt.Printf(" *** ENABLE CONTROL***\n")

	return nil
}

// DisplayIcon
func (ui *ConsolePairingUI) DisplayIcon(icon string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"icon":"weather.png"}],"jsonrpc": "2.0","method":"displayIcon","time":132123123}' -t '$node/SLC6M6GIPGQAK/led-controller'

	err := ui.sendRpcRequest("displayIcon", map[string]string{
		"icon": icon,
	})

	if err != nil {
		return err
	}

	fmt.Printf(" *** DISPLAY ICON: %s ***\n", icon)

	return nil
}
func (ui *ConsolePairingUI) sendRpcRequest(method string, payload map[string]string) error {
	topic := fmt.Sprintf("$node/%s/led-controller", ui.serial)
	return ui.conn.GetServiceClient(topic).Call(method, []interface{}{payload}, nil, 15*time.Second)
}
