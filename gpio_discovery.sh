#!/bin/bash
# GPIO Pin Discovery Script for Raspberry Pi CM5
# This script helps identify available and reserved GPIO pins

echo "=== GPIO Availability Check ==="
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "Note: Some GPIO information may be limited without root. Run with sudo for full details."
   echo ""
fi

# List sysfs GPIO directory
if [ -d /sys/class/gpio ]; then
    echo "Currently exported GPIO pins in /sys/class/gpio:"
    ls -d /sys/class/gpio/gpio* 2>/dev/null | sed 's|.*/gpio||' | sort -n || echo "  (none)"
    echo ""
else
    echo "ERROR: /sys/class/gpio not found - GPIO sysfs interface not available"
    exit 1
fi

# Check device tree overlays (if available)
if [ -f /boot/config.txt ]; then
    echo "GPIO-related config.txt settings:"
    grep -i gpio /boot/config.txt 2>/dev/null || echo "  (none found)"
    echo ""
fi

# List GPIO controller info
echo "GPIO controller information:"
if [ -d /sys/class/gpio/gpiochip0 ]; then
    for gpiochip in /sys/class/gpio/gpiochip*; do
        if [ -f "$gpiochip/label" ]; then
            label=$(cat "$gpiochip/label")
            ngpio=$(cat "$gpiochip/ngpio" 2>/dev/null || echo "?")
            base=$(cat "$gpiochip/base" 2>/dev/null || echo "?")
            echo "  $label: base=$base, ngpio=$ngpio"
        fi
    done
else
    echo "  (GPIO chips not found)"
fi
echo ""

# Test specific pins for CM5
echo "Testing CM5 default pin assignments:"
test_pins=(4 17 22 23 24 25 26 27 13 19 20 21)

for pin in "${test_pins[@]}"; do
    gpio_path="/sys/class/gpio/gpio$pin"
    if [ -d "$gpio_path" ]; then
        echo "  GPIO $pin: EXPORTED (already in use)"
    else
        # Try to export and check if it's valid
        if echo "$pin" > /sys/class/gpio/export 2>/dev/null; then
            echo "  GPIO $pin: OK (exported successfully)"
            echo "$pin" > /sys/class/gpio/unexport 2>/dev/null
        else
            echo "  GPIO $pin: UNAVAILABLE (invalid or reserved)"
        fi
    fi
done
echo ""

# Show pin function via device tree
if command -v raspi-gpio &> /dev/null; then
    echo "GPIO pin functions (raspi-gpio):"
    raspi-gpio get 13,17,19,22,23,24 2>/dev/null || echo "  (raspi-gpio command failed)"
fi
