package main

import "github.com/theojulienne/go-wireless/wpactl"

type WifiManager struct {
	Controller  *wpactl.WPAController
	stateChange []chan string
}

const (
	WifiStateDisconnected = "disconnected"
	WifiStateConnected    = "connected"
	WifiStateInvalidKey   = "invalid_key"
)

func NewWifiManager(iface string) (*WifiManager, error) {
	ctl, err := wpactl.NewController(iface)
	if err != nil {
		return nil, err
	}

	manager := &WifiManager{}
	manager.stateChange = make([]chan string, 0)
	manager.Controller = ctl

	go manager.eventLoop()

	return manager, nil
}

func (m *WifiManager) WatchState() chan string {
	ch := make(chan string, 128)

	m.stateChange = append(m.stateChange, ch)

	return ch
}

func (m *WifiManager) UnwatchState(target chan string) {
	for i, c := range m.stateChange {
		if c == target {
			m.stateChange[i] = nil
		}
	}
}

func (m *WifiManager) emitState(state string) {
	for _, ch := range m.stateChange {
		if ch != nil {
			ch <- state
		}
	}
}

func (m *WifiManager) eventLoop() {
	for {
		event := <-m.Controller.EventChannel
		logger.Debugf("process: %v", event)
		switch event.Name {
		case "CTRL-EVENT-DISCONNECTED":
			m.emitState(WifiStateDisconnected)
		case "CTRL-EVENT-CONNECTED":
			m.emitState(WifiStateConnected)
		case "CTRL-EVENT-SSID-TEMP-DISABLED":
			m.emitState(WifiStateInvalidKey)
		}
	}
}

func (m *WifiManager) Cleanup() {
	m.Controller.Cleanup()
}

func (m *WifiManager) WifiConfigured() (bool, error) {
	networks, err := m.Controller.ListNetworks()
	if err != nil {
		return false, nil
	}
	enabledNetworks := 0
	for _, network := range networks {
		result, _ := m.Controller.GetNetworkSetting(network.Id, "disabled")
		if result == "1" {
			continue
		}
		enabledNetworks++
	}
	return (enabledNetworks > 0), nil
}

func (m *WifiManager) DisableAllNetworks() error {
	networks, err := m.Controller.ListNetworks()
	if err != nil {
		return err
	}

	for _, network := range networks {
		m.Controller.DisableNetwork(network.Id)
	}

	return nil
}

func (m *WifiManager) AddStandardNetwork(ssid string, key string) error {
	i, err := m.Controller.AddNetwork()
	if err != nil {
		return err
	}
	// FIXME: handle errors for all of these!
	m.Controller.SetNetworkSettingString(i, "ssid", ssid)
	m.Controller.SetNetworkSettingString(i, "psk", key)
	m.Controller.SetNetworkSettingRaw(i, "scan_ssid", "1")
	m.Controller.SetNetworkSettingRaw(i, "key_mgmt", "WPA-PSK")
	m.Controller.SelectNetwork(i)
	m.Controller.SaveConfiguration()

	return nil
}
