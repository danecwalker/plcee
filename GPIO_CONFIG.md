# GPIO Configuration for DELPHI on Raspberry Pi CM5

## Overview

This application now uses a custom sysfs-based GPIO implementation instead of the `periph` library. GPIO pins are configurable via `gpio_config.yaml`.

## Error: "invalid argument" when exporting GPIO

The error `failed to export GPIO 13: write /sys/class/gpio/export: invalid argument` typically means one of the following:

1. **GPIO pin is invalid or not available** on your CM5 board
2. **GPIO pin is already in use** by the kernel or another process
3. **GPIO pin doesn't exist** in the device tree configuration
4. **Insufficient permissions** (not running as root)

## Troubleshooting Steps

### Step 1: Run the GPIO Discovery Script

```bash
sudo ./gpio_discovery.sh
```

This will show:
- Which GPIO pins are currently exported
- Which pins are available for use
- GPIO controller information

### Step 2: Test Individual Pins

Try exporting a known-working pin manually:

```bash
# Test GPIO 4 (usually available on CM5)
echo 4 > /sys/class/gpio/export
cat /sys/class/gpio/gpio4/direction

# Unexport when done
echo 4 > /sys/class/gpio/unexport
```

### Step 3: Identify Your CM5 Pinout

The default pin assignments are:
- **FootSwitch**: GPIO 13 (input)
- **EStop**: GPIO 17 (input)
- **ProxInput**: GPIO 22 (input)
- **DumpValve**: GPIO 23 (output)
- **Speed**: GPIO 24 (output)
- **Buzz**: GPIO 19 (output)

If any of these pins are unavailable, you need to remap them.

## Configuration

### Using gpio_config.yaml

Edit `gpio_config.yaml` to map your actual GPIO pins:

```yaml
gpio_pins:
  foot_switch: 4      # Change from 13 to GPIO 4
  estop: 17
  prox_input: 22
  dump_valve: 23
  speed: 24
  buzz: 19
```

### Check Application Logs

When the app starts, it will log:
```
2025/11/11 02:08:00 GPIO configuration loaded:
  FootSwitch: GPIO 4
  EStop: GPIO 17
  ProxInput: GPIO 22
  DumpValve: GPIO 23
  Speed: GPIO 24
  Buzz: GPIO 19
2025/11/11 02:08:00 Currently exported GPIO pins: [4 17 22]
```

## Common CM5 GPIO Issues

### Issue: GPIO 13 not available
Some CM5 boards may use GPIO 13 for specific functions. Common alternatives:
- GPIO 4, 5, 6, 12, 16, 20, 21, 25, 26, 27

### Issue: Running without root
GPIO sysfs access requires root permissions. Run with:
```bash
sudo ./delphi
```

### Issue: GPIO already exported by kernel
Some GPIOs may be reserved by the kernel for built-in functions. Check:
```bash
cat /sys/kernel/debug/gpio
```

## Testing GPIO Access

Once configured, test that GPIO access works:

```bash
# Run with verbose output
./delphi 2>&1 | grep -i gpio
```

You should see:
```
GPIO configuration loaded
Currently exported GPIO pins
FootSwitch pin: GPIO XX
EStop pin: GPIO XX
...
GPIO pins configured successfully
```

## Advanced: Check Device Tree

To see which GPIOs are available in your device tree:

```bash
# List all GPIO banks
cat /proc/device-tree/model

# Check GPIO pinmux (if available)
grep -r "gpio" /proc/device-tree/ | head -20
```

## Default Pins for Reference

These are standard GPIO assignments for Raspberry Pi (may vary on CM5):

| Function | GPIO | Alternative |
|----------|------|-------------|
| LED | 17 | 27 |
| Button | 4 | 18, 23 |
| Output | 23 | 24, 25 |
| Output | 24 | 25, 26 |
| Output | 19 | 20, 21 |

Test with a simple pin first (usually GPIO 4 or 17 are reliable), then configure the rest accordingly.
