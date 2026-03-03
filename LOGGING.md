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
// Update all logging settings in one command
{"Name": "SetLogSettings", "Data": {"LogDelayMs": 5000, "IntervalMs": 1000, "Enabled": true}}
```

## How It Works

1. **Logging starts** when the FootSwitch is pressed and `LogEnabled` is true
2. **Data is logged** at the specified `LogIntervalMs` interval while FootSwitch is held
3. **Logging pauses** when FootSwitch is released
4. **New session logic**:
   - If FootSwitch is pressed again within `LogDelayMs`, logging continues to the same file
   - If more than `LogDelayMs` has elapsed, a new recording file is created

## USB Drive Setup

1. Mount your USB drive to `/mnt/usb`
2. The app writes logs under `/mnt/usb/data`
3. The system automatically creates files named `recording_YYYY-MM-DD_HHMMSS.csv`

## Device Logs

- The app also stores recordings locally under `/var/log/delphi`
- Download all local logs as a zip from:
  - `GET /logs/device/download`
  - Requires authenticated session cookie (`/auth` login)

## Logging Health Endpoint

- `GET /health/logging` returns live logging diagnostics (no auth required):
  - overall status (`ok`, `degraded`, `error`)
  - USB state (`connected`, `mounted`, `healthy`, `error`)
  - local device log health (`healthy`, `error`)
  - logging queue depth/capacity/utilization
  - control loop fault state (`controlLoop.error`)

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
