#!/bin/bash
# CM5 GPIO Discovery Tool
# Lists GPIO chips and their pin ranges

echo "=== CM5 GPIO Chip Information ==="
echo ""

for chip in /sys/class/gpio/gpiochip*; do
    if [ -d "$chip" ]; then
        label=$(cat "$chip/label" 2>/dev/null || echo "unknown")
        base=$(cat "$chip/base" 2>/dev/null || echo "?")
        ngpio=$(cat "$chip/ngpio" 2>/dev/null || echo "?")
        
        if [ "$ngpio" != "?" ] && [ "$base" != "?" ]; then
            end=$((base + ngpio - 1))
            echo "Chip: $label"
            echo "  Base: $base, Count: $ngpio"
            echo "  GPIO Range: $base-$end"
            echo ""
        fi
    fi
done

echo "=== Testing GPIO Availability ==="
echo ""
echo "Testing first GPIO from each chip..."
echo ""

# Test a few pins from different ranges
test_pins=(512 527 533 565 569)

for pin in "${test_pins[@]}"; do
    echo -n "GPIO $pin: "
    if echo "$pin" > /sys/class/gpio/export 2>/dev/null; then
        if echo "out" > /sys/class/gpio/gpio$pin/direction 2>/dev/null; then
            echo "✓ OK"
            echo "$pin" > /sys/class/gpio/unexport 2>/dev/null
        else
            echo "✗ Can't set direction"
            echo "$pin" > /sys/class/gpio/unexport 2>/dev/null
        fi
    else
        echo "✗ Can't export"
    fi
done
