package main

import (
	"fmt"
	"math"
	"sort"
)

// CalFromRaw converts a raw sensor value to a calibrated value using the calibration table
func CalFromRaw(data *CalTableConfig, rawValue float64) float64 {
	if len(data.CalPoints) < 2 {
		return rawValue
	}

	// Convert map keys and values from string to float64
	convertedCalPoints := make(map[float64]float64)
	for k, v := range data.CalPoints {
		var keyFloat, valFloat float64
		if _, err1 := fmt.Sscanf(k, "%f", &keyFloat); err1 != nil {
			continue
		}
		if _, err2 := fmt.Sscanf(v, "%f", &valFloat); err2 != nil {
			continue
		}
		convertedCalPoints[keyFloat] = valFloat
	}

	// Require at least 2 valid points
	if len(convertedCalPoints) < 2 {
		return rawValue
	}

	// Extract and sort known keys (the calibration output values)
	knownVals := make([]float64, 0, len(convertedCalPoints))
	for k := range convertedCalPoints {
		knownVals = append(knownVals, k)
	}
	sort.Float64s(knownVals)

	// Helper function to safely interpolate
	safeInterp := func(k1, k2, r1, r2, raw float64) float64 {
		if r2 == r1 {
			return k1 // avoid division by zero; return one of the known keys
		}
		val := k1 + (k2-k1)*(raw-r1)/(r2-r1)
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return rawValue
		}
		return val
	}

	// Handle extrapolation below the lowest point
	if rawValue < convertedCalPoints[knownVals[0]] {
		k1, k2 := knownVals[0], knownVals[1]
		r1, r2 := convertedCalPoints[k1], convertedCalPoints[k2]
		return safeInterp(k1, k2, r1, r2, rawValue)
	}

	// Handle extrapolation above the highest point
	if rawValue > convertedCalPoints[knownVals[len(knownVals)-1]] {
		k1 := knownVals[len(knownVals)-2]
		k2 := knownVals[len(knownVals)-1]
		r1, r2 := convertedCalPoints[k1], convertedCalPoints[k2]
		return safeInterp(k1, k2, r1, r2, rawValue)
	}

	// Interpolate between known points
	for i := 0; i < len(knownVals)-1; i++ {
		k1, k2 := knownVals[i], knownVals[i+1]
		r1, r2 := convertedCalPoints[k1], convertedCalPoints[k2]
		if rawValue >= r1 && rawValue <= r2 {
			return safeInterp(k1, k2, r1, r2, rawValue)
		}
	}

	return rawValue
}

// loop processes state updates based on calibration and alarm thresholds.
func loop(state *State, data *Data) {
	mu.RLock()
	tensionSettings := data.TensionSettings
	distancePerPulse := data.DistancePerPulse

	calPoints := make(map[string]string, len(data.CalTable.CalPoints))
	for k, v := range data.CalTable.CalPoints {
		calPoints[k] = v
	}

	currentLoad := state.Load
	proxInput := state.ProxInput
	prevProxInput := state.PrevProxInput
	proxValue := state.ProxValue
	footSwitch := state.FootSwitch
	eStop := state.EStop
	mu.RUnlock()

	if len(calPoints) > 0 {
		currentLoad = CalFromRaw(&CalTableConfig{CalPoints: calPoints}, currentLoad)
	}

	// Detect rising edge
	if proxInput && !prevProxInput {
		proxValue += distancePerPulse
	}

	alarmWarn := currentLoad >= tensionSettings.MaxTensionValue*tensionSettings.WarnTensionPercent/100
	alarmError := currentLoad >= tensionSettings.MaxTensionValue*tensionSettings.ErrorTensionPercent/100
	maxTension := currentLoad >= tensionSettings.MaxTensionValue

	alarmError = alarmError || maxTension || eStop
	dumpValve := footSwitch && !maxTension && !eStop

	mu.Lock()
	state.Load = currentLoad
	state.ProxValue = proxValue
	state.PrevProxInput = proxInput
	state.AlarmWarn = alarmWarn
	state.AlarmError = alarmError
	state.MaxTension = maxTension
	state.DumpValve = dumpValve
	mu.Unlock()
}
