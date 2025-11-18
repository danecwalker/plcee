package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GPIOLevel represents the state of a GPIO pin
type GPIOLevel int

const (
	Low  GPIOLevel = 0
	High GPIOLevel = 1
)

const sysfsGPIOPath = "/sys/class/gpio"

// gpioBase is the base offset for all GPIO pins (set at runtime)
var gpioBase int = 0

// SetGPIOBase sets the base offset for GPIO pin numbering
func SetGPIOBase(base int) {
	gpioBase = base
	fmt.Printf("GPIO base offset set to: %d\n", gpioBase)
}

// GPIOPin represents a GPIO pin accessed via sysfs
type GPIOPin struct {
	number   int
	pin      string
	basePath string
	dir      string // "in" or "out"
}

// NewGPIOPin exports a GPIO pin and opens it
func NewGPIOPin(pinNumber int, direction string) (*GPIOPin, error) {
	// Add base offset to pin number
	actualPin := pinNumber + gpioBase

	gp := &GPIOPin{
		number:   actualPin,
		pin:      strconv.Itoa(actualPin),
		basePath: filepath.Join(sysfsGPIOPath, fmt.Sprintf("gpio%d", actualPin)),
		dir:      direction,
	}

	// Check if already exported
	if _, err := os.Stat(gp.basePath); os.IsNotExist(err) {
		// Pin not exported, so export it
		exportPath := filepath.Join(sysfsGPIOPath, "export")
		f, err := os.OpenFile(exportPath, os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open GPIO export (may need root): %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(gp.pin); err != nil {
			return nil, fmt.Errorf("failed to export GPIO %d (offset %d + base %d, may already be in use or invalid pin): %w", actualPin, pinNumber, gpioBase, err)
		}
	} else if err != nil {
		// Stat returned an error other than "not exist"
		return nil, fmt.Errorf("failed to stat GPIO %d path: %w", actualPin, err)
	}
	// else: pin already exported, which is fine

	// Set direction
	directionPath := filepath.Join(gp.basePath, "direction")
	if err := os.WriteFile(directionPath, []byte(direction), 0644); err != nil {
		return nil, fmt.Errorf("failed to set GPIO %d direction to %s (may need root): %w", actualPin, direction, err)
	}

	return gp, nil
}

// In configures the pin as input with optional pull-up
func (gp *GPIOPin) In(pullUp bool) error {
	directionPath := filepath.Join(gp.basePath, "direction")
	return os.WriteFile(directionPath, []byte("in"), 0644)
}

// Out configures the pin as output with initial level
func (gp *GPIOPin) Out(level GPIOLevel) error {
	// Set initial value before changing direction
	valuePath := filepath.Join(gp.basePath, "value")
	levelStr := "0"
	if level == High {
		levelStr = "1"
	}

	if err := os.WriteFile(valuePath, []byte(levelStr), 0644); err != nil {
		return fmt.Errorf("failed to set GPIO %d initial value: %w", gp.number, err)
	}

	directionPath := filepath.Join(gp.basePath, "direction")
	return os.WriteFile(directionPath, []byte("out"), 0644)
}

// Read reads the current level of the pin
func (gp *GPIOPin) Read() GPIOLevel {
	valuePath := filepath.Join(gp.basePath, "value")
	data, err := os.ReadFile(valuePath)
	if err != nil {
		fmt.Printf("error reading GPIO %d value: %v\n", gp.number, err)
		return Low
	}

	value := strings.TrimSpace(string(data))
	if value == "1" {
		return High
	}
	return Low
}

// Write writes a level to the pin
func (gp *GPIOPin) Write(level GPIOLevel) error {
	valuePath := filepath.Join(gp.basePath, "value")
	levelStr := "0"
	if level == High {
		levelStr = "1"
	}

	return os.WriteFile(valuePath, []byte(levelStr), 0644)
}

// Halt closes the pin and unexports it
func (gp *GPIOPin) Halt() error {
	return unexportGPIO(gp.number)
}

// unexportGPIO unexports a GPIO pin
func unexportGPIO(pinNumber int) error {
	unexportPath := filepath.Join(sysfsGPIOPath, "unexport")
	f, err := os.OpenFile(unexportPath, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GPIO unexport: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(strconv.Itoa(pinNumber)); err != nil {
		return fmt.Errorf("failed to unexport GPIO %d: %w", pinNumber, err)
	}
	return nil
}

// ListExportedGPIOs returns a list of currently exported GPIO pins
func ListExportedGPIOs() ([]int, error) {
	entries, err := os.ReadDir(sysfsGPIOPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read GPIO directory: %w", err)
	}

	var exported []int
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "gpio") {
			numStr := strings.TrimPrefix(entry.Name(), "gpio")
			if num, err := strconv.Atoi(numStr); err == nil {
				exported = append(exported, num)
			}
		}
	}
	return exported, nil
}
