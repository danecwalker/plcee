# CM5 GPIO Configuration Guide

## Problem

The Compute Module 5 (CM5) uses different GPIO numbering than standard Raspberry Pi boards. Instead of GPIOs 0-27, CM5 uses higher numbers (512+) organized by GPIO chips.

Your error:
```
failed to export GPIO 13: write /sys/class/gpio/export: invalid argument
```

This means GPIO 13 doesn't exist on your CM5.

## Solution

### Step 1: Auto-Detection (Recommended)

The application now **automatically detects available GPIO pins on CM5**. Simply run:

```bash
sudo ./delphi
```

The app will:
1. Read `/sys/class/gpio/gpiochip*` to find available GPIO controllers
2. Auto-assign pins from different chips to your functions
3. Log the discovered configuration

You should see output like:
```
Found GPIO chip: 2710_gpio (base=512, ngpio=54)
Found GPIO chip: pinctrl-bcm2835 (base=566, ngpio=4)
GPIO configuration:
  FootSwitch: GPIO 512
  EStop: GPIO 513
  ...
```

### Step 2: Manual Configuration (If Auto-Detection Fails)

Create a `gpio_config.yaml` file with your actual GPIO numbers:

```yaml
gpio_pins:
  foot_switch: 512
  estop: 513
  prox_input: 514
  dump_valve: 515
  speed: 516
  buzz: 517
```

### Step 3: Find Your GPIO Numbers

Run this on the CM5 to list all available GPIO chips:

```bash
for chip in /sys/class/gpio/gpiochip*; do
  label=$(cat $chip/label)
  base=$(cat $chip/base)
  ngpio=$(cat $chip/ngpio)
  end=$((base + ngpio - 1))
  echo "$label: GPIO $base-$end"
done
```

Example output:
```
2710_gpio: GPIO 512-565
pinctrl-bcm2835: GPIO 566-569
```

### Step 4: Test a GPIO Pin

Test if a specific GPIO works:

```bash
# Test GPIO 512
sudo bash -c 'echo 512 > /sys/class/gpio/export && echo out > /sys/class/gpio/gpio512/direction && echo 1 > /sys/class/gpio/gpio512/value && echo "GPIO 512 works!" && echo 512 > /sys/class/gpio/unexport'
```

## GPIO Chip Information

### 2710_gpio
- **Base**: 512
- **Count**: 54 pins
- **Range**: GPIO 512-565
- **Usage**: Main GPIO controller for most pins
- **Recommended for**: All application functions

### pinctrl-bcm2835
- **Base**: 566
- **Count**: 4 pins
- **Range**: GPIO 566-569
- **Usage**: Built-in pins (often reserved)
- **Recommended for**: Backup if 2710_gpio runs out

## Typical CM5 Assignment

For a typical CM5 setup with the main GPIO controller:

```yaml
gpio_pins:
  foot_switch: 512   # Input
  estop: 513         # Input
  prox_input: 514    # Input
  dump_valve: 515    # Output
  speed: 516         # Output
  buzz: 517          # Output
```

## Troubleshooting

### "Permission denied" errors
```bash
sudo ./delphi
```

### GPIO still not exporting
```bash
# Check if GPIO is already exported
ls /sys/class/gpio/gpio512/

# If it exists, check what's using it
sudo lsof /sys/class/gpio/gpio512/

# Unexport if needed
echo 512 | sudo tee /sys/class/gpio/unexport
```

### Find physical pins for GPIO numbers
Use the CM5 documentation or pinout reference. Most GPIO 512+ pins map to standard header positions.

## Application Logging

The app logs GPIO information on startup. Look for:

```
Found GPIO chip: ...
GPIO configuration:
  FootSwitch: GPIO XXX
  EStop: GPIO XXX
  ...
Currently exported GPIO pins: [...]
GPIO pins configured successfully
```

This confirms auto-detection worked correctly.
