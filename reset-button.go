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

// There are now 5 states: rest, select, grace, commit and abort.

// The device is in the rest state until the button is depressed. It then
// moves to the select state.

// In the select state, the LED cycles between the following modes:

// - halt (grey)
// - reboot (green)
// - reset-userdata (yellow)
// - reset-root (red)
// - abort (white)

// When the user releases the button, the device enters the grace state for the specified mode unless the selection
// was abort, in which case it enters the abort state.

// In the grace state, the color fades. During this time, if the user
// presses the button again, the device moves into the abort state. Otherwise,
// the device proceeds to the commit state and the action selected action
// is taken.

// In the abort state, the color fades from white to black and then the device returns to the rest state.

const (
	shortDelay        = time.Millisecond * time.Duration(100)
	selectionDelay    = time.Second * time.Duration(3)
	graceDelay        = time.Second * time.Duration(30)
	abortDelay        = time.Second * time.Duration(3)
	factoryResetMagic = 168
)

// the modes that we cycle between when we are in the 'select' state.

// the modes form a pendulum that swings from halt -> reboot -> reset-userdata ->reset-root -> abort and back again.

var (
	modeCycle = []string{"halt", "reboot", "reset-userdata", "reset-root", "abort", "reset-root", "reset-userdata", "reboot", "halt", "abort"}
)

// a resetButton is a state machine that listens to the reset button
type resetButton struct {
	current   state                    // the current state of the reset button controller
	modeIndex int                      // the currently selected mode
	callback  func(m *model.ResetMode) // the callback used to display the state of the controller to the user
	timeout   *time.Timer              // the timer for the current state
	ticks     *time.Timer              // the tick timer - we sample hardware button on these ticks
}

// a state of the resetButton state machine
type state interface {
	onEnter(r *resetButton)         // method called on entry to a new state.
	onUp(r *resetButton) state      // method called when the reset button is released
	onDown(r *resetButton) state    // method called when the down button is released
	onTimeout(r *resetButton) state // method called when the timeout expires.
}

// start a new reset button monitor
func startResetMonitor(callback func(m *model.ResetMode)) {
	r := &resetButton{
		current:   &stateRest{},
		modeIndex: 0,
		callback:  callback,
		timeout:   time.NewTimer(0),
		ticks:     time.NewTimer(shortDelay),
	}
	r.timeout.Stop()
	select {
	case _ = <-r.timeout.C:
	default:
	}
	go r.run()
}

// transition the receiver to the new state, if the new state is not nil.
func (r *resetButton) transition(s state) {
	if s != nil {
		r.current = s
		r.timeout.Stop()
		select {
		case _ = <-r.timeout.C:
		default:
		}
		s.onEnter(r)
	}
}

// run the state machine
func (r *resetButton) run() {
	down := false
	r.transition(r.current)
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
						r.transition(r.current.onDown(r))
					}
				} else if contents == "1" {
					if down {
						down = false
						r.transition(r.current.onUp(r))
					}
				} else {
					logger.Warningf("unexpected input from button %s", contents)
				}
			}

			reset.Close()
		}

		select {
		case _ = <-r.ticks.C:
			r.ticks.Reset(shortDelay)
		case _ = <-r.timeout.C:
			r.transition(r.current.onTimeout(r))
		}
	}
}

// describe the state machine
func (r *resetButton) String() string {
	return fmt.Sprintf("%s - %s\n", reflect.ValueOf(r.current).Type().Elem(), modeCycle[r.modeIndex])
}

// commit the currently selected mode
func (r *resetButton) commit() {
	if path, err := exec.LookPath("reset-helper.sh"); err != nil {
		logger.Warningf("could not find reset-helper.sh: %v", err)
	} else {
		if err := exec.Command(path, modeCycle[r.modeIndex]).Run(); err != nil {
			logger.Warningf("failed to launch reset-helper.sh: %v", err)
		}
	}
	if modeCycle[r.modeIndex] == "reset-root" {
		os.Exit(factoryResetMagic)
	}
}

//
// The baseState is a pseudo-state from which other states inherit the default
// implementation of the state interface.
//
type baseState struct {
}

func (s *baseState) onEnter(r *resetButton) {
}

func (s *baseState) onDown(r *resetButton) state {
	return nil
}

func (s *baseState) onUp(r *resetButton) state {
	return nil
}

func (s *baseState) onTimeout(r *resetButton) state {
	return nil
}

// stateRest

//
// This is the rest state for the reset button. In this state, we wait for
// the reset button to be pressed.
//
type stateRest struct {
	baseState
}

func (s *stateRest) onEnter(r *resetButton) {
	r.modeIndex = 0
	r.callback(&model.ResetMode{
		Mode:     "none",
		Hold:     true,
		Duration: 0,
	})
}

func (s *stateRest) onDown(r *resetButton) state {
	return &stateSelect{}
}

// stateSelect

//
// In this state we cycle between the different possible
// modes until the user releases the button, we then
// move into the grace state or, if the currently displayed
// mode is "abort", the abort state.
//

type stateSelect struct {
	baseState
}

func (s *stateSelect) onEnter(r *resetButton) {
	r.timeout.Reset(selectionDelay)
	r.callback(&model.ResetMode{
		Mode:     modeCycle[r.modeIndex],
		Hold:     true,
		Duration: selectionDelay,
	})
}

func (s *stateSelect) onUp(r *resetButton) state {
	if modeCycle[r.modeIndex] == "abort" {
		return &stateAbort{}
	} else {
		return &stateGrace{}
	}
}

func (s *stateSelect) onTimeout(r *resetButton) state {
	r.modeIndex += 1
	r.modeIndex = r.modeIndex % len(modeCycle)
	return &stateSelect{}
}

// stateGrace

//
// In this state, we give the user an opportunity to cancel the their
// selected action by reacting to down presses. If the user presses
// the button, we move into the abort state, otherwise we
// move to the commit state where the selected mode is committed.
//

type stateGrace struct {
	baseState
}

func (s *stateGrace) onEnter(r *resetButton) {
	r.timeout.Reset(graceDelay)
	r.callback(&model.ResetMode{
		Mode:     modeCycle[r.modeIndex],
		Hold:     false,
		Duration: graceDelay,
	})
}

func (s *stateGrace) onTimeout(r *resetButton) state {
	return &stateCommit{}
}

func (s *stateGrace) onDown(r *resetButton) state {
	return &stateAbort{}
}

// stateAbort

//
// In this state, we provide a visual indication to the user that
// their abort request was respected. In particular, we flash white, then fade.
//
type stateAbort struct {
	baseState
}

func (s *stateAbort) onEnter(r *resetButton) {
	r.timeout.Reset(abortDelay)
	r.callback(&model.ResetMode{
		Mode:     modeCycle[r.modeIndex],
		Hold:     true,
		Duration: abortDelay,
	})
}

func (s *stateAbort) onTimeout(r *resetButton) state {
	return &stateRest{}
}

// onEnter - commit the action.
type stateCommit struct {
	baseState
}

func (s *stateCommit) onEnter(r *resetButton) {
	r.commit()
}
