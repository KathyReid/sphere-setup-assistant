package main

import (
	"io/ioutil"
	"log"
	"os/exec"
)

type AccessPointManager struct {
	NetworkInterface string
	HostapdJob       UpstartJob
	config           AssistantConfig
}

func NewAccessPointManager(config AssistantConfig) *AccessPointManager {
	manager := &AccessPointManager{}
	manager.NetworkInterface = "ap0"
	manager.HostapdJob = UpstartJob{"hostapd-ap0"}
	manager.config = config

	return manager
}

func (a *AccessPointManager) StartHostAP() {
	a.HostapdJob.Stop()
	a.HostapdJob.Start()
}

func (a *AccessPointManager) StopHostAP() {
	a.HostapdJob.Stop()
}

func (a *AccessPointManager) WriteAPConfig() {
	s := ""
	s += "interface=ap0\n"
	s += "driver=nl80211\n"
	s += "ssid=" + a.config.Wireless_Host.SSID + "\n"
	s += "hw_mode=g\n"
	s += "channel=6\n"
	s += "macaddr_acl=0\n"
	s += "auth_algs=1\n"
	s += "ignore_broadcast_ssid=0\n"
	s += "wpa=3\n"
	s += "wpa_passphrase=" + a.config.Wireless_Host.Key + "\n"
	s += "wpa_key_mgmt=WPA-PSK\n"
	s += "wpa_pairwise=TKIP\n"
	s += "rsn_pairwise=CCMP\n"
	ioutil.WriteFile("/etc/hostapd-ap0.conf", []byte(s), 0600)
}

func (a *AccessPointManager) iptables(cmd ...string) {
	log.Println("iptables ", cmd)
	err := exec.Command("/sbin/iptables", cmd...).Run()
	if err != nil {
		log.Fatal("iptables failed: ", err)
	}
}

func (a *AccessPointManager) SetupFirewall() {
	if a.config.Wireless_Host.Full_Network_Access {
		a.iptables("-F")
	} else {
		a.iptables("-P", "INPUT", "ACCEPT")
		a.iptables("-P", "OUTPUT", "ACCEPT")
		a.iptables("-P", "FORWARD", "ACCEPT")
		a.iptables("-F")

		// allow nothing on here. before doing this, we will eventually allow setup access on one port.
		a.iptables("-A", "INPUT", "-i", a.NetworkInterface, "-j", "DROP")
	}
}
