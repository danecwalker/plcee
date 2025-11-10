package main

import "time"

// loggingLogic handles logging state transitions and log entry queueing
// This is called from the main loop to minimize allocations
func loggingLogic(state *State, data *Data, prevFootSwitch *bool, logTimer *time.Time, logIntervalDuration *time.Duration, usbCheckTimer *time.Time) {
	// Update log interval duration if changed
	if data.LogSettings.IntervalMs > 0 {
		newInterval := time.Duration(data.LogSettings.IntervalMs) * time.Millisecond
		if newInterval != *logIntervalDuration {
			*logIntervalDuration = newInterval
		}
	}

	// Check USB connection periodically (every 2 seconds to avoid overhead)
	now := time.Now()
	if now.Sub(*usbCheckTimer) >= 2*time.Second {
		state.UsbConnected = checkUsbConnected()
		*usbCheckTimer = now
	}

	// Handle foot switch state changes
	if state.FootSwitch && !*prevFootSwitch {
		// FootSwitch turned ON - start/continue logging
		if data.LogSettings.Enabled {
			queueLogStart()
		}
	} else if !state.FootSwitch && *prevFootSwitch {
		// FootSwitch turned OFF - stop logging
		if data.LogSettings.Enabled {
			queueLogStop()
		}
	}

	// Log entry if FootSwitch is on and interval has elapsed
	if state.FootSwitch && data.LogSettings.Enabled {
		if now.Sub(*logTimer) >= *logIntervalDuration {
			queueLogEntry(state)
			*logTimer = now
		}
	}

	*prevFootSwitch = state.FootSwitch
}
