package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// I2C ioctl commands
	I2C_SLAVE = 0x0703

	ADS1115_ADDRESS = 0x48

	// Registers
	ADS1X15_REG_POINTER_CONVERT = 0x00
	ADS1X15_REG_POINTER_CONFIG  = 0x01

	// Config register bits
	ADS1X15_CONFIG_OS_SINGLE    = 0x8000
	ADS1X15_CONFIG_MUX_DIFF_0_1 = 0x0000
	ADS1X15_CONFIG_MUX_DIFF_0_3 = 0x1000
	ADS1X15_CONFIG_MUX_DIFF_1_3 = 0x2000
	ADS1X15_CONFIG_MUX_DIFF_2_3 = 0x3000
	ADS1X15_CONFIG_MUX_SINGLE_0 = 0x4000
	ADS1X15_CONFIG_MUX_SINGLE_1 = 0x5000
	ADS1X15_CONFIG_MUX_SINGLE_2 = 0x6000
	ADS1X15_CONFIG_MUX_SINGLE_3 = 0x7000
	ADS1X15_CONFIG_PGA_6_144V   = 0x0000
	ADS1X15_CONFIG_PGA_4_096V   = 0x0200
	ADS1X15_CONFIG_PGA_2_048V   = 0x0400
	ADS1X15_CONFIG_PGA_1_024V   = 0x0600
	ADS1X15_CONFIG_PGA_0_512V   = 0x0800
	ADS1X15_CONFIG_PGA_0_256V   = 0x0A00
	ADS1X15_CONFIG_MODE_CONTIN  = 0x0000
	ADS1X15_CONFIG_MODE_SINGLE  = 0x0100
	ADS1X15_CONFIG_DR_8SPS      = 0x0000
	ADS1X15_CONFIG_DR_16SPS     = 0x0020
	ADS1X15_CONFIG_DR_32SPS     = 0x0040
	ADS1X15_CONFIG_DR_64SPS     = 0x0060
	ADS1X15_CONFIG_DR_128SPS    = 0x0080
	ADS1X15_CONFIG_DR_250SPS    = 0x00A0
	ADS1X15_CONFIG_DR_475SPS    = 0x00C0
	ADS1X15_CONFIG_DR_860SPS    = 0x00E0
	ADS1X15_CONFIG_CMODE_TRAD   = 0x0000
	ADS1X15_CONFIG_CMODE_WINDOW = 0x0010
	ADS1X15_CONFIG_CPOL_ACTVLOW = 0x0000
	ADS1X15_CONFIG_CPOL_ACTVHI  = 0x0008
	ADS1X15_CONFIG_CLAT_NONLAT  = 0x0000
	ADS1X15_CONFIG_CLAT_LATCH   = 0x0004
	ADS1X15_CONFIG_CQUE_1CONV   = 0x0000
	ADS1X15_CONFIG_CQUE_2CONV   = 0x0001
	ADS1X15_CONFIG_CQUE_4CONV   = 0x0002
	ADS1X15_CONFIG_CQUE_NONE    = 0x0003
)

type ADS1115 struct {
	fd      int
	address uint8
}

func NewADS1115(busPath string, address uint8) (*ADS1115, error) {
	fd, err := unix.Open(busPath, unix.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %s: %w", busPath, err)
	}

	// Set I2C slave address
	if err := unix.IoctlSetInt(fd, I2C_SLAVE, int(address)); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("failed to set I2C address: %w", err)
	}

	return &ADS1115{
		fd:      fd,
		address: address,
	}, nil
}

func (a *ADS1115) Close() error {
	return unix.Close(a.fd)
}

func (a *ADS1115) writeRegister(reg uint8, value uint16) error {
	buf := []byte{reg, byte(value >> 8), byte(value & 0xFF)}
	n, err := unix.Write(a.fd, buf)
	if err != nil {
		return fmt.Errorf("failed to write register: %w", err)
	}
	if n != len(buf) {
		return fmt.Errorf("incomplete write: wrote %d bytes, expected %d", n, len(buf))
	}
	return nil
}

func (a *ADS1115) readRegister(reg uint8) (uint16, error) {
	// Write register address
	_, err := unix.Write(a.fd, []byte{reg})
	if err != nil {
		return 0, fmt.Errorf("failed to write register address: %w", err)
	}

	// Read 2 bytes
	buf := make([]byte, 2)
	n, err := unix.Read(a.fd, buf)
	if err != nil {
		return 0, fmt.Errorf("failed to read register: %w", err)
	}
	if n != 2 {
		return 0, fmt.Errorf("incomplete read: read %d bytes, expected 2", n)
	}

	return uint16(buf[0])<<8 | uint16(buf[1]), nil
}

func (a *ADS1115) ReadChannel(channel int) (int16, error) {
	if channel < 0 || channel > 3 {
		return 0, fmt.Errorf("invalid channel: %d (must be 0-3)", channel)
	}

	// Configure the ADC
	config := ADS1X15_CONFIG_OS_SINGLE |
		ADS1X15_CONFIG_MODE_SINGLE |
		ADS1X15_CONFIG_PGA_4_096V |
		ADS1X15_CONFIG_DR_128SPS |
		ADS1X15_CONFIG_CMODE_TRAD |
		ADS1X15_CONFIG_CPOL_ACTVLOW |
		ADS1X15_CONFIG_CLAT_NONLAT |
		ADS1X15_CONFIG_CQUE_NONE

	// Set the channel
	switch channel {
	case 0:
		config |= ADS1X15_CONFIG_MUX_SINGLE_0
	case 1:
		config |= ADS1X15_CONFIG_MUX_SINGLE_1
	case 2:
		config |= ADS1X15_CONFIG_MUX_SINGLE_2
	case 3:
		config |= ADS1X15_CONFIG_MUX_SINGLE_3
	}

	// Write config to start conversion
	if err := a.writeRegister(ADS1X15_REG_POINTER_CONFIG, uint16(config)); err != nil {
		return 0, err
	}

	// Wait for conversion to complete
	time.Sleep(10 * time.Millisecond)

	// Read the conversion result
	value, err := a.readRegister(ADS1X15_REG_POINTER_CONVERT)
	if err != nil {
		return 0, err
	}

	return int16(value), nil
}

func (a *ADS1115) ReadVoltage(channel int) (float64, error) {
	raw, err := a.ReadChannel(channel)
	if err != nil {
		return 0, err
	}

	// Convert to voltage (PGA = 4.096V)
	// ADS1115 is 16-bit, so full scale is ±32768
	voltage := float64(raw) * 4.096 / 32768.0
	return voltage, nil
}
