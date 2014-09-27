package main

import (
	"fmt"
)

type ConsolePairingUI struct {

}

func (ui *ConsolePairingUI) DisplayColorHint(color string) {
	fmt.Printf(" *** COLOR HINT: %s ***\n", color)
}

func (ui *ConsolePairingUI) DisplayPairingCode(code string) {
	fmt.Printf(" *** PAIRING CODE: %s ***\n", code)
}
