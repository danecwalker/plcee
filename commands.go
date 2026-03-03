package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
)

const (
	minLogDelayMs    = 0
	maxLogDelayMs    = 3600000
	minLogIntervalMs = 10
	maxLogIntervalMs = 60000
)

func processCommands(state *State, data *Data) {
	// Process up to 10 commands per cycle to avoid starving the control loop.
	const maxCommands = 10
	processed := 0

	for processed < maxCommands {
		select {
		case cmd := <-commandQueue:
			log.Printf("processing command: %s", cmd.Name)

			err := applyCommand(cmd, state, data)
			if err != nil {
				log.Printf("command %s failed: %v", cmd.Name, err)
			}
			sendCommandResult(cmd, err)
			processed++

		default:
			// No more commands in queue.
			return
		}
	}
}

func sendCommandResult(cmd Command, err error) {
	if cmd.Result == nil {
		return
	}

	result := CommandResult{OK: err == nil}
	if err != nil {
		result.Error = err.Error()
	}

	select {
	case cmd.Result <- result:
	default:
	}
	close(cmd.Result)
}

func decodeCommandData(raw any, dst any) error {
	bytes, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}

func applyCommand(cmd Command, state *State, data *Data) error {
	switch cmd.Name {
	case "SetDumpValve":
		val, ok := cmd.Data.(bool)
		if !ok {
			return fmt.Errorf("invalid data type for SetDumpValve")
		}

		mu.Lock()
		state.DumpValve = val
		state.Buzz = true
		mu.Unlock()
		log.Printf("dump valve set to: %v", val)
		return nil

	case "SetSpeed":
		val, ok := cmd.Data.(bool)
		if !ok {
			return fmt.Errorf("invalid data type for SetSpeed")
		}

		mu.Lock()
		state.Speed = val
		state.Buzz = true
		mu.Unlock()
		log.Printf("speed set to: %v", val)
		return nil

	case "SetTensionSettings":
		var settings MaxTensionConfig
		if err := decodeCommandData(cmd.Data, &settings); err != nil {
			return fmt.Errorf("invalid tension settings payload: %w", err)
		}
		if err := validateTensionSettings(settings); err != nil {
			return err
		}

		mu.Lock()
		data.TensionSettings = settings
		state.Buzz = true
		mu.Unlock()

		log.Printf("tension settings updated: %+v", settings)
		markDataDirty()
		return nil

	case "SetLogSettings":
		var settings LogSettingConfig
		if err := decodeCommandData(cmd.Data, &settings); err != nil {
			return fmt.Errorf("invalid log settings payload: %w", err)
		}
		if err := validateLogSettings(settings); err != nil {
			return err
		}

		mu.Lock()
		data.LogSettings = settings
		state.Buzz = true
		mu.Unlock()

		log.Printf("log settings updated: %+v", settings)
		markDataDirty()
		return nil

	case "SetCalibrationTable":
		var settings CalTableConfig
		if err := decodeCommandData(cmd.Data, &settings); err != nil {
			return fmt.Errorf("invalid calibration table payload: %w", err)
		}
		if err := validateCalibrationTable(settings); err != nil {
			return err
		}

		mu.Lock()
		data.CalTable = settings
		state.Buzz = true
		mu.Unlock()

		log.Printf("calibration table updated with %d points", len(settings.CalPoints))
		markDataDirty()
		return nil

	case "ZeroDistance":
		mu.Lock()
		state.ProxValue = 0
		state.Buzz = true
		mu.Unlock()
		log.Println("proximity distance zeroed")
		return nil

	default:
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
}

func validateTensionSettings(settings MaxTensionConfig) error {
	if math.IsNaN(settings.MaxTensionValue) || math.IsInf(settings.MaxTensionValue, 0) || settings.MaxTensionValue < 0 {
		return fmt.Errorf("MaxTensionValue must be a finite number >= 0")
	}
	if math.IsNaN(settings.WarnTensionPercent) || math.IsInf(settings.WarnTensionPercent, 0) || settings.WarnTensionPercent < 0 || settings.WarnTensionPercent > 100 {
		return fmt.Errorf("WarnTensionPercent must be between 0 and 100")
	}
	if math.IsNaN(settings.ErrorTensionPercent) || math.IsInf(settings.ErrorTensionPercent, 0) || settings.ErrorTensionPercent < 0 || settings.ErrorTensionPercent > 100 {
		return fmt.Errorf("ErrorTensionPercent must be between 0 and 100")
	}
	if settings.ErrorTensionPercent < settings.WarnTensionPercent {
		return fmt.Errorf("ErrorTensionPercent must be >= WarnTensionPercent")
	}
	return nil
}

func validateLogSettings(settings LogSettingConfig) error {
	if settings.LogDelayMs < minLogDelayMs || settings.LogDelayMs > maxLogDelayMs {
		return fmt.Errorf("LogDelayMs must be between %d and %d", minLogDelayMs, maxLogDelayMs)
	}
	if settings.IntervalMs < minLogIntervalMs || settings.IntervalMs > maxLogIntervalMs {
		return fmt.Errorf("IntervalMs must be between %d and %d", minLogIntervalMs, maxLogIntervalMs)
	}
	return nil
}

func validateCalibrationTable(settings CalTableConfig) error {
	if settings.CalPoints == nil {
		return fmt.Errorf("CalTable cannot be nil")
	}

	for knownLoad, rawReading := range settings.CalPoints {
		if _, err := strconv.ParseFloat(knownLoad, 64); err != nil {
			return fmt.Errorf("invalid known load '%s'", knownLoad)
		}
		if _, err := strconv.ParseFloat(rawReading, 64); err != nil {
			return fmt.Errorf("invalid raw reading '%s'", rawReading)
		}
	}

	return nil
}
