package main

import (
        "os/exec"
        "strings"
)

type UpstartJob struct {
        Name string
}

func (j *UpstartJob) Status() (string, error) {
        out, err := exec.Command("status " + j.Name).Output()
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
        exec.Command("start " + j.Name).Run()
}

func (j *UpstartJob) Stop() {
        exec.Command("stop " + j.Name).Run()
}
