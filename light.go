package garagepi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (e Executor) handleLightGet(w http.ResponseWriter, r *http.Request) {
	args := []string{GpioReadCommand, tostr(e.gpioLightPin)}

	e.logger.Log("Reading light state")
	discovered, err := e.executeCommand(e.gpioExecutable, args...)
	discovered = strings.TrimSpace(discovered)
	if err != nil {
		e.logger.Log(fmt.Sprintf("Error executing: '%s %s' - light state unknown", e.gpioExecutable, strings.Join(args, " ")))
		w.Write([]byte("error - light state: unknown"))
	} else {
		state, err := stateNumberToOnOffString(discovered)
		if err != nil {
			e.logger.Log(fmt.Sprintf("Error reading light state: %v", err))
			w.Write([]byte("error - light state: unknown"))
		} else {
			e.logger.Log(fmt.Sprintf("Light state: %s", state))
			w.Write([]byte(fmt.Sprintf("light state: %s", state)))
		}
	}
}

func stateNumberToOnOffString(number string) (string, error) {
	switch number {
	case GpioLowState:
		return "off", nil
	case GpioHighState:
		return "on", nil
	default:
		return "", errors.New(fmt.Sprintf("Unrecognized state: %s", number))
	}
}

func onOffStringToStateNumber(onOff string) (string, error) {
	switch onOff {
	case "on":
		return GpioHighState, nil
	case "off":
		return GpioLowState, nil
	default:
		return "", errors.New(fmt.Sprintf("Unrecognized state: %s", onOff))
	}
}

func (e Executor) handleLightState(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		e.logger.Log("Error parsing form - assuming light should be turned on.")
		e.turnLightOn(w)
		return
	}

	state := r.Form.Get("state")

	if state == "" {
		e.logger.Log("No state provided - assuming light should be turned on.")
		e.turnLightOn(w)
		return
	}

	gpioState, err := onOffStringToStateNumber(state)
	if err != nil {
		e.logger.Log(fmt.Sprintf("Invalid state provided (%s) - assuming light should be turned on.", state))
		e.turnLightOn(w)
		return
	}

	switch gpioState {
	case GpioLowState:
		e.turnLightOff(w)
		return
	case GpioHighState:
		e.turnLightOn(w)
		return
	}
}

func (e Executor) turnLightOn(w http.ResponseWriter) {
	e.setLightState(w, true)
}

func (e Executor) turnLightOff(w http.ResponseWriter) {
	e.setLightState(w, false)
}

func (e Executor) setLightState(w http.ResponseWriter, stateOn bool) {
	var state string
	var args []string
	if stateOn {
		state = "on"
		args = []string{GpioWriteCommand, tostr(e.gpioLightPin), GpioHighState}
	} else {
		state = "off"
		args = []string{GpioWriteCommand, tostr(e.gpioLightPin), GpioLowState}
	}

	e.logger.Log(fmt.Sprintf("Setting light state to %s", state))
	_, err := e.executeCommand(e.gpioExecutable, args...)
	if err != nil {
		e.logger.Log(fmt.Sprintf("Error executing: '%s %s'", e.gpioExecutable, strings.Join(args, " ")))
		w.Write([]byte("error - light state unchanged"))
	} else {
		e.logger.Log(fmt.Sprintf("Light state: %s", state))
		w.Write([]byte(fmt.Sprintf("light state: %s", state)))
	}
}
