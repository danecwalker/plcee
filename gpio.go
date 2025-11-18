package main

import (
	"fmt"
	"log"
	"math"
)

// setupPins initializes all GPIO pins and returns a Pins struct
func setupPins(adc *ADS1115) (*Pins, error) {
	var pins Pins

	// Log currently exported GPIO pins for diagnostics
	if exported, err := ListExportedGPIOs(); err == nil {
		log.Printf("Currently exported GPIO pins: %v", exported)
	} else {
		log.Printf("warning: failed to list exported GPIO pins: %v", err)
	}

	// Configure input pins with pull-up resistors (pull-up handled by hardware or OS)
	var err error
	pins.FootSwitch, err = NewGPIOPin(13, "in")
	if err != nil {
		return nil, fmt.Errorf("failed to configure FootSwitch pin (offset 13): %w", err)
	}
	log.Printf("FootSwitch pin: offset 13")

	pins.EStop, err = NewGPIOPin(17, "in")
	if err != nil {
		return nil, fmt.Errorf("failed to configure EStop pin (offset 17): %w", err)
	}
	log.Printf("EStop pin: offset 17")

	pins.ProxInput, err = NewGPIOPin(22, "in")
	if err != nil {
		return nil, fmt.Errorf("failed to configure ProxInput pin (offset 22): %w", err)
	}
	log.Printf("ProxInput pin: offset 22")

	// Configure output pins
	pins.DumpValve, err = NewGPIOPin(23, "out")
	if err != nil {
		return nil, fmt.Errorf("failed to configure DumpValve pin (offset 23): %w", err)
	}
	if err := pins.DumpValve.Out(Low); err != nil {
		return nil, fmt.Errorf("failed to set DumpValve initial level: %w", err)
	}
	log.Printf("DumpValve pin: offset 23")

	pins.Speed, err = NewGPIOPin(24, "out")
	if err != nil {
		return nil, fmt.Errorf("failed to configure Speed pin (offset 24): %w", err)
	}
	if err := pins.Speed.Out(Low); err != nil {
		return nil, fmt.Errorf("failed to set Speed initial level: %w", err)
	}
	log.Printf("Speed pin: offset 24")

	pins.Buzz, err = NewGPIOPin(19, "out")
	if err != nil {
		return nil, fmt.Errorf("failed to configure Buzz pin (offset 19): %w", err)
	}
	if err := pins.Buzz.Out(Low); err != nil {
		return nil, fmt.Errorf("failed to set Buzz initial level: %w", err)
	}
	log.Printf("Buzz pin: offset 19")

	pins.ADC = adc
	pins.ADCChan = 0 // Load sensor on channel 0

	log.Println("GPIO pins configured successfully")
	return &pins, nil
}

// closePins closes all GPIO pins and ADC resources
func closePins(pins *Pins) {
	if pins == nil {
		return
	}

	if pins.FootSwitch != nil {
		pins.FootSwitch.Halt()
	}
	if pins.EStop != nil {
		pins.EStop.Halt()
	}
	if pins.ProxInput != nil {
		pins.ProxInput.Halt()
	}
	if pins.DumpValve != nil {
		pins.DumpValve.Halt()
	}
	if pins.Speed != nil {
		pins.Speed.Halt()
	}
	if pins.Buzz != nil {
		pins.Buzz.Halt()
	}
	if pins.ADC != nil {
		if err := pins.ADC.Close(); err != nil {
			log.Printf("warning: failed to close ADC: %v", err)
		}
	}

	log.Println("GPIO pins and ADC closed successfully")
}

// read updates the state from GPIO pin values
func (s *State) read(pins *Pins) {
	mu.Lock()
	defer mu.Unlock()

	footSwitchLevel := pins.FootSwitch.Read()
	estopLevel := pins.EStop.Read()
	proxLevel := pins.ProxInput.Read()

	s.FootSwitch = footSwitchLevel == High
	s.EStop = estopLevel == Low
	s.ProxInput = proxLevel == High

	sample, err := pins.ADC.ReadVoltage(pins.ADCChan)
	if err != nil {
		log.Printf("error reading ADC load sensor: %v", err)
	} else {
		s.Load = math.Floor((sample*2.5)*1000) / 1000
	}
}

// write outputs the state to GPIO pins
func (s *State) write(pins *Pins) {
	mu.RLock()
	defer mu.RUnlock()

	if s.DumpValve {
		pins.DumpValve.Write(High)
	} else {
		pins.DumpValve.Write(Low)
	}

	if s.Speed {
		pins.Speed.Write(High)
	} else {
		pins.Speed.Write(Low)
	}

	if s.Buzz {
		pins.Buzz.Write(High)
		s.Buzz = false
	} else {
		pins.Buzz.Write(Low)
	}
}
