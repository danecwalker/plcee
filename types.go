package main

import (
	"sync"
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
	TensionSettings  MaxTensionConfig `yaml:"tension_settings"`
	CalTable         CalTableConfig   `yaml:"cal_table"`
	LogSettings      LogSettingConfig `yaml:"log_settings"`
	DistancePerPulse float64          `yaml:"distance_per_pulse" json:"DistancePerPulse"`
	AdminPassword    string           `yaml:"admin_password" json:"AdminPassword"`
	ProtectedRoutes  []string         `yaml:"protected_routes" json:"ProtectedRoutes"`
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
	FootSwitch *GPIOPin
	EStop      *GPIOPin
	ProxInput  *GPIOPin

	DumpValve *GPIOPin
	Speed     *GPIOPin
	Buzz      *GPIOPin

	ADC     *ADS1115
	ADCChan int
}

// Command represents a control command
type Command struct {
	Name string
	Data any
}

var (
	mu           sync.RWMutex
	commandQueue chan Command
	dataFile     = "/usr/local/bin/data.yaml"
	// dataFile       = "data.yaml"
	dataWriteQueue chan *Data
	dataDirty      bool
	dataDirtyMutex sync.Mutex
)
