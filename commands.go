package main

import (
	"encoding/json"
	"log"
)

func processCommands(state *State, data *Data) {
	// Process up to 10 commands per cycle to avoid starving the control loop
	maxCommands := 10
	processed := 0

	for processed < maxCommands {
		select {
		case cmd := <-commandQueue:
			log.Printf("processing command: %s", cmd.Name)

			// Process command based on type
			switch cmd.Name {
			case "SetDumpValve":
				if val, ok := cmd.Data.(bool); ok {
					mu.Lock()
					state.DumpValve = val
					state.Buzz = true
					mu.Unlock()
					log.Printf("dump valve set to: %v", val)
				} else {
					log.Printf("error: invalid data type for SetDumpValve command")
				}
			case "SetSpeed":
				if val, ok := cmd.Data.(bool); ok {
					mu.Lock()
					state.Speed = val
					state.Buzz = true
					mu.Unlock()
					log.Printf("speed set to: %v", val)
				} else {
					log.Printf("error: invalid data type for SetSpeed command")
				}
			case "SetTensionSettings":
				var settings MaxTensionConfig
				bytes, err := json.Marshal(cmd.Data)
				if err != nil {
					log.Printf("error marshaling tension settings: %v", err)
					continue
				}
				if err := json.Unmarshal(bytes, &settings); err != nil {
					log.Printf("error unmarshaling tension settings: %v", err)
					continue
				}

				mu.Lock()
				data.TensionSettings = settings
				state.Buzz = true
				mu.Unlock()

				log.Printf("tension settings updated: %+v", settings)
				markDataDirty()
			case "SetLogSettings":
				var settings LogSettingConfig
				bytes, err := json.Marshal(cmd.Data)
				if err != nil {
					log.Printf("error marshaling log settings: %v", err)
					continue
				}
				if err := json.Unmarshal(bytes, &settings); err != nil {
					log.Printf("error unmarshaling log settings: %v", err)
					continue
				}

				mu.Lock()
				data.LogSettings = settings
				state.Buzz = true
				mu.Unlock()

				log.Printf("log settings updated: %+v", settings)
				markDataDirty()
			case "SetCalibrationTable":
				var settings CalTableConfig
				bytes, err := json.Marshal(cmd.Data)
				if err != nil {
					log.Printf("error marshaling calibration table: %v", err)
					continue
				}
				if err := json.Unmarshal(bytes, &settings); err != nil {
					log.Printf("error unmarshaling calibration table: %v", err)
					continue
				}

				mu.Lock()
				data.CalTable = settings
				state.Buzz = true
				mu.Unlock()

				log.Printf("calibration table updated with %d points", len(settings.CalPoints))
				markDataDirty()
			case "ZeroDistance":
				mu.Lock()
				state.ProxValue = 0
				state.Buzz = true
				mu.Unlock()
				log.Println("proximity distance zeroed")
			default:
				log.Printf("warning: unknown command received: %s", cmd.Name)
			}

			processed++

		default:
			// No more commands in queue
			return
		}
	}
}
