package nrf24

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	ErrPkg        = errors.New("nrf24dev")
	ErrMaxRetries = errors.New("max retransmissions reached")
	ErrTimeout    = errors.New("timeout waiting for device")
)

type (
	Address [5]byte
	Packet  [32]byte
)

func (a Address) String() string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X", a[0], a[1], a[2], a[3], a[4])
}

type (
	DataRate  byte
	PALevel   byte
	CRCLength byte
)

const (
	// DataRate250kbps represents a data rate of 250kbps
	DataRate250kbps DataRate = iota
	// DataRate1mbps represents a data rate of 1mbps
	DataRate1mbps
	// DataRate2mbps represents a data rate of 2mbps
	DataRate2mbps
)

func (d DataRate) String() string {
	switch d {
	case DataRate250kbps:
		return "250kbps"
	case DataRate1mbps:
		return "1mbps"
	case DataRate2mbps:
		return "2mbps"
	default:
		return "unknown"
	}
}

const (
	// PALevelMin represents a power amplifier level of -18dBm
	PALevelMin PALevel = iota
	// PALevelLow represents a power amplifier level of -12dBm
	PALevelLow
	// PALevelHigh represents a power amplifier level of -6dBm
	PALevelHigh
	// PALevelMax represents a power amplifier level of 0dBm
	PALevelMax
)

func (p PALevel) String() string {
	switch p {
	case PALevelMin:
		return "-18dBm"
	case PALevelLow:
		return "-12dBm"
	case PALevelHigh:
		return "-6dBm"
	case PALevelMax:
		return "0dBm"
	default:
		return "unknown"
	}
}

const (
	// CRCLengthDisabled disables CRC
	CRCLengthDisabled CRCLength = iota
	// CRCLength8 enables 8-bit CRC
	CRCLength8
	// CRCLength16 enables 16-bit CRC
	CRCLength16
)

// Status Register Bits
const (
	StatusDataReady    = 1 << 6 // RX_DR
	StatusDataSent     = 1 << 5 // TX_DS
	StatusMaxRetries   = 1 << 4 // MAX_RT
	StatusRXFIFOEmpty  = 7 << 1 // RX_P_NO (111)
	StatusTXFIFOFull   = 1 << 0 // TX_FULL
)

// --- NRF24L01 Registers/Commands/Bits ---

// NRF24 Register Addresses
const (
	_CONFIG      = 0x00
	_RF_CH       = 0x05
	_RF_SETUP    = 0x06
	_STATUS      = 0x07
	_OBSERVE_TX  = 0x08
	_RPD         = 0x09
	_RX_ADDR_P0  = 0x0A
	_RX_ADDR_P1  = 0x0B
	_TX_ADDR_REG = 0x10
	_RX_PW_P0    = 0x11 // Receive Payload Width for Data Pipe 0
	_RX_PW_P1    = 0x12 // Receive Payload Width for Data Pipe 1
	//_RX_PW_P2 = 0x13
	//_RX_PW_P3 = 0x14
	//_RX_PW_P4 = 0x15
	//_RX_PW_P5 = 0x16

	_DYNPD   = 0x1C // Dynamic Payload Register
	_FEATURE = 0x1D // Feature Register

	_W_REGISTER   = 0x20
	_R_RX_PAYLOAD = 0x61
	_W_TX_PAYLOAD = 0xA0
	_W_ACK_PAYLOAD = 0xA8 // + pipe (0-5)
	_W_TX_PAYLOAD_NOACK = 0xB0
	_FLUSH_TX     = 0xE1
	_FLUSH_RX     = 0xE2
	_NOP          = 0xFF
)

// NRF24 Register Bit Definitions
const (
	_PWR_UP  = 1 << 1
	_PRIM_RX = 1 << 0
	_RX_DR   = 1 << 6
	_TX_DS   = 1 << 5
	_MAX_RT  = 1 << 4
	_EN_CRC  = 1 << 3
	_CRCO    = 1 << 2
	// _RX_EMPTY = 1 << 0

	_SETUP_RETR = 0x04
	_EN_AA      = 0x01 // Auto Ack
	_EN_RXADDR  = 0x02
	_ERX_P0     = 1 << 0
	_ERX_P1     = 1 << 1
	_SETUP_AW   = 0x03

	_EN_DPL     = 1 << 2 // Enable Dynamic Payload Length
	_EN_ACK_PAY = 1 << 1 // Enable ACK Payload
	_EN_DYN_ACK = 1 << 0 // Enable Payload with No ACK
)

const _MAX_PAYLOAD_BYTES = 32

const _R_RX_PL_WID = 0x60

type RadioConfig struct {
	// ChannelNumber determines the specific radio frequency within the 2.4 GHz ISM band that your module will use to
	// transmit and listen for data. The range is between 0 to 124.
	// Channel numbers like 70-80 (around 2470-2480 MHz) are often good choices because they sit above the main Wi-Fi
	// spectrum used in many regions.
	ChannelNumber byte
	// RxAddr is the address of this radio module in order to receive messages.
	RxAddr Address
	// EnableDynamicPayload enables or disables dynamic packet size.
	// Defaults to false (disabled) if not provided.
	EnableDynamicPayload bool
	// PayloadSize is the payload size in bytes when EnableDynamicPayload is false.
	// Range: 1 to 32.
	// Defaults to 32 if not provided.
	PayloadSize byte
	// EnableAutoAck enables or disables hardware auto-acknowledgements.
	// Defaults to true (enabled) if not provided.
	EnableAutoAck bool
	// DataRate sets the data rate.
	// Defaults to DataRate250kbps if not provided.
	DataRate DataRate
	// PALevel sets the power amplifier level.
	// Defaults to PALevelMax if not provided.
	PALevel PALevel
	// AutoRetransmitDelay sets the auto-retransmit delay.
	// The value is in microseconds and must be a multiple of 250.
	// Range: 250 to 4000.
	// Defaults to 250 if not provided.
	AutoRetransmitDelay uint16
	// AutoRetransmitCount sets the auto-retransmit count.
	// Range: 0 to 15.
	// Defaults to 3 if not provided.
	AutoRetransmitCount byte
	// AddressWidth sets the address width.
	// Range: 3 to 5.
	// Defaults to 5 if not provided.
	AddressWidth byte
	// CRCLength sets the CRC length.
	// Defaults to CRCLength16 if not provided.
	CRCLength CRCLength
}

type HardwareConfig struct {
	RadioConfig
	// CE is the Chip Enable pin interface.
	CE Pin
	// IRQ is the Interrupt Request pin interface.
	// Optional. If not provided, polling is used.
	IRQ Pin
}

type Device struct {
	config  HardwareConfig
	conn    SPI
	irqChan chan struct{}
	nrfPort io.Closer
	mu      sync.Mutex
	scratch [33]byte // Max payload (32) + 1 status byte
}

// NewWithHardware creates and initializes a new NRF24L01 driver with the provided hardware interfaces.
func NewWithHardware(c HardwareConfig, conn SPI) (*Device, error) {
	if !c.EnableDynamicPayload && (c.PayloadSize == 0 || c.PayloadSize > 32) {
		c.PayloadSize = 32
	}
	// By default, enable auto-acknowledgements
	if !c.EnableAutoAck { // If explicitly set to false, keep it false
		c.EnableAutoAck = true
	}
	if c.DataRate == 0 {
		c.DataRate = DataRate250kbps
	}
	if c.PALevel == 0 {
		c.PALevel = PALevelMax
	}
	if c.AutoRetransmitDelay == 0 {
		c.AutoRetransmitDelay = 250
	}
	if c.AutoRetransmitCount == 0 {
		c.AutoRetransmitCount = 3
	}
	if c.AddressWidth == 0 {
		c.AddressWidth = 5
	}
	if c.AddressWidth < 3 || c.AddressWidth > 5 {
		return nil, fmt.Errorf("AddressWidth must be 3, 4, or 5")
	}
	if c.CRCLength == 0 {
		c.CRCLength = CRCLength16
	}

	if c.CE == nil {
		return nil, fmt.Errorf("CE pin not configured")
	}

	dev := &Device{
		config: c,
		conn:   conn,
	}

	// --- Hardware Initialization ---

	if dev.config.ChannelNumber > 124 {
		return nil, fmt.Errorf("channel number must be between 0 and 124")
	}

	globalLogger.Info("Initializing NRF24L01 SPI communication...")

	// Setup CE
	dev.config.CE.Out(Low)

	// Setup IRQ if provided
	if dev.config.IRQ != nil {
		dev.config.IRQ.In(PullUp)
		dev.irqChan = make(chan struct{}, 1)
		// Watch starts a goroutine that calls the handler on edge
		err := dev.config.IRQ.Watch(FallingEdge, func() {
			select {
			case dev.irqChan <- struct{}{}:
			default:
				// Channel full
			}
		})
		if err != nil {
			return nil, fmt.Errorf("failed to watch IRQ pin: %w", err)
		}
	}

	// 6. Reset and Power Up Radio
	// Ensure CE is Low (Standby-I) during configuration
	dev.setCE(false) // false = Low
	dev.writeRegister(_CONFIG, 0)
	dev.clearStatus()
	dev.flushTX()
	dev.flushRX()

	var configValue byte = _PWR_UP | _PRIM_RX // Power up and set as primary receiver
	switch dev.config.CRCLength {
	case CRCLength8:
		configValue |= _EN_CRC
	case CRCLength16:
		configValue |= _EN_CRC | _CRCO
	}
	dev.writeRegister(_CONFIG, configValue)
	time.Sleep(5 * time.Millisecond)

	// 7. Set RF parameters
	dev.writeRegister(_RF_CH, dev.config.ChannelNumber)

	// Set Address Width
	dev.writeRegister(_SETUP_AW, dev.config.AddressWidth-2)

	// Set Auto Retransmit Delay and Count
	ard := (dev.config.AutoRetransmitDelay/250 - 1) & 0x0F
	arc := dev.config.AutoRetransmitCount & 0x0F
	dev.writeRegister(_SETUP_RETR, (byte(ard)<<4)|byte(arc))

	// Set Data Rate and Power Level
	var rfSetup byte
	switch dev.config.DataRate {
	case DataRate1mbps:
		// 00001000, RF_DR_HIGH = 0, RF_DR_LOW = 0
	case DataRate2mbps:
		rfSetup |= 1 << 3 // RF_DR_HIGH
	case DataRate250kbps:
		rfSetup |= 1 << 5 // RF_DR_LOW
	}
	switch dev.config.PALevel {
	case PALevelMin:
		// 0
	case PALevelLow:
		rfSetup |= 1 << 1
	case PALevelHigh:
		rfSetup |= 2 << 1
	case PALevelMax:
		rfSetup |= 3 << 1
	}
	dev.writeRegister(_RF_SETUP, rfSetup)

	// 8. Configure Auto Ack and Pipes
	if dev.config.EnableAutoAck {
		dev.writeRegister(_EN_AA, _ERX_P0|_ERX_P1)
	} else {
		dev.writeRegister(_EN_AA, 0)
	}
	dev.writeRegister(_EN_RXADDR, _ERX_P0|_ERX_P1)

	// 9. Set Addresses and Payload Sizes
	dev.writeRegisterN(_RX_ADDR_P1, dev.config.RxAddr[:])

	// Always enable Dynamic ACK feature to support TransmitNoAck
	featureVal := byte(_EN_DYN_ACK)

	if dev.config.EnableDynamicPayload {
		// Enable dynamic payload length (DPL) and ACK payloads on all pipes
		featureVal |= _EN_DPL | _EN_ACK_PAY
		dev.writeRegister(_FEATURE, featureVal)
		// Enable dynamic payload on data pipes 0 and 1
		dev.writeRegister(_DYNPD, _ERX_P0|_ERX_P1)
	} else {
		// Disable dynamic payload features
		dev.writeRegister(_FEATURE, featureVal)
		// Disable dynamic payload on all pipes
		dev.writeRegister(_DYNPD, 0)
		// Set payload width for pipes 0 and 1
		dev.writeRegister(_RX_PW_P0, dev.config.PayloadSize)
		dev.writeRegister(_RX_PW_P1, dev.config.PayloadSize)
	}

	// 10. Verify Connection
	// Read back the channel to ensure SPI write/read is working
	readChannel := dev.readRegister(_RF_CH)
	if readChannel != dev.config.ChannelNumber {
		dev.Close()
		return nil, fmt.Errorf("failed to verify NRF24L01 connection: check wiring/power")
	}

	globalLogger.Info("NRF24L01 initialized and powered up. Ready to operate.")

	// Set CE high to start listening ONLY after full configuration
	dev.setCE(true)

	return dev, nil
}

func (d *Device) String() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	return fmt.Sprintf("NRF24L01(Channel=%d, DataRate=%s, PALevel=%s, RxAddr=%s, DynamicPayload=%v, AutoAck=%v)",
		d.config.ChannelNumber,
		d.config.DataRate,
		d.config.PALevel,
		d.config.RxAddr,
		d.config.EnableDynamicPayload,
		d.config.EnableAutoAck,
	)
}

// Close cleans up the resources used by the NRF24L01 driver.
// It powers down the radio, closes the SPI connection, and releases GPIO pins.
// This method is concurrent safe.
func (dev *Device) Close() error {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	// 1. Power down
	// We duplicate logic here to avoid deadlock if we called PowerDown() which locks
	dev.writeRegister(_CONFIG, dev.readRegister(_CONFIG)&^byte(_PWR_UP))
	globalLogger.Info("NRF24L01 powered down.")

	// 2. Clean up SPI
	if dev.nrfPort != nil {
		if err := dev.nrfPort.Close(); err != nil {
			globalLogger.Warn("Failed to close SPI port")
		}
		globalLogger.Info("SPI bus closed.")
	}

	// 3. Clean up GPIO
	if dev.config.IRQ != nil {
		dev.config.IRQ.Unwatch()
	}
	globalLogger.Info("GPIO interface closed.")

	return nil
}

// --- NRF24L01 Core Functions (SPI interaction) ---

func (d *Device) spiTransfer(len int) (status byte, response []byte) {
	// Perform full-duplex transaction on the scratch buffer
	// We use the same slice for read and write
	slice := d.scratch[:len]
	if err := d.conn.Tx(slice, slice); err != nil {
		globalLogger.Error("SPI Transfer Error")
		return 0, nil
	}

	if len > 0 {
		return d.scratch[0], d.scratch[1:len]
	}
	return 0, nil
}

func (d *Device) writeRegister(reg, val byte) {
	d.scratch[0] = _W_REGISTER | reg
	d.scratch[1] = val
	d.spiTransfer(2)
}

func (d *Device) readRegister(reg byte) byte {
	d.scratch[0] = reg
	d.scratch[1] = _NOP
	_, data := d.spiTransfer(2)
	if len(data) > 0 {
		return data[0]
	}
	return 0
}

func (d *Device) writeRegisterN(reg byte, data []byte) {
	d.scratch[0] = _W_REGISTER | reg
	copy(d.scratch[1:], data)
	d.spiTransfer(1 + len(data))
}

func (d *Device) flushTX() {
	d.scratch[0] = _FLUSH_TX
	d.spiTransfer(1)
}

func (d *Device) flushRX() {
	d.scratch[0] = _FLUSH_RX
	d.spiTransfer(1)
}

func (d *Device) clearStatus() {
	d.writeRegister(_STATUS, _RX_DR|_TX_DS|_MAX_RT)
}

func (d *Device) setCE(level bool) {
	if level {
		d.config.CE.Out(High)
	} else {
		d.config.CE.Out(Low)
	}
}

// setTargetAddress is for changing dynamically the target address to send messages to
func (d *Device) setTargetAddress(addr Address) {
	d.setCE(false) // Ensure we are in standby
	d.writeRegisterN(_TX_ADDR_REG, addr[:])

	// If using Auto-Ack (EN_AA), you MUST also update RX_ADDR_P0
	// to match TX_ADDR, because the ACK comes back to P0.
	d.writeRegisterN(_RX_ADDR_P0, addr[:])

	time.Sleep(time.Millisecond)
}

// --- NRF24L01 Configuration ---

// OpenRxPipe enables a data pipe (0-5) with the specified address.
// For Pipe 0 and 1, a full address (3-5 bytes depending on configuration) must be provided.
// For Pipes 2-5, only the LSB (1 byte) is required, as they share the high bytes with Pipe 1.
// If a full address is provided for Pipes 2-5, only the LSB is used.
// This method automatically configures the payload size/type based on the current configuration.
// Note: Pipe 0 is also used for receiving Auto-Ack packets. changing it might affect TX.
// This method is concurrent safe.
func (d *Device) OpenRxPipe(pipeID int, address []byte) error {
	if pipeID < 0 || pipeID > 5 {
		return fmt.Errorf("pipeID must be between 0 and 5")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Configure Address
	if pipeID <= 1 {
		// Pipes 0 and 1 require full address width
		if len(address) < int(d.config.AddressWidth) {
			return fmt.Errorf("pipe %d requires %d byte address", pipeID, d.config.AddressWidth)
		}
		// Write full address
		// Register is 0x0A (P0) or 0x0B (P1)
		reg := byte(_RX_ADDR_P0 + pipeID)
		d.writeRegisterN(reg, address[:d.config.AddressWidth])
	} else {
		// Pipes 2-5 require 1 byte (LSB)
		if len(address) == 0 {
			return fmt.Errorf("pipe %d requires at least 1 byte address", pipeID)
		}
		// Write LSB only
		// Register is 0x0C (P2) ... 0x0F (P5)
		reg := byte(_RX_ADDR_P0 + pipeID)
		d.writeRegister(reg, address[0])
	}

	// 2. Configure Payload
	if d.config.EnableDynamicPayload {
		// Enable DYNPD bit for this pipe
		d.writeRegister(_DYNPD, d.readRegister(_DYNPD)|(1<<pipeID))
		// Ensure feature is on (should be already from Start, but safe to check)
		if d.readRegister(_FEATURE)&_EN_DPL == 0 {
			d.writeRegister(_FEATURE, d.readRegister(_FEATURE)|_EN_DPL)
		}
	} else {
		// Disable DYNPD bit for this pipe
		d.writeRegister(_DYNPD, d.readRegister(_DYNPD)&^(1<<pipeID))
		// Set Static Payload Width
		// Register is 0x11 (P0) ... 0x16 (P5)
		reg := byte(_RX_PW_P0 + pipeID)
		d.writeRegister(reg, d.config.PayloadSize)
	}

	// 3. Enable Pipe in EN_RXADDR
	d.writeRegister(_EN_RXADDR, d.readRegister(_EN_RXADDR)|(1<<pipeID))

	// 4. Configure Auto-Ack
	if d.config.EnableAutoAck {
		d.writeRegister(_EN_AA, d.readRegister(_EN_AA)|(1<<pipeID))
	} else {
		d.writeRegister(_EN_AA, d.readRegister(_EN_AA)&^(1<<pipeID))
	}

	return nil
}

// CloseRxPipe disables a specific data pipe (0-5).
// This method is concurrent safe.
func (d *Device) CloseRxPipe(pipeID int) error {
	if pipeID < 0 || pipeID > 5 {
		return fmt.Errorf("pipeID must be between 0 and 5")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear bit in EN_RXADDR
	d.writeRegister(_EN_RXADDR, d.readRegister(_EN_RXADDR)&^(1<<pipeID))
	// Clear bit in EN_AA
	d.writeRegister(_EN_AA, d.readRegister(_EN_AA)&^(1<<pipeID))

	return nil
}

// GetRetransmissionCounters returns the number of lost packets and the number of retransmissions
// for the last sent packet.
// lostPackets: Number of packets lost (count resets when changing channel).
// currentRetries: Number of retransmissions for the latest transmission.
// This method is concurrent safe.
func (d *Device) GetRetransmissionCounters() (lostPackets byte, currentRetries byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	val := d.readRegister(_OBSERVE_TX)
	lostPackets = (val >> 4) & 0x0F
	currentRetries = val & 0x0F
	return
}

// IsCarrierDetected returns true if a carrier is detected on the current channel.
// This is useful for checking if a channel is clear before transmitting or for
// simple collision avoidance. On NRF24L01+, it detects signals > -64dBm.
// This method is concurrent safe.
func (d *Device) IsCarrierDetected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Bit 0 of RPD register
	return (d.readRegister(_RPD) & 0x01) != 0
}

// FlushTX clears the transmit FIFO buffer.
// This method is concurrent safe.
func (d *Device) FlushTX() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.flushTX()
}

// FlushRX clears the receive FIFO buffer.
// This method is concurrent safe.
func (d *Device) FlushRX() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.flushRX()
}

// GetStatus reads the current value of the STATUS register.
// This is useful for debugging or polling the radio state.
// This method is concurrent safe.
func (d *Device) GetStatus() byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readRegister(_STATUS)
}

// SetChannel changes the radio channel (frequency).
// channel must be between 0 and 124.
// This method is concurrent safe.
func (d *Device) SetChannel(channel byte) error {
	if channel > 124 {
		return fmt.Errorf("channel number must be between 0 and 124")
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	d.writeRegister(_RF_CH, channel)
	d.config.ChannelNumber = channel
	return nil
}

// SetDataRate changes the air data rate.
// This method is concurrent safe.
func (d *Device) SetDataRate(rate DataRate) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config.DataRate = rate
	return d.updateRFSetup()
}

// SetPALevel changes the power amplifier level.
// This method is concurrent safe.
func (d *Device) SetPALevel(level PALevel) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config.PALevel = level
	return d.updateRFSetup()
}

// SetAutoRetransmit configures the automatic retransmission parameters.
// delay: 250 to 4000 microseconds (must be multiple of 250).
// count: 0 to 15 retransmits.
// This method is concurrent safe.
func (d *Device) SetAutoRetransmit(delay uint16, count byte) error {
	if delay < 250 || delay > 4000 || delay%250 != 0 {
		return fmt.Errorf("delay must be between 250 and 4000 us and multiple of 250")
	}
	if count > 15 {
		return fmt.Errorf("count must be between 0 and 15")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	ard := (delay/250 - 1) & 0x0F
	arc := count & 0x0F
	d.writeRegister(_SETUP_RETR, (byte(ard)<<4)|byte(arc))

	d.config.AutoRetransmitDelay = delay
	d.config.AutoRetransmitCount = count
	return nil
}

// updateRFSetup writes the RF_SETUP register based on current config.
// Call with lock held.
func (d *Device) updateRFSetup() error {
	var rfSetup byte
	switch d.config.DataRate {
	case DataRate1mbps:
		// 00001000, RF_DR_HIGH = 0, RF_DR_LOW = 0
	case DataRate2mbps:
		rfSetup |= 1 << 3 // RF_DR_HIGH
	case DataRate250kbps:
		rfSetup |= 1 << 5 // RF_DR_LOW
	}
	switch d.config.PALevel {
	case PALevelMin:
		// 0
	case PALevelLow:
		rfSetup |= 1 << 1
	case PALevelHigh:
		rfSetup |= 2 << 1
	case PALevelMax:
		rfSetup |= 3 << 1
	}
	d.writeRegister(_RF_SETUP, rfSetup)
	return nil
}

// --- NRF24L01 Power Management ---

// PowerDown puts the NRF24L01 into Power Down mode.
// In this mode, the radio is disabled with minimal current consumption (approx. 900nA).
// This is useful for battery-powered applications when the radio is not in use.
// This method is concurrent safe.
func (d *Device) PowerDown() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.writeRegister(_CONFIG, d.readRegister(_CONFIG)&^byte(_PWR_UP))
}

// PowerUp wakes the NRF24L01 from Power Down mode.
// After calling PowerUp, it takes approximately 1.5ms for the crystal oscillator to stabilize
// before the radio can enter Standby or RX/TX modes.
// This method is concurrent safe.
func (d *Device) PowerUp() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.writeRegister(_CONFIG, d.readRegister(_CONFIG)|_PWR_UP)
	time.Sleep(2 * time.Millisecond) // Wait for oscillator stabilization
}

func (d *Device) startListening() {
	d.setCE(false)
	d.writeRegister(_CONFIG, d.readRegister(_CONFIG)|_PRIM_RX)
	d.setCE(true)
	time.Sleep(130 * time.Microsecond)
	d.clearStatus()
	d.flushRX()
}

func (d *Device) stopListening() {
	d.setCE(false)
	d.writeRegister(_CONFIG, d.readRegister(_CONFIG) & ^byte(_PRIM_RX))
}

// --- NRF24L01 Read/Write ---

func (d *Device) available() bool {
	return ((d.readRegister(_STATUS) >> 1) & 0x07) != 7
}

func (d *Device) getDynamicPayloadSize() byte {
	// Send command 0x60 and a NOP to get the 1-byte response
	d.scratch[0] = _R_RX_PL_WID
	d.scratch[1] = _NOP
	_, data := d.spiTransfer(2)
	if len(data) > 0 {
		if data[0] > 32 { // Hardware bug/noise check
			d.flushRX()
			return 0
		}
		return data[0]
	}
	return 0
}

func (d *Device) readDynamic() ([]byte, bool) {
	if !d.available() {
		return nil, false
	}

	// 1. Ask the radio how big the current packet is
	size := d.getDynamicPayloadSize()
	if size == 0 {
		// If the radio says data is available but the size is 0, it's either an empty packet
		// or a glitch. In either case, we must remove it from the FIFO or we will loop forever.
		// Since we can't "read" 0 bytes to advance the FIFO, we flush.
		d.flushRX()
		d.clearStatus()
		return nil, false
	}

	// 2. Read exactly that many bytes
	d.scratch[0] = _R_RX_PAYLOAD
	for i := 1; i <= int(size); i++ {
		d.scratch[i] = _NOP
	}

	_, data := d.spiTransfer(int(size) + 1)

	// Copy result to safe buffer BEFORE calling clearStatus which reuses scratch
	result := make([]byte, len(data))
	copy(result, data)

	d.clearStatus()
	
	return result, true
}

func (d *Device) readFixedPayload() ([]byte, bool) {
	if !d.available() {
		return nil, false
	}

	size := int(d.config.PayloadSize)
	// Read exactly size bytes
	d.scratch[0] = _R_RX_PAYLOAD
	for i := 1; i <= size; i++ {
		d.scratch[i] = _NOP
	}

	_, data := d.spiTransfer(size + 1)

	// Copy result to safe buffer BEFORE calling clearStatus which reuses scratch
	result := make([]byte, len(data))
	copy(result, data)

	d.clearStatus()

	return result, true
}

func (d *Device) write(data []byte, noAck bool) error {
	d.stopListening()

	cmdPrefix := byte(_W_TX_PAYLOAD)
	if noAck {
		cmdPrefix = _W_TX_PAYLOAD_NOACK
	}

	d.scratch[0] = cmdPrefix
	
	if d.config.EnableDynamicPayload {
		copy(d.scratch[1:], data)
		d.spiTransfer(1 + len(data))
	} else {
		// For fixed payload, ensure it's always d.config.PayloadSize
		// We need to clear the scratch buffer first to ensure padding is 0
		size := int(d.config.PayloadSize)
		for i := 1; i <= size; i++ {
			d.scratch[i] = 0
		}
		copy(d.scratch[1:], data) // Copy up to len(data), rest will be zeros
		d.spiTransfer(1 + size)
	}

	d.setCE(true)
	time.Sleep(15 * time.Microsecond)
	d.setCE(false)

	// Calculate a safe timeout based on retransmit settings.
	// (Delay * Count) is the maximum time the hardware will spend retrying.
	// We add a 50ms safety buffer for SPI communication and OS scheduling.
	timeoutDuration := time.Duration(d.config.AutoRetransmitDelay)*time.Duration(d.config.AutoRetransmitCount)*time.Microsecond + 50*time.Millisecond
	timeout := time.After(timeoutDuration)

	for {
		select {
		case <-timeout:
			d.clearStatus()
			d.flushTX()
			return fmt.Errorf("%w: %w", ErrPkg, ErrTimeout)
		default:
			status := d.readRegister(_STATUS)
			if status&(_TX_DS|_MAX_RT) != 0 {
				d.clearStatus()
				if status&_MAX_RT != 0 {
					d.flushTX()
					return fmt.Errorf("%w: %w", ErrPkg, ErrMaxRetries)
				}
				return nil
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// Transmit sends a message.
// This method is concurrent safe.
// It returns an error if you are trying to send a message bigger than the max payload size.
func (dev *Device) Transmit(destAddr Address, p []byte) error {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	dev.stopListening()

	limit := int(_MAX_PAYLOAD_BYTES)
	if !dev.config.EnableDynamicPayload {
		limit = int(dev.config.PayloadSize)
	}

	if len(p) > limit {
		return fmt.Errorf("%w: payload too large (%d bytes), limit is %d", ErrPkg, len(p), limit)
	}

	dev.setTargetAddress(destAddr)

	if err := dev.write(p, false); err != nil {
		dev.startListening()
		return fmt.Errorf("failed to send data: %w", err)
	}

	dev.startListening()
	return nil
}

// WriteAckPayload writes a payload to be transmitted with the ACK packet.
// This allows for bi-directional communication where the receiver replies to the
// transmitter instantly.
// pipeID: The pipe number (0-5) that will transmit this ACK payload.
// data: The payload data (1-32 bytes).
// Note: EnableDynamicPayload must be true for this feature to work.
// This method is concurrent safe.
func (d *Device) WriteAckPayload(pipeID int, data []byte) error {
	if !d.config.EnableAutoAck {
		return fmt.Errorf("AckPayloads require EnableAutoAck to be true")
	}
	if !d.config.EnableDynamicPayload {
		return fmt.Errorf("AckPayloads require EnableDynamicPayload to be true")
	}
	if pipeID < 0 || pipeID > 5 {
		return fmt.Errorf("pipeID must be between 0 and 5")
	}
	if len(data) > _MAX_PAYLOAD_BYTES {
		return fmt.Errorf("%w: payload too large (%d bytes), max is %d", ErrPkg, len(data), _MAX_PAYLOAD_BYTES)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.scratch[0] = _W_ACK_PAYLOAD | byte(pipeID)
	copy(d.scratch[1:], data)
	d.spiTransfer(1 + len(data))
	
	return nil
}

// TransmitNoAck sends a message with a "No Acknowledgement" flag in the packet header.
// Unlike a regular Transmit with EnableAutoAck set to false, this method explicitly tells
// the receiver NOT to send an ACK packet. This is the preferred method for broadcasting
// to multiple receivers or for high-speed, low-reliability data as it prevents receivers
// from wasting power and airtime sending ACKs that the transmitter isn't listening for.
// This method is concurrent safe.
func (dev *Device) TransmitNoAck(destAddr Address, p []byte) error {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	dev.stopListening()

	limit := int(_MAX_PAYLOAD_BYTES)
	if !dev.config.EnableDynamicPayload {
		limit = int(dev.config.PayloadSize)
	}

	if len(p) > limit {
		return fmt.Errorf("%w: payload too large (%d bytes), limit is %d", ErrPkg, len(p), limit)
	}

	dev.setTargetAddress(destAddr)

	if err := dev.write(p, true); err != nil {
		dev.startListening()
		return fmt.Errorf("failed to send data: %w", err)
	}

	dev.startListening()
	return nil
}

// SetAddressWidth sets the address width (3, 4, or 5 bytes).
// This method is concurrent safe.
func (d *Device) SetAddressWidth(width byte) error {
	if width < 3 || width > 5 {
		return fmt.Errorf("AddressWidth must be 3, 4, or 5")
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	d.writeRegister(_SETUP_AW, width-2)
	d.config.AddressWidth = width
	return nil
}

// Receive tries to receive a packet from the NRF24L01 module.
// This method is non-blocking and assumes the radio has been put into receive mode (e.g., by calling Start).
// It returns the packet and true if a message is available, otherwise returns an empty packet and false.
// This method is concurrent safe.
func (dev *Device) Receive() ([]byte, bool) {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.config.EnableDynamicPayload {
		payload, ok := dev.readDynamic()
		if ok {
			return payload, true
		}
	} else {
		payload, ok := dev.readFixedPayload()
		if ok {
			return payload, true
		}
	}
	return nil, false
}

// WaitForInterrupt blocks until the IRQ pin goes low (active) or the context is cancelled.
// It returns the content of the STATUS register.
// If IrqPin is not configured, it returns an error.
// This method is concurrent safe.
func (d *Device) WaitForInterrupt(ctx context.Context) (byte, error) {
	if d.config.IRQ == nil {
		return 0, fmt.Errorf("IRQ pin not configured")
	}

	// Check if interrupt is already active (low = false)
	if d.config.IRQ.Read() == Low {
		d.mu.Lock()
		status := d.readRegister(_STATUS)
		d.mu.Unlock()
		return status, nil
	}

	// Wait for signal from the Watch callback or context
	select {
	case <-d.irqChan:
		d.mu.Lock()
		status := d.readRegister(_STATUS)
		d.mu.Unlock()
		return status, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// ReceiveBlocking waits for a packet to arrive or for the context to be cancelled.
// It blocks efficiently using the IRQ pin if configured, or falls back to polling.
// This method is concurrent safe.
func (d *Device) ReceiveBlocking(ctx context.Context) ([]byte, error) {
	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 1. Check if data is already available
		data, ok := d.Receive() // Receive is already thread-safe
		if ok {
			return data, nil
		}

		// 2. Wait for data
		if d.config.IRQ != nil {
			status, err := d.WaitForInterrupt(ctx)
			if err != nil {
				return nil, err
			}
			
			// Check if it was RX_DR (Data Ready)
			if status&_RX_DR != 0 {
				// Loop again to call Receive() and fetch data
				continue
			}
			// If it was another interrupt (e.g. MaxRT), clear it so we don't get stuck
			d.clearInterrupts(status)
		} else {
			// Polling fallback
			// Check context cancellation before sleeping
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// Fall through
			}
			
			// Use Sleep instead of time.NewTimer/time.After to avoid heap allocation overhead
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// clearInterrupts clears the specified interrupt flags in the STATUS register.
// This is concurrent safe.
func (d *Device) clearInterrupts(flags byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Write 1 to clear bits
	d.writeRegister(_STATUS, flags)
}

// Ping sends a ping to a specific address.
// This method is concurrent safe.
func (d *Device) Ping(_ context.Context, addr Address) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Set the target address
	d.setTargetAddress(Address(addr))

	// 2. Send a single "null" byte (0x00) as a ping
	// Your existing write() function returns true only if TX_DS (Data Sent)
	// is set, which requires an ACK when EN_AA is enabled.
	err := d.write([]byte{0x00}, false)

	if err == nil {
		globalLogger.Info("Ping Success")
		return true, nil
	}
	
	globalLogger.Info("Ping Failed")
	return false, nil
}
