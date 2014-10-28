package main

import (
	"os/exec"

	"code.google.com/p/gcfg"
)

type AssistantConfig struct {
	Wireless_Host struct {
		SSID                string
		Key                 string
		Full_Network_Access bool
		Always_Active       bool
		Enables_Control     bool
	}
}

func LoadConfig(path string) AssistantConfig {
	var cfg AssistantConfig

	// defaults
	uniqueSuffix, _ := exec.Command("/bin/sh", "-c", "/opt/ninjablocks/bin/sphere-go-serial | sha256sum | cut -c1-8").Output()
	cfg.Wireless_Host.SSID = "NinjaSphere-" + string(uniqueSuffix)
	cfg.Wireless_Host.Key = "ninjasphere"
	cfg.Wireless_Host.Full_Network_Access = false
	cfg.Wireless_Host.Always_Active = false
	cfg.Wireless_Host.Enables_Control = false

	// load from config file (optionally)
	gcfg.ReadFileInto(&cfg, path)

	return cfg
}
