package main

import (
	"log"
	"os/exec"
)

type InterfaceManager struct {
	iface    string
	cmd      *exec.Cmd
	cmdReady chan bool
}

func NewInterfaceManager(iface string) *InterfaceManager {
	im := &InterfaceManager{iface, nil, make(chan bool, 1)}

	im.cmdReady <- true // start out as if a process has exited

	return im
}

func (im *InterfaceManager) execCmd(cmd string) {
	if im.cmd != nil && im.cmd.Process != nil {
		im.cmd.Process.Kill()
	}

	<-im.cmdReady

	im.cmd = exec.Command(cmd)
	go func() {
		out, _ := im.cmd.CombinedOutput()
		log.Println("cmd returned with:", out)
		im.cmd = nil

		im.cmdReady <- true
	}()
}

func (im *InterfaceManager) Up() {
	im.execCmd("ifup " + im.iface)
}

func (im *InterfaceManager) Down() {
	im.execCmd("ifdown " + im.iface)
}
