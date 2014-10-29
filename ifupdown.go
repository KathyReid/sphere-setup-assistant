package main

import "os/exec"

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

func (im *InterfaceManager) execCmd(cmd string, arg string) {
	if im.cmd != nil && im.cmd.Process != nil {
		im.cmd.Process.Kill()
	}

	<-im.cmdReady

	im.cmd = exec.Command(cmd, arg)
	go func() {
		out, err := im.cmd.CombinedOutput()
		if err != nil {
			logger.Errorf("error occured running %s : %s", cmd, err)
		}
		logger.Debugf("cmd returned with: %s", string(out))
		im.cmd = nil

		im.cmdReady <- true
	}()
}

func (im *InterfaceManager) Up() {
	logger.Debugf("running ifup")
	im.execCmd("/sbin/ifup", im.iface)
}

func (im *InterfaceManager) Down() {
	logger.Debugf("running ifdown")
	im.execCmd("/sbin/ifdown", im.iface)
}
