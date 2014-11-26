package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/sphere-go-led-controller/model"
)

const (
	rpcStateIdle             = 0
	rpcStateAwaitingResponse = 1
)

// ControlChecker sends a heartbeat led controller to ensure it is in control mode
type ControlChecker struct {
	sync.Mutex
	pairingUI ConsolePairingUI
	ticker    *time.Ticker
	state     int
}

func NewControlChecker(pairingUI ConsolePairingUI) *ControlChecker {
	return &ControlChecker{pairingUI: pairingUI}
}

func (c *ControlChecker) StartHeartbeat() {
	c.Lock()
	defer c.Unlock()

	c.ticker = time.NewTicker(time.Second * 5)

	go func() {
		for t := range c.ticker.C {
			logger.Debugf("Sending heartbeat at %s", t)
			c.enableControl()
		}
	}()

}

func (c *ControlChecker) StopHeartbeat() bool {
	c.Lock()
	defer c.Unlock()

	if c.ticker != nil {
		c.ticker.Stop()
		return true

		// TODO check if ticker was running and send DisableControl
	} else {
		return false
	}

}

func (c *ControlChecker) enableControl() {

	// are we in call
	if c.state == rpcStateAwaitingResponse {
		// if so skip this request
		return
	}

	// otherwise preseed
	c.state = rpcStateAwaitingResponse
	defer func() {
		logger.Debugf("reset heartbeat state to IDLE")
		c.state = rpcStateIdle
	}()

	err := c.pairingUI.EnableControl()

	if err != nil {
		logger.Errorf("Failed to send enable %s", err)
	}
	logger.Debugf("Heartbeat complete")

}

// ConsolePairingUI proxy interface to the led controller

type ConsolePairingUI interface {
	DisplayColorHint(color string) error
	DisplayPairingCode(code string) error
	EnableControl() error
	DisplayIcon(icon string) error
	DisplayResetMode(m *model.ResetMode) error
}

type consolePairingUI struct {
	conn   *ninja.Connection
	serial string
}

type dummyConsolePairingUI struct {
}

func (*dummyConsolePairingUI) DisplayColorHint(color string) error {
	return nil
}

func (*dummyConsolePairingUI) DisplayPairingCode(code string) error {
	return nil
}

func (*dummyConsolePairingUI) EnableControl() error {
	return nil
}

func (*dummyConsolePairingUI) DisplayIcon(icon string) error {
	return nil
}

func (*dummyConsolePairingUI) DisplayResetMode(m *model.ResetMode) error {
	return nil
}

// NewConsolePairingUI build a new console pairing ui
func NewConsolePairingUI() (ConsolePairingUI, error) {

	if factoryReset {
		return &dummyConsolePairingUI{}, nil
	} else {

		conn, err := ninja.Connect("sphere-setup-assistant")

		if err != nil {
			log.Fatalf("Failed to connect to mqtt: %s", err)
		}

		return &consolePairingUI{
			conn:   conn,
			serial: config.Serial(),
		}, nil
	}
}

// DisplayPairingCode makes an rpc call to the led-controller for displaying a color hint
func (ui *consolePairingUI) DisplayColorHint(color string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"color":"#FF0000"}],"jsonrpc": "2.0","method":"displayColor","time":132123123}' -t '$node/:node/led-controller'

	logger.Debugf(" *** COLOR HINT: %s ***", color)

	err := ui.sendRpcRequest("displayColor", map[string]string{
		"color": color,
	})

	if err != nil {
		return err
	}

	return nil

}

// DisplayPairingCode makes an rpc call to the led-controller for displaying the pairing code
func (ui *consolePairingUI) DisplayPairingCode(code string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"code":"1234"}],"jsonrpc": "2.0","method":"displayPairingCode","time":132123123}' -t '$node/:node/led-controller'

	logger.Debugf(" *** PAIRING CODE: %s ***", code)

	err := ui.sendRpcRequest("displayPairingCode", map[string]string{
		"code": code,
	})

	if err != nil {
		return err
	}

	return nil
}

// EnableControl once paired we need to led-controller to enable control
func (ui *consolePairingUI) EnableControl() error {
	// mosquitto_pub -m '{"id":123, "params": [],"jsonrpc": "2.0","method":"enableControl","time":132123123}' -t '$node/:node/led-controller'

	err := ui.sendRpcRequest("enableControl", make(map[string]string))

	if err != nil {
		return err
	}

	logger.Debugf(" *** ENABLE CONTROL***")

	return nil
}

// DisplayIcon
func (ui *consolePairingUI) DisplayIcon(icon string) error {
	// mosquitto_pub -m '{"id":123, "params": [{"icon":"weather.png"}],"jsonrpc": "2.0","method":"displayIcon","time":132123123}' -t '$node/SLC6M6GIPGQAK/led-controller'

	logger.Debugf(" *** DISPLAY ICON: %s ***", icon)

	err := ui.sendRpcRequest("displayIcon", map[string]string{
		"icon": icon,
	})

	if err != nil {
		return err
	}

	return nil
}

func (ui *consolePairingUI) DisplayResetMode(m *model.ResetMode) error {

	logger.Debugf(" *** DISPLAY RESET MODE: %v ***", m)

	err := ui.sendMarshaledRpcRequest("displayResetMode", m)

	if err != nil {
		return err
	}

	return nil
}

func (ui *consolePairingUI) sendRpcRequest(method string, payload map[string]string) error {
	topic := fmt.Sprintf("$node/%s/led-controller", ui.serial)
	return ui.conn.GetServiceClient(topic).Call(method, []interface{}{payload}, nil, 15*time.Second)
}

func (ui *consolePairingUI) sendMarshaledRpcRequest(method string, payload interface{}) error {
	topic := fmt.Sprintf("$node/%s/led-controller", ui.serial)
	return ui.conn.GetServiceClient(topic).Call(method, payload, nil, 15*time.Second)
}
