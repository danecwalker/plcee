package main

import (
	"log"
	"math"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/ads1x15"
)

// setupPins initializes all GPIO pins and returns a Pins struct
func setupPins(adc *ads1x15.Dev) *Pins {
	var pins Pins

	// Configure input pins with pull-up resistors
	pins.FootSwitch = gpioreg.ByName("13")
	if pins.FootSwitch == nil {
		log.Fatal("failed to open GPIO pin 13 (FootSwitch)")
	}
	log.Printf("FootSwitch pin: %s", pins.FootSwitch)
	if err := pins.FootSwitch.In(gpio.PullUp, gpio.NoEdge); err != nil {
		log.Printf("warning: failed to configure FootSwitch pull-up: %v", err)
		// Try without pull-up
		if err := pins.FootSwitch.In(gpio.Float, gpio.NoEdge); err != nil {
			log.Fatalf("failed to configure FootSwitch pin: %v", err)
		}
	}

	pins.EStop = gpioreg.ByName("17")
	if pins.EStop == nil {
		log.Fatal("failed to open GPIO pin 17 (EStop)")
	}
	log.Printf("EStop pin: %s", pins.EStop)
	if err := pins.EStop.In(gpio.PullUp, gpio.NoEdge); err != nil {
		log.Printf("warning: failed to configure EStop pull-up: %v", err)
		// Try without pull-up
		if err := pins.EStop.In(gpio.Float, gpio.NoEdge); err != nil {
			log.Fatalf("failed to configure EStop pin: %v", err)
		}
	}

	pins.ProxInput = gpioreg.ByName("22")
	if pins.ProxInput == nil {
		log.Fatal("failed to open GPIO pin 22 (ProxInput)")
	}
	log.Printf("ProxInput pin: %s", pins.ProxInput)
	if err := pins.ProxInput.In(gpio.PullUp, gpio.NoEdge); err != nil {
		log.Printf("warning: failed to configure ProxInput pull-up: %v", err)
		// Try without pull-up
		if err := pins.ProxInput.In(gpio.Float, gpio.NoEdge); err != nil {
			log.Fatalf("failed to configure ProxInput pin: %v", err)
		}
	}

	// Configure output pins
	pins.DumpValve = gpioreg.ByName("23")
	if pins.DumpValve == nil {
		log.Fatal("failed to open GPIO pin 23 (DumpValve)")
	}
	log.Printf("DumpValve pin: %s", pins.DumpValve)
	if err := pins.DumpValve.Out(gpio.Low); err != nil {
		log.Fatalf("failed to configure DumpValve pin: %v", err)
	}

	pins.Speed = gpioreg.ByName("24")
	if pins.Speed == nil {
		log.Fatal("failed to open GPIO pin 24 (Speed)")
	}
	log.Printf("Speed pin: %s", pins.Speed)
	if err := pins.Speed.Out(gpio.Low); err != nil {
		log.Fatalf("failed to configure Speed pin: %v", err)
	}

	pins.Buzz = gpioreg.ByName("19")
	if pins.Buzz == nil {
		log.Fatal("failed to open GPIO pin 19 (Buzz)")
	}
	log.Printf("Buzz pin: %s", pins.Buzz)
	if err := pins.Buzz.Out(gpio.Low); err != nil {
		log.Fatalf("failed to configure Buzz pin: %v", err)
	}

	pin, err := adc.PinForChannel(ads1x15.Channel0, 4096*physic.MilliVolt, 10*physic.Hertz, ads1x15.BestQuality)
	if err != nil {
		log.Fatalf("failed to configure ADC channel 0: %v", err)
	}
	pins.Load = pin

	log.Println("GPIO pins configured successfully")
	return &pins
}

// read updates the state from GPIO pin values
func (s *State) read(pins *Pins) {
	mu.Lock()
	defer mu.Unlock()

	footSwitchLevel := pins.FootSwitch.Read()
	estopLevel := pins.EStop.Read()
	proxLevel := pins.ProxInput.Read()

	s.FootSwitch = footSwitchLevel == gpio.High
	s.EStop = estopLevel == gpio.Low
	s.ProxInput = proxLevel == gpio.High

	sample, err := pins.Load.Read()
	if err != nil {
		log.Printf("error reading ADC load sensor: %v", err)
	} else {
		s.Load = math.Floor(((float64(sample.Raw)/32767.0)*4.096*2.5)*1000) / 1000
	}
}

// write outputs the state to GPIO pins
func (s *State) write(pins *Pins) {
	mu.RLock()
	defer mu.RUnlock()

	if s.DumpValve {
		pins.DumpValve.Out(gpio.High)
	} else {
		pins.DumpValve.Out(gpio.Low)
	}

	if s.Speed {
		pins.Speed.Out(gpio.High)
	} else {
		pins.Speed.Out(gpio.Low)
	}

	if s.Buzz {
		pins.Buzz.Out(gpio.High)
		s.Buzz = false
	} else {
		pins.Buzz.Out(gpio.Low)
	}
}
