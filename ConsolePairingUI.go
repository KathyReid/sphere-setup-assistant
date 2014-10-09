package main

import (
	"fmt"
)

type ConsolePairingUI struct {

}

func (ui *ConsolePairingUI) DisplayColorHint(color string) {
	// mosquitto_pub -m '{"id":123, "params": [{"color":"#FF0000"}],"jsonrpc": "2.0","method":"displayColor","time":132123123}' -t '$node/:node/led-controller'
	fmt.Printf(" *** COLOR HINT: %s ***\n", color)
}

func (ui *ConsolePairingUI) DisplayPairingCode(code string) {
	// mosquitto_pub -m '{"id":123, "params": [{"code":"1234"}],"jsonrpc": "2.0","method":"displayPairingCode","time":132123123}' -t '$node/:node/led-controller'
	fmt.Printf(" *** PAIRING CODE: %s ***\n", code)
}
