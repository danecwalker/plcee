#!/bin/bash
# Quick test of GPIO access on CM5

echo "Testing GPIO access..."
echo ""

# Get the pinctrl-rp1 chip (should be GPIO 569+)
CHIP=$(grep -l "pinctrl-rp1" /sys/class/gpio/gpiochip*/label 2>/dev/null | head -1)
if [ -z "$CHIP" ]; then
    echo "ERROR: Cannot find pinctrl-rp1 chip"
    exit 1
fi

CHIP_DIR=$(dirname "$CHIP")
BASE=$(cat "$CHIP_DIR/base")
NGPIO=$(cat "$CHIP_DIR/ngpio")
END=$((BASE + NGPIO - 1))

echo "Found pinctrl-rp1:"
echo "  Base: $BASE"
echo "  Range: $BASE-$END"
echo ""

# Test the first 6 pins
echo "Testing GPIO $BASE-$((BASE+5))..."
for i in 0 1 2 3 4 5; do
    PIN=$((BASE + i))
    echo -n "  GPIO $PIN: "
    
    if echo "$PIN" > /sys/class/gpio/export 2>/dev/null; then
        if echo "out" > /sys/class/gpio/gpio$PIN/direction 2>/dev/null; then
            echo "✓ OK"
            echo "$PIN" > /sys/class/gpio/unexport 2>/dev/null
        else
            echo "✗ Cannot set direction"
            echo "$PIN" > /sys/class/gpio/unexport 2>/dev/null
        fi
    else
        echo "✗ Cannot export"
    fi
done

echo ""
echo "GPIO $BASE-$((BASE+5)) are recommended for the application."
