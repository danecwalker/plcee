package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var hardwareError error

func main() {
	// Command-line flags
	busName := flag.String("bus", "/dev/i2c-1", "I2C bus device path (default: /dev/i2c-1)")
	gpioBase := flag.Int("gpio-base", 569, "GPIO base pin number for CM5 (default: 569 for pinctrl-rp1)")
	distance := flag.Float64("distance", 0.189, "Distance per pulse in meters (default: 0.189m for standard setup)")
	flag.Parse()

	log.Println("starting PLCEE application...")
	log.Printf("I2C bus: %q", *busName)
	log.Printf("GPIO base: %d", *gpioBase)

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
			DistancePerPulse: *distance,
			AdminPassword:    "Hello",
			ProtectedRoutes:  []string{""},
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

	// Set GPIO base offset
	SetGPIOBase(*gpioBase)

	// Start background data writer
	go startDataWriter(&data)
	log.Println("data persistence worker started")

	// Initialize hardware
	var pins *Pins
	adc, err := NewADS1115(*busName, ADS1115_ADDRESS)
	if err != nil {
		log.Printf("failed to initialize ADS1115 ADC: %v", err)
		hardwareError = err
	} else {
		log.Println("ADS1115 ADC initialized successfully")
		pins, err = setupPins(adc)
		if err != nil {
			log.Printf("failed to setup GPIO pins: %v", err)
			hardwareError = err
			// Close ADC if pin setup failed
			adc.Close()
		}
	}

	// Only start control loop if hardware initialized successfully
	if hardwareError == nil && pins != nil {
		// Ensure cleanup on exit
		defer closePins(pins)

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
		mux.HandleFunc("/auth", authHandler(&data))
		mux.HandleFunc("/api/hardware-error", hardwareErrorAPIHandler())
		mux.HandleFunc("/", rootHandler(&data))

		log.Println("starting HTTP server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	} else {
		// Hardware error - start minimal server with error page
		log.Printf("hardware initialization failed, starting in error mode")
		mux := http.NewServeMux()
		mux.HandleFunc("/api/hardware-error", hardwareErrorAPIHandler())
		mux.HandleFunc("/auth", authHandler(&data))
		mux.HandleFunc("/", rootHandler(&data))

		log.Println("starting HTTP server on :8080 (error mode)")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}
}
