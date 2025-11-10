# Logging System

## Overview
The logging system runs as a background worker that saves sensor data to CSV files on a USB drive when certain conditions are met.

## Features
- **Zero-allocation main loop**: All logging operations are queued to a background worker
- **USB detection**: Automatically detects USB drive and only logs when connected
- **Smart session management**: Creates new recordings or continues existing ones based on timing
- **CSV format**: Easy to import into spreadsheet applications

## Configuration

### State Fields
- `LogEnabled` (bool): Enable/disable logging system
- `LogIntervalMs` (int): Interval between log entries in milliseconds (default: 1000ms)
- `LogDelayMs` (int): Maximum delay before starting a new recording session (default: 5000ms)
- `UsbConnected` (bool): Read-only status of USB connection

### Commands
Send these commands to the `/command` endpoint:

```json
// Enable/disable logging
{"Name": "SetLogEnabled", "Data": true}

// Set log interval (milliseconds)
{"Name": "SetLogInterval", "Data": 100}

// Set session timeout (milliseconds)
{"Name": "SetLogDelay", "Data": 5000}
```

## How It Works

1. **Logging starts** when the FootSwitch is pressed and `LogEnabled` is true
2. **Data is logged** at the specified `LogIntervalMs` interval while FootSwitch is held
3. **Logging pauses** when FootSwitch is released
4. **New session logic**:
   - If FootSwitch is pressed again within `LogDelayMs`, logging continues to the same file
   - If more than `LogDelayMs` has elapsed, a new recording file is created

## USB Drive Setup

1. Mount your USB drive to `/media/usb` (or update `usbMountPath` in `logging.go`)
2. The system will automatically create files named `recording_1.csv`, `recording_2.csv`, etc.

## CSV Format

Each recording contains the following columns:
- `Timestamp`: ISO 8601 timestamp with nanosecond precision
- `Load`: Load sensor value
- `ProxValue`: Proximity sensor value
- `DumpValve`: Dump valve state (true/false)
- `Speed`: Speed control state (true/false)
- `AlarmError`: Error alarm state (true/false)
- `AlarmWarn`: Warning alarm state (true/false)
- `MaxTension`: Max tension reached (true/false)

## Example Recording

```csv
Timestamp,Load,ProxValue,DumpValve,Speed,AlarmError,AlarmWarn,MaxTension
2025-11-05T10:30:00.123456789Z,1.234,5.678,false,true,false,false,false
2025-11-05T10:30:01.123456789Z,1.456,5.890,false,true,false,false,false
```
