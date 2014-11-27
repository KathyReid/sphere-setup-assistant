package main

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/ninjasphere/sphere-go-led-controller/model"
	"github.com/ninjasphere/sphere-go-led-controller/ui"
	"github.com/ninjasphere/sphere-go-led-controller/util"
)

// Implements the pairing UI with a direct connection to the LED matrix.
type directPairingUI struct {
	layout *ui.PairingLayout
}

func newDirectPairingUI() (*directPairingUI, error) {

	pairingUI := &directPairingUI{
		layout: ui.NewPairingLayout(),
	}

	led, err := util.GetLEDConnection()

	if err != nil {
		log.Fatalf("Failed to get connection to LED matrix: %s", err)
	}

	go func() {

		s, err := util.GetLEDConnection()

		if err != nil {
			log.Fatalf("Failed to get connection to LED matrix: %s", err)
		}

		// Send a blank image to the led matrix
		util.WriteLEDMatrix(image.NewRGBA(image.Rect(0, 0, 16, 16)), s)

		// Main drawing loop
		for {
			image, err := pairingUI.layout.Render()
			if err != nil {
				log.Fatalf("Unable to render to led: %s", err)
			}
			util.WriteLEDMatrix(image, led)
		}

	}()

	return pairingUI, nil
}

func (u *directPairingUI) DisplayColorHint(color string) error {
	fmt.Fprintf(os.Stderr, "color hint %s\n", color)
	return u.DisplayColorHint(color)
}

func (u *directPairingUI) DisplayPairingCode(code string) error {
	fmt.Fprintf(os.Stderr, "pairing code %d\n", code)
	return u.DisplayPairingCode(code)
}

func (u *directPairingUI) EnableControl() error {
	return fmt.Errorf("Control is not available in reset mode.")
}

func (u *directPairingUI) DisplayIcon(icon string) error {
	return u.DisplayIcon(icon)
}

func (u *directPairingUI) DisplayResetMode(m *model.ResetMode) error {
	return u.DisplayResetMode(m)
}
