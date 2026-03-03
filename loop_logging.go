package main

import "time"

// loggingLogic handles logging state transitions and log entry queueing.
func loggingLogic(state *State, data *Data, prevFootSwitch *bool, prevLoggingEnabled *bool, logTimer *time.Time, logIntervalDuration *time.Duration) {
	mu.RLock()
	logEnabled := data.LogSettings.Enabled
	intervalMs := data.LogSettings.IntervalMs
	footSwitch := state.FootSwitch
	mu.RUnlock()

	mu.Lock()
	state.LogEnabled = logEnabled
	mu.Unlock()

	if *logIntervalDuration <= 0 {
		*logIntervalDuration = time.Second
	}

	if intervalMs > 0 {
		newInterval := time.Duration(intervalMs) * time.Millisecond
		if newInterval != *logIntervalDuration {
			*logIntervalDuration = newInterval
		}
	}

	now := time.Now()

	// Handle logging enable/disable transitions explicitly.
	if logEnabled != *prevLoggingEnabled {
		if !logEnabled && *prevLoggingEnabled {
			queueLogStop()
		}
		if logEnabled && footSwitch {
			queueLogStart()
		}
		*prevLoggingEnabled = logEnabled
	}

	// Handle foot switch state changes.
	if footSwitch && !*prevFootSwitch {
		if logEnabled {
			queueLogStart()
		}
	} else if !footSwitch && *prevFootSwitch {
		if logEnabled {
			queueLogStop()
		}
	}

	// Log entry if FootSwitch is on and interval has elapsed.
	if footSwitch && logEnabled {
		if now.Sub(*logTimer) >= *logIntervalDuration {
			queueLogEntry(state)
			*logTimer = now
		}
	}

	*prevFootSwitch = footSwitch
}
