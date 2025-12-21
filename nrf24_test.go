package nrf24

import (
	"bytes"
	"testing"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

// --- Mocks ---

type mockPin struct {
	mode   string
	level  gpio.Level
	pullUp bool
}

func (m *mockPin) Out(l gpio.Level) error {
	m.mode = "output"
	m.level = l
	return nil
}

func (m *mockPin) In(pull gpio.Pull, edge gpio.Edge) error {
	m.mode = "input"
	if pull == gpio.PullUp {
		m.pullUp = true
	}
	return nil
}

func (m *mockPin) Read() gpio.Level { return m.level }

func (m *mockPin) Watch(edge gpio.Edge, handler func()) error { return nil }
func (m *mockPin) Unwatch() error                         { return nil }

type mockSPIConn struct {
	tx      []byte
	rxQueue [][]byte // Queue of responses to return for subsequent Tx calls
}

func (m *mockSPIConn) Tx(w, r []byte) error {
	m.tx = append(m.tx, w...)
	
	if len(m.rxQueue) > 0 {
		// Pop the next response
		nextRx := m.rxQueue[0]
		m.rxQueue = m.rxQueue[1:]
		
		// Copy min(len(r), len(nextRx))
		n := len(r)
		if len(nextRx) < n {
			n = len(nextRx)
		}
		copy(r, nextRx[:n])
	}
	return nil
}

func (m *mockSPIConn) queueRx(data []byte) {
	m.rxQueue = append(m.rxQueue, data)
}

func (m *mockSPIConn) Duplex() conn.Duplex { return conn.Full }
func (m *mockSPIConn) TxPackets(p []spi.Packet) error { return nil }
func (m *mockSPIConn) String() string { return "mockSPI" }
func (m *mockSPIConn) Close() error { return nil }
func (m *mockSPIConn) Connect(f physic.Frequency, mode spi.Mode, bits int) (spi.Conn, error) {
	return m, nil
}
func (m *mockSPIConn) LimitSpeed(f physic.Frequency) error { return nil }


// --- Tests ---

func TestInitialization(t *testing.T) {
	// Setup Mocks
	mockSPI := &mockSPIConn{}
	mockCE := &mockPin{}
	mockIRQ := &mockPin{}

	// Config
	cfg := Config{
		ChannelNumber: 76,
		RxAddr:        Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
		Logger:        &nopLogger{}, // Silence logs
	}

	// Call newDriver
	dev, err := newDriver(cfg, mockSPI, mockCE, mockIRQ)
	if err != nil {
		t.Fatalf("newDriver failed: %v", err)
	}

	// Verify CE was set to Output and started Low
	if mockCE.mode != "output" {
		t.Errorf("Expected CE pin to be output, got %s", mockCE.mode)
	}
	
	// Verify SPI commands
	// We look for specific register writes that should happen during init.
	// Example: Writing Channel 76 to register _RF_CH (0x05)
	// Write command is 0x20 | reg. So 0x25.
	
	expectedOp := []byte{0x20 | _RF_CH, 76}
	if !bytes.Contains(mockSPI.tx, expectedOp) {
		t.Errorf("Expected SPI write to RF_CH (0x%X), but not found in TX buffer: %X", expectedOp, mockSPI.tx)
	}

	// Verify Power Up
	// _CONFIG (0x00) should be written with _PWR_UP (bit 1) and _PRIM_RX (bit 0) set.
	// Default CRCLength16 sets _EN_CRC (bit 3) and _CRCO (bit 2).
	// Value: 0000 1111 = 0x0F.
	// Command: 0x20 | 0x00 = 0x20. Payload: 0x0F.
	expectedPowerUp := []byte{0x20 | _CONFIG, 0x0F}
	if !bytes.Contains(mockSPI.tx, expectedPowerUp) {
		t.Errorf("Expected SPI write to CONFIG for PowerUp (0x%X), but not found: %X", expectedPowerUp, mockSPI.tx)
	}

	// Verify CE is High at the end (Listening)
	if mockCE.level != gpio.High {
		t.Errorf("Expected CE to be High (Listening) after init, got %v", mockCE.level)
	}

	dev.Close()
}

func TestTransmit(t *testing.T) {
	mockSPI := &mockSPIConn{}
	mockCE := &mockPin{}
	cfg := Config{Logger: &nopLogger{}}
	
	dev, _ := newDriver(cfg, mockSPI, mockCE, nil)
	
	// Reset TX buffer to clear init commands
	mockSPI.tx = nil

	// Simulation sequence for Transmit:
	// 1. stopListening() -> read(_CONFIG), write(_CONFIG)
	// 2. setTargetAddress() -> write(_TX_ADDR), write(_RX_ADDR_P0)
	// 3. write() -> stopListening() again -> read(_CONFIG), write(_CONFIG)
	// 4. write() -> W_TX_PAYLOAD
	// 5. write() loop -> read(_STATUS) -> MUST RETURN TX_DS (0x20)
	
	// Queue dummy responses for the setup steps
	mockSPI.queueRx([]byte{0, 0}) // 1. Read Config
	mockSPI.queueRx([]byte{0})    // 2. Write Config
	mockSPI.queueRx([]byte{0})    // 3. Write TX Addr
	mockSPI.queueRx([]byte{0})    // 4. Write RX Addr
	mockSPI.queueRx([]byte{0, 0}) // 5. Read Config (write calls stopListening again)
	mockSPI.queueRx([]byte{0})    // 6. Write Config
	mockSPI.queueRx([]byte{0})    // 7. Write Payload
	
	// Queue the SUCCESS status
	mockSPI.queueRx([]byte{0, 0x20}) // 8. Read Status (returns 0x20)

	// Queue responses for the cleanup (clearStatus, startListening)
	// If we don't queue these, they get zeros, which is fine, but good to be explicit or loose.
	// We'll leave them to default to zeros.

	payload := []byte("hello")
	addr := Address{0x01, 0x02, 0x03, 0x04, 0x05}
	
	err := dev.Transmit(addr, payload)
	if err != nil {
		t.Fatalf("Transmit failed: %v", err)
	}

	// Verify W_TX_PAYLOAD command sent
	// Command: 0xA0 (_W_TX_PAYLOAD)
	expectedCmd := []byte{0xA0}
	expectedCmd = append(expectedCmd, payload...)
	// Note: Since fixed payload is default (32 bytes), padding will occur.
	// Start checks the prefix.
	if !bytes.Contains(mockSPI.tx, []byte{0xA0, 'h', 'e', 'l', 'l', 'o'}) {
		t.Errorf("Expected W_TX_PAYLOAD with data, got TX trace: %X", mockSPI.tx)
	}
}

func TestTransmitFailure(t *testing.T) {
	mockSPI := &mockSPIConn{}
	mockCE := &mockPin{}
	cfg := Config{Logger: &nopLogger{}}
	dev, _ := newDriver(cfg, mockSPI, mockCE, nil)

	// Test Case 1: Max Retries reached (_MAX_RT = 0x10)
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	// Dummies for Transmit setup
	for i := 0; i < 7; i++ {
		mockSPI.queueRx([]byte{0})
	}
	// Return MAX_RT status
	mockSPI.queueRx([]byte{0x00, 0x10})

	err := dev.Transmit(Address{1, 2, 3, 4, 5}, []byte("fail"))
	if err == nil {
		t.Fatal("Expected error on MaxRetries, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("max retransmissions reached")) {
		t.Errorf("Expected MaxRetries error message, got: %v", err)
	}

	// Test Case 2: Timeout (Device unresponsive - returns 0)
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	// Dummies for Transmit setup
	for i := 0; i < 7; i++ {
		mockSPI.queueRx([]byte{0})
	}
	// Note: If no responses queued, mock returns 0.
	// We'll let it time out in real time (short timeout in driver).
	// Actually, the loop sleeps 1ms, so 25ms timeout is fast for tests.
	err = dev.Transmit(Address{1, 2, 3, 4, 5}, []byte("timeout"))
	if err == nil {
		t.Fatal("Expected error on Timeout, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("timeout waiting for device")) {
		t.Errorf("Expected Timeout error message, got: %v", err)
	}
}

func TestReceive(t *testing.T) {
	mockSPI := &mockSPIConn{}
	mockCE := &mockPin{}
	// Enable dynamic payload for simpler testing (no padding check needed)
	cfg := Config{
		Logger:               &nopLogger{},
		EnableDynamicPayload: true,
	}
	
	dev, _ := newDriver(cfg, mockSPI, mockCE, nil)
	mockSPI.tx = nil

	// Simulation for Receive():
	// 1. available() -> reads STATUS.
	//    Expects: _RX_DR (bit 6) = 0x40. Pipe 0 (000). 
	//    So status = 0x40.
	//    Cmd: [STATUS, NOP] -> Resp: [0, 0x40] (Wait, readRegister returns byte 1)
	mockSPI.queueRx([]byte{0x00, 0x40})

	// 2. readDynamic() -> getDynamicPayloadSize()
	//    Cmd: [R_RX_PL_WID, NOP] -> Resp: [Status, Size]
	//    Let's say payload is "world" (5 bytes).
	mockSPI.queueRx([]byte{0x40, 0x05})

	// 3. readDynamic() -> read payload
	//    Cmd: [R_RX_PAYLOAD, NOP, NOP, NOP, NOP, NOP]
	//    Resp: [Status, 'w', 'o', 'r', 'l', 'd']
	mockSPI.queueRx([]byte{0x40, 'w', 'o', 'r', 'l', 'd'})
	
	// 4. clearStatus() -> writeRegister(STATUS, ...)
	//    Returns status (ignored).
	mockSPI.queueRx([]byte{0x00, 0x00})

	data, found := dev.Receive()
	if !found {
		t.Fatal("Expected Receive to return true")
	}
	if string(data) != "world" {
		t.Errorf("Expected payload 'world', got '%s'", string(data))
	}
}

func TestConfiguration(t *testing.T) {
	mockSPI := &mockSPIConn{}
	cfg := Config{Logger: &nopLogger{}}
	dev, _ := newDriver(cfg, mockSPI, &mockPin{}, nil)

	// Test SetChannel
	mockSPI.tx = nil
	dev.SetChannel(88)
	// Write 88 to RF_CH (0x05) -> [0x25, 88]
	if !bytes.Contains(mockSPI.tx, []byte{0x25, 88}) {
		t.Errorf("SetChannel(88) didn't write to SPI correctly: %X", mockSPI.tx)
	}

	// Test SetDataRate
	mockSPI.tx = nil
	dev.SetDataRate(DataRate2mbps)
	// RF_SETUP (0x06). 2mbps sets RF_DR_HIGH (bit 3) = 0x08.
	// PALevelMax (default) sets bits 2:1 = 11 = 0x06.
	// Total: 0x0E.
	if !bytes.Contains(mockSPI.tx, []byte{0x26, 0x0E}) {
		t.Errorf("SetDataRate didn't write to SPI correctly: %X", mockSPI.tx)
	}
}

func TestOpenRxPipe(t *testing.T) {
	mockSPI := &mockSPIConn{}
	cfg := Config{
		Logger:        &nopLogger{},
		EnableAutoAck: true,
	}
	dev, _ := newDriver(cfg, mockSPI, &mockPin{}, nil)

	// Test Pipe 1 (Full Address)
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	// OpenRxPipe reads: DYNPD, EN_RXADDR, EN_AA
	mockSPI.queueRx([]byte{0, 0}) // Read DYNPD
	mockSPI.queueRx([]byte{0, 0}) // Read EN_RXADDR
	mockSPI.queueRx([]byte{0, 0}) // Read EN_AA
	
	addr := []byte{0xA1, 0xA2, 0xA3, 0xA4, 0xA5}
	dev.OpenRxPipe(1, addr)
	
	// Should write full address to _RX_ADDR_P1 (0x0B). Command 0x2B.
	if !bytes.Contains(mockSPI.tx, append([]byte{0x2B}, addr...)) {
		t.Errorf("OpenRxPipe(1) didn't write full address correctly: %X", mockSPI.tx)
	}

	// Test Pipe 2 (LSB only)
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	// OpenRxPipe reads: DYNPD, EN_RXADDR, EN_AA
	mockSPI.queueRx([]byte{0, 0}) // Read DYNPD
	mockSPI.queueRx([]byte{0, 0}) // Read EN_RXADDR
	mockSPI.queueRx([]byte{0, 0}) // Read EN_AA
	
	dev.OpenRxPipe(2, []byte{0xCC})
	
	// Should write LSB (0xCC) to _RX_ADDR_P2 (0x0C). Command 0x2C.
	if !bytes.Contains(mockSPI.tx, []byte{0x2C, 0xCC}) {
		t.Errorf("OpenRxPipe(2) didn't write LSB correctly: %X", mockSPI.tx)
	}
	// Should enable bit 2 in EN_RXADDR (0x02) and EN_AA (0x01)
	// Write Command for EN_RXADDR: 0x22. Value: 0x04 (bit 2).
	if !bytes.Contains(mockSPI.tx, []byte{0x22, 0x04}) {
		t.Errorf("OpenRxPipe(2) didn't enable pipe in EN_RXADDR: %X", mockSPI.tx)
	}
}

func TestReceiveFixed(t *testing.T) {
	mockSPI := &mockSPIConn{}
	cfg := Config{
		Logger:               &nopLogger{},
		EnableDynamicPayload: false,
		PayloadSize:          5,
	}
	dev, _ := newDriver(cfg, mockSPI, &mockPin{}, nil)
	mockSPI.tx = nil

	// Simulation for Receive() with Fixed Payload:
	// 1. available() -> reads STATUS. Expects _RX_DR (0x40).
	mockSPI.queueRx([]byte{0x00, 0x40})

	// 2. readFixedPayload() -> R_RX_PAYLOAD.
	//    Cmd: [R_RX_PAYLOAD, NOP, NOP, NOP, NOP, NOP] (Length 5)
	//    Resp: [Status, 'h', 'e', 'l', 'l', 'o']
	mockSPI.queueRx([]byte{0x40, 'h', 'e', 'l', 'l', 'o'})
	
	// 3. clearStatus() -> writeRegister(STATUS, ...)
	mockSPI.queueRx([]byte{0x00, 0x00})

	data, found := dev.Receive()
	if !found {
		t.Fatal("Expected Receive to return true")
	}
	if string(data) != "hello" {
		t.Errorf("Expected payload 'hello', got '%s'", string(data))
	}
}

func TestCloseRxPipe(t *testing.T) {
	mockSPI := &mockSPIConn{}
	cfg := Config{Logger: &nopLogger{}}
	dev, _ := newDriver(cfg, mockSPI, &mockPin{}, nil)
	mockSPI.tx = nil
	mockSPI.rxQueue = nil

	// Pre-condition: Assume bits are set. readRegister returns values with bits set.
	// CloseRxPipe reads EN_RXADDR and EN_AA.
	// We simulate they are currently 0xFF (all pipes enabled).
	// But we also need to account for the writes consuming from the queue!
	
	mockSPI.queueRx([]byte{0, 0xFF}) // 1. Read EN_RXADDR -> returns 0xFF
	mockSPI.queueRx([]byte{0})       // 2. Write EN_RXADDR -> dummy
	mockSPI.queueRx([]byte{0, 0xFF}) // 3. Read EN_AA -> returns 0xFF
	mockSPI.queueRx([]byte{0})       // 4. Write EN_AA -> dummy
	
	dev.CloseRxPipe(2)
	
	// Should clear bit 2 (0x04). Result 0xFB.
	// Write EN_RXADDR (0x22) -> 0xFB
	if !bytes.Contains(mockSPI.tx, []byte{0x22, 0xFB}) {
		t.Errorf("CloseRxPipe(2) didn't clear EN_RXADDR correctly: %X", mockSPI.tx)
	}
	// Write EN_AA (0x21) -> 0xFB
	if !bytes.Contains(mockSPI.tx, []byte{0x21, 0xFB}) {
		t.Errorf("CloseRxPipe(2) didn't clear EN_AA correctly: %X", mockSPI.tx)
	}
}

func TestDiagnostics(t *testing.T) {
	mockSPI := &mockSPIConn{}
	cfg := Config{Logger: &nopLogger{}}
	dev, _ := newDriver(cfg, mockSPI, &mockPin{}, nil)
	
	// 1. FlushTX
	mockSPI.tx = nil
	dev.FlushTX()
	if !bytes.Contains(mockSPI.tx, []byte{0xE1}) { // _FLUSH_TX
		t.Errorf("FlushTX sent wrong command: %X", mockSPI.tx)
	}

	// 2. FlushRX
	mockSPI.tx = nil
	dev.FlushRX()
	if !bytes.Contains(mockSPI.tx, []byte{0xE2}) { // _FLUSH_RX
		t.Errorf("FlushRX sent wrong command: %X", mockSPI.tx)
	}

	// 3. GetStatus
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	mockSPI.queueRx([]byte{0x00, 0x0E}) // Return 0x0E (RX_EMPTY 1110)
	status := dev.GetStatus()
	if status != 0x0E {
		t.Errorf("GetStatus expected 0x0E, got 0x%X", status)
	}

	// 4. GetRetransmissionCounters
	// Register OBSERVE_TX (0x08). 
	// Value: High nibble = Lost Packets, Low nibble = Retries.
	// Let's return 0xF3 (15 lost, 3 retries).
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	mockSPI.queueRx([]byte{0, 0xF3})
	lost, retries := dev.GetRetransmissionCounters()
	if lost != 15 || retries != 3 {
		t.Errorf("GetRetransmissionCounters expected (15, 3), got (%d, %d)", lost, retries)
	}

	// 5. IsCarrierDetected
	// Register RPD (0x09). Bit 0 is flag.
	// Return 1.
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	mockSPI.queueRx([]byte{0, 0x01})
	if !dev.IsCarrierDetected() {
		t.Error("IsCarrierDetected expected true")
	}

	// 6. TransmitNoAck
	// Verify it sends command 0xB0 instead of 0xA0
	mockSPI.tx = nil
	mockSPI.rxQueue = nil
	// Queue 7 dummies for setup (stopListening, setTargetAddress, write setup) + 1 for status
	for i := 0; i < 7; i++ {
		mockSPI.queueRx([]byte{0})
	}
	mockSPI.queueRx([]byte{0, 0x20}) // Status Success

	dev.TransmitNoAck(Address{1}, []byte("hi"))
	
	// Check for command 0xB0
	if !bytes.Contains(mockSPI.tx, []byte{0xB0, 'h', 'i'}) {
		t.Errorf("TransmitNoAck didn't send 0xB0 command. TX: %X", mockSPI.tx)
	}
}
