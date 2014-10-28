package main

import (
	"os/exec"
	"strings"
)

type UpstartJob struct {
	Name string
}

func (j *UpstartJob) Status() (string, error) {
	out, err := exec.Command("/sbin/status", j.Name).Output()
	if err != nil {
		parts := strings.Split(string(out), " ")
		return parts[1], nil
	} else {
		return "", err
	}
}

func (j *UpstartJob) Running() (bool, error) {
	status, err := j.Status()
	if err != nil {
		running := (status == "start/running")
		return running, nil
	} else {
		return false, err
	}
}

func (j *UpstartJob) Start() {
	logger.Debugf("Starting upstart job: ", j.Name)
	data, err := exec.Command("/sbin/start", j.Name).Output()
	logger.Debugf("exec result: %s err: %s", string(data), err)
}

func (j *UpstartJob) Stop() {
	logger.Debugf("Stopping upstart job: ", j.Name)
	data, err := exec.Command("/sbin/stop", j.Name).Output()
	logger.Debugf("exec result: %s err: %s", string(data), err)
}
