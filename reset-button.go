package main

import (
	"fmt"
	"github.com/ninjasphere/sphere-go-led-controller/model"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"
)

const (
	shortDelay         = time.Millisecond * time.Duration(100)
	longDelay          = time.Second
	resetUserDataPress = time.Second * time.Duration(3)
	resetRootPress     = time.Second * time.Duration(45)
	gracePeriod        = time.Second * time.Duration(3)
	factoryResetMagic  = 168
)

type resetButton struct {
	current  state
	mode     string
	callback func(m *model.ResetMode)
}

type state interface {
	onUp(r *resetButton) state
	onDown(r *resetButton) state
	onTick(r *resetButton, d time.Duration) state
	currentTicks() time.Duration
	delay() time.Duration
}

type baseState struct {
	ticks time.Duration
}

// initial state
type state0 struct {
	baseState
}

// after reset button pressed
type state1 struct {
	baseState
}

// after reset button pressed for resetUserDataPress seconds
type state2 struct {
	baseState
}

// after reset button pressed for resetRootPress seconds
type state3 struct {
	baseState
}

// grace state a down action in this state will clear the reset action
type state4 struct {
	baseState
}

func startResetMonitor(callback func(m *model.ResetMode)) {
	r := &resetButton{
		current:  &state0{},
		mode:     "none",
		callback: callback,
	}
	go r.run()
}

func (r *resetButton) run() {
	down := false
	for {
		if reset, err := os.Open("/sys/class/gpio/gpio20/value"); err != nil {
			logger.Warningf("failed to open /sys/class/gpio/gpio20/value - reset loop aborting: %v", err)
			return
		} else {
			if bytes, err := ioutil.ReadAll(reset); err != nil {
				logger.Warningf("aborting read loop - failed to read reset button: %v", err)
				return
			} else {
				contents := string(bytes)
				contents = strings.TrimSpace(contents)
				if contents == "0" {
					if !down {
						down = true
						r.onDown()
					} else {
						r.onTick(r.current.delay())
					}
				} else if contents == "1" {
					if down {
						down = false
						r.onUp()
					} else {
						r.onTick(r.current.delay())
					}
				} else {
					logger.Warningf("unexpected input from button %s", contents)
				}
			}

			reset.Close()
		}
		time.Sleep(r.current.delay())
	}
}

func (r *resetButton) onDown() {
	logger.Debugf("down received in %s", r.String())
	nextState := r.current.onDown(r)
	if nextState != nil {
		r.current = nextState
		logger.Debugf("moved to %s", r.String())
	}
}

func (r *resetButton) onUp() {
	logger.Debugf("up received in %s", r.String())
	nextState := r.current.onUp(r)
	if nextState != nil {
		r.current = nextState
		logger.Debugf("moved to %s", r.String())
	}
}

func (r *resetButton) onTick(ticks time.Duration) {
	if r.mode != "none" {
		logger.Debugf("tick received in %s", r.String())
	}
	nextState := r.current.onTick(r, ticks)
	if nextState != nil {
		r.current = nextState
		logger.Debugf("moved to %s", r.String())
	}
}

func (r *resetButton) String() string {
	return fmt.Sprintf("%s(%d) - %s\n", reflect.ValueOf(r.current).Type().Elem(), r.current.currentTicks()/time.Second, r.mode)
}

func (r *resetButton) commit() {
	if path, err := exec.LookPath("reset-helper.sh"); err != nil {
		logger.Warningf("could not find reset-helper.sh: %v", err)
	} else {
		if err := exec.Command(path, r.mode).Run(); err != nil {
			logger.Warningf("failed to launch reset-helper.sh: %v", err)
		}
	}
	if r.mode == "reset-root" {
		os.Exit(factoryResetMagic)
	}
}

func (r *resetButton) updateMode(mode string) {
	r.mode = mode
	r.callback(&model.ResetMode{
		Mode:     r.mode,
		Hold:     true,
		Duration: 0,
	})
}

func (r *resetButton) countDown(duration time.Duration) state {
	r.callback(&model.ResetMode{
		Mode:     r.mode,
		Hold:     false,
		Duration: duration,
	})

	return &state4{
		baseState: baseState{
			ticks: duration,
		},
	}
}

//

func (s *baseState) onDown(r *resetButton) state {
	return nil
}

func (s *baseState) onUp(r *resetButton) state {
	return nil
}

func (s *baseState) onTick(r *resetButton, ticks time.Duration) state {
	s.ticks += ticks
	return nil
}

func (s *baseState) currentTicks() time.Duration {
	return s.ticks
}

func (s *baseState) delay() time.Duration {
	return shortDelay
}

// state0 is the reset state

func (s *state0) onDown(r *resetButton) state {
	r.updateMode("reboot")
	return &state1{
		baseState: baseState{
			ticks: 0,
		},
	}
}

func (s *state0) delay() time.Duration {
	return longDelay
}

// simple reboot
func (s *state1) onUp(r *resetButton) state {
	return r.countDown(time.Second)
}

// after 5 ticks
func (s *state1) onTick(r *resetButton, ticks time.Duration) state {
	s.baseState.onTick(r, ticks)
	if s.baseState.ticks > resetUserDataPress {
		r.updateMode("reset-userdata")
		return &state2{
			baseState: baseState{
				ticks: s.baseState.ticks,
			},
		}
	}
	return nil
}

// reset user-data reboot
func (s *state2) onUp(r *resetButton) state {
	return r.countDown(gracePeriod)
}

func (s *state2) onTick(r *resetButton, ticks time.Duration) state {
	s.baseState.onTick(r, ticks)
	if s.baseState.ticks > resetRootPress {
		r.updateMode("reset-root")
		return &state3{
			baseState: baseState{
				ticks: s.baseState.ticks,
			},
		}
	}
	return nil
}

// reset root reboot
func (s *state3) onUp(r *resetButton) state {
	return r.countDown(gracePeriod)
}

// reset root reboot
func (s *state4) onDown(r *resetButton) state {
	r.updateMode("none")
	return &state0{}
}

// reset root reboot
func (s *state4) onTick(r *resetButton, ticks time.Duration) state {
	s.baseState.onTick(r, -ticks)
	if s.baseState.ticks < 0 {
		r.commit()
		return &state0{}
	}
	return nil
}
