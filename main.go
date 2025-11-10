package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ads1x15"
	"periph.io/x/host/v3"
)

func main() {
	log.Println("starting PLCEE application...")

	if s, err := host.Init(); err != nil {
		log.Fatalf("failed to initialize peripheral host: %v", err)
	} else {
		log.Println("peripheral host initialized successfully")
		log.Printf("periph loaded: %s", s)
	}

	var data Data

	// check if data file exists
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		log.Printf("data file '%s' not found, creating with default values", dataFile)
		data = Data{
			TensionSettings: MaxTensionConfig{
				MaxTensionValue:     0.0,
				WarnTensionPercent:  0.0,
				ErrorTensionPercent: 0.0,
			},
			CalTable: CalTableConfig{
				CalPoints: map[string]string{},
			},
			LogSettings: LogSettingConfig{
				LogDelayMs: 1000,
				IntervalMs: 1000,
				Enabled:    false,
			},
		}
		// create default data file
		f, err := os.Create(dataFile)
		if err != nil {
			log.Fatalf("failed to create data file: %v", err)
		}
		defer f.Close()

		encoder := yaml.NewEncoder(f)
		if err := encoder.Encode(&data); err != nil {
			log.Fatalf("failed to encode default data to YAML: %v", err)
		}
		log.Println("default data file created successfully")
	} else {
		log.Printf("loading data from '%s'", dataFile)
		// read data file
		f, err := os.Open(dataFile)
		if err != nil {
			log.Fatalf("failed to open data file: %v", err)
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(&data); err != nil {
			log.Fatalf("failed to decode YAML data: %v", err)
		}
		log.Printf("data loaded successfully: max tension=%.2f, warn=%.1f%%, error=%.1f%%, cal points=%d",
			data.TensionSettings.MaxTensionValue,
			data.TensionSettings.WarnTensionPercent*100,
			data.TensionSettings.ErrorTensionPercent*100,
			len(data.CalTable.CalPoints))
	}

	commandQueue = make(chan Command, 100)
	log.Println("command queue initialized")

	dataWriteQueue = make(chan *Data, 10)
	log.Println("data write queue initialized")

	// Start background data writer
	go startDataWriter(&data)
	log.Println("data persistence worker started")

	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I2C bus: %v", err)
	}
	defer bus.Close()
	log.Println("I2C bus opened successfully")

	adc, err := ads1x15.NewADS1115(bus, &ads1x15.DefaultOpts)
	if err != nil {
		log.Fatalf("failed to initialize ADS1115 ADC: %v", err)
	}
	log.Println("ADS1115 ADC initialized successfully")

	pins := setupPins(adc)

	var state State = State{
		DumpValve: false,
		Speed:     false,

		ProxValue: 0.0,

		Load: 0.0,

		FootSwitch: false,
		EStop:      false,
		ProxInput:  false,

		AlarmError: false,
		AlarmWarn:  false,
		MaxTension: false,

		UsbConnected: false,
	}

	state.read(pins)
	log.Println("initial state read from pins")

	// Start background logging worker
	startLogger(&state, &data)

	// Application logic using state...
	log.Println("starting main control loop")
	go func() {
		var prevFootSwitch bool
		var logTimer time.Time
		var usbCheckTimer time.Time
		logIntervalDuration := time.Duration(data.LogSettings.IntervalMs) * time.Millisecond

		for {
			state.read(pins)
			processCommands(&state, &data)
			loop(&state, &data)
			loggingLogic(&state, &data, &prevFootSwitch, &logTimer, &logIntervalDuration, &usbCheckTimer)
			state.write(pins)
		}
	}()

	mux := http.NewServeMux()
	log.Println("setting up HTTP handlers")

	mux.HandleFunc("/snapshot/stream", snapshotStreamHandler(pins, &state))
	mux.HandleFunc("/snapshot", snapshotHandler(pins, &state))
	mux.HandleFunc("/data", dataHandler(&data))
	mux.HandleFunc("/command", commandHandler())
	mux.HandleFunc("/auth", authHandler())
	mux.HandleFunc("/", rootHandler())

	log.Println("starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
