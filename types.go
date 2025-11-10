package main

import (
	"sync"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/devices/v3/ads1x15"
)

// MaxTensionConfig holds tension warning/error thresholds
type MaxTensionConfig struct {
	MaxTensionValue     float64 `yaml:"max_tension_value" json:"MaxTensionValue"`
	WarnTensionPercent  float64 `yaml:"warn_tension_percent" json:"WarnTensionPercent"`
	ErrorTensionPercent float64 `yaml:"error_tension_percent" json:"ErrorTensionPercent"`
}

// CalTableConfig holds calibration points
type CalTableConfig struct {
	CalPoints map[string]string `yaml:"cal_points" json:"CalTable"`
}

// LogSettingConfig holds logging configuration
type LogSettingConfig struct {
	LogDelayMs int  `yaml:"delay_ms" json:"LogDelayMs"`
	IntervalMs int  `yaml:"interval_ms" json:"IntervalMs"`
	Enabled    bool `yaml:"enabled" json:"Enabled"`
}

// Data holds persistent configuration
type Data struct {
	TensionSettings MaxTensionConfig `yaml:"tension_settings"`
	CalTable        CalTableConfig   `yaml:"cal_table"`
	LogSettings     LogSettingConfig `yaml:"log_settings"`
}

// State holds the current system state
type State struct {
	// Inputs
	FootSwitch    bool
	EStop         bool
	ProxInput     bool
	PrevProxInput bool

	// Outputs
	DumpValve bool
	Speed     bool

	// Analog
	Load float64

	UsbConnected bool

	// Memory
	Buzz       bool
	ProxValue  float64
	AlarmError bool
	AlarmWarn  bool
	MaxTension bool
}

// Pins holds GPIO pin references
type Pins struct {
	FootSwitch gpio.PinIn
	EStop      gpio.PinIn
	ProxInput  gpio.PinIn

	DumpValve gpio.PinOut
	Speed     gpio.PinOut
	Buzz      gpio.PinOut

	Load ads1x15.PinADC
}

// Command represents a control command
type Command struct {
	Name string
	Data any
}

var (
	mu             sync.RWMutex
	commandQueue   chan Command
	dataFile       = "data.yaml"
	dataWriteQueue chan *Data
	dataDirty      bool
	dataDirtyMutex sync.Mutex
)
