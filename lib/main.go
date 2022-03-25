package sds011

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"go.bug.st/serial"
)

const (
	version       = "0.2.1"
	ff            = 0xFF
	baud          = 9600   // Specified baud rate
	head          = 0xAA   // Start of message
	tail          = 0xAB   // End of message
	cmdid         = 0xB4   // Command indicator
	ack           = 0xC5   // Command acknowledgment response
	meas          = 0xC0   // Measurement query response
	anyid         = 0xFFFF // Any ID matches
	get      byte = 0      // Ask to get/read value
	set      byte = 1      // Ask to set/write value
	sleep    byte = 0      // Sleep state
	wake     byte = 1      // Wake state
	active   byte = 0      // Sensor periodically measures automatically: Channel()
	query    byte = 1      // Sensor only measures when requested: Query()
	modeCmd  byte = 2
	queryCmd byte = 4
	idCmd    byte = 5
	stateCmd byte = 6
	fwCmd    byte = 7
	dutyCmd  byte = 8
)

var (
	Device = "/dev/ttyUSB0"
)

type sensor struct {
	conn         Connection
	measurements []chan Measurement
	Id           uint16  // Tracking
	Firmware     string  // Tracking
	Mode         byte    // Tracking
	State        byte    // Tracking
	Duty         byte    // Tracking
	PM2_5        float32 // Tracking
	PM10         float32 // Tracking
	Debug        bool    // If true: display message passing
	Track        bool    // If true: get data from tracked internal state
}

type Measurement struct {
	ID    uint16
	PM2_5 float32
	PM10  float32
}

type Connection interface {
	Write(message []byte) (int, error)
	Read(response []byte) (int, error)
}

func checksum(bytes []byte) byte {
	// Message checksums are over all except the first 2 and last 2 bytes
	sum := byte(0)
	for i := range bytes {
		sum += bytes[i]
	}
	return sum & ff
}

func split(halfword uint16) []byte {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, halfword)
	return data
}

func Sensor(devs ...string) *sensor {
	if len(devs) > 0 {
		Device = devs[0]
	}
	_, err := os.Stat(Device)
	if err != nil {
		fmt.Println("Device " + Device + " not accessible")
		os.Exit(1)
	}
	devmode := &serial.Mode{BaudRate: baud}
	port, err := serial.Open(Device, devmode)
	if err != nil {
		fmt.Println("Can't open port on device " + Device)
		os.Exit(2)
	}
	debug := false
	if os.Getenv("SDS011_DEBUG") == "1" {
		debug = true
	}
	track := false
	if os.Getenv("SDS011_TRACK") == "1" {
		track = true
	}
	return &sensor{
		conn:  port,
		Id:    anyid,
		Mode:  ff,
		State: ff,
		Duty:  ff,
		Debug: debug,
		Track: track,
	}
}

func (sensor *sensor) read() []byte {
	// Scan for head on connection
	responseHead := make([]byte, 1)
	_, err := sensor.conn.Read(responseHead)
	for err == nil && responseHead[0] != head {
		_, err = sensor.conn.Read(responseHead)
	}
	response := make([]byte, 10)
	response[0] = head
	_, err = sensor.conn.Read(response[1:])
	if sensor.Debug {
		fmt.Printf("[I: %x]\n", response)
	}
	for err != nil {
		_, err = sensor.conn.Read(response[1:])
		if sensor.Debug {
			fmt.Printf("[I: %x]\n", response)
		}
	}
	return response
}

func (sensor *sensor) write(command byte, args []byte) {
	var data = make([]byte, 15)
	data[0] = command
	copy(data[1:15], args)
	// If no specific DeviceId is set in the command, target any
	if data[13] == 0 && data[14] == 0 {
		data[13] = ff
		data[14] = ff
	}
	var message bytes.Buffer
	message.Write([]byte{head, cmdid})
	message.Write(data)
	message.WriteByte(checksum(data))
	message.WriteByte(tail)
	if sensor.Debug {
		fmt.Printf("[O: %x]\n", message.Bytes())
	}
	_, err := sensor.conn.Write(message.Bytes())
	for err != nil {
		_, err = sensor.conn.Write(message.Bytes())
	}
}

func (sensor *sensor) Poll() *Measurement {
	response := sensor.read()
	for response[0] != head || response[1] != meas || response[9] != tail || checksum(response[2:8]) != response[8] {
		response = sensor.read()
	}
	sensor.Id = binary.LittleEndian.Uint16(response[6:8])
	sensor.PM2_5 = float32(binary.LittleEndian.Uint16(response[2:4])) / 10
	sensor.PM10 = float32(binary.LittleEndian.Uint16(response[4:6])) / 10
	measurement := Measurement{
		ID:    sensor.Id,
		PM2_5: sensor.PM2_5,
		PM10:  sensor.PM10,
	}
	return &measurement
}

func (sensor *sensor) Query() *Measurement {
	sensor.write(queryCmd, nil)
	return sensor.Poll()
}

func (sensor *sensor) Channel() chan Measurement {
	ch := make(chan Measurement)
	var found bool
	for i := range sensor.measurements {
		if sensor.measurements[i] == ch {
			found = true
			break
		}
	}
	if !found {
		sensor.measurements = append(sensor.measurements, ch)
	}
	go func(handlers []chan Measurement) {
		for true {
			measurement := sensor.Poll()
			ch<- *measurement
		}
	}(sensor.measurements)
	return ch
}

func (sensor *sensor) Get(command byte, args []byte) byte {
	// command: modeCmd, stateCmd, dutyCmd
	sensor.write(command, []byte{get})
	response := sensor.read()
	for response[0] != head || response[1] != ack || response[2] != command || response[9] != tail || checksum(response[2:8]) != response[8] {
		response = sensor.read()
	}
	sensor.Id = binary.LittleEndian.Uint16(response[6:8])
	switch command {
	case modeCmd:
		sensor.Mode = response[4]
	case stateCmd:
		sensor.State = response[4]
	case dutyCmd:
		sensor.Duty = response[4]
	}
	return response[4]
}

func (sensor *sensor) Set(command byte, args []byte) {
	// command: modeCmd, stateCmd, dutyCmd
	sensor.write(command, args)
	if command == stateCmd && args[0] == set && args[1] == sleep {
		sensor.State = sleep
		return
	}

	response := sensor.read()
	for response[0] != head || response[1] != ack || response[2] != command || response[9] != tail || checksum(response[2:8]) != response[8] {
		response = sensor.read()
	}
	sensor.Id = binary.LittleEndian.Uint16(response[6:8])
	switch command {
	case modeCmd:
		sensor.Mode = response[4]
	case stateCmd:
		sensor.State = response[4]
	case dutyCmd:
		sensor.Duty = response[4]
	}
}

func (sensor *sensor) GetMode() byte {
	if sensor.Track && sensor.Mode != ff {
		return sensor.Mode
	}
	return sensor.Get(modeCmd, nil)
}

func (sensor *sensor) SetActive() {
	sensor.Set(modeCmd, []byte{set, active})
}

func (sensor *sensor) SetQuery() {
	sensor.Set(modeCmd, []byte{set, query})
}

func (sensor *sensor) GetId() uint16 {
	if sensor.Track && sensor.Id != anyid {
		return sensor.Id
	}
	// Some command to get the Device ID
	sensor.GetFirmware()
	return sensor.Id
}

func (sensor *sensor) SetId(id uint16) {
	sensor.Set(idCmd, append(make([]byte, 10), split(id)...))
}

func (sensor *sensor) GetState() byte {
	if sensor.Track && sensor.State != ff {
		return sensor.State
	}
	return sensor.Get(stateCmd, nil)
}

func (sensor *sensor) Sleep() {
	sensor.Set(stateCmd, []byte{set, sleep})
}

func (sensor *sensor) Wake() {
	sensor.Set(stateCmd, []byte{set, wake})
}

func (sensor *sensor) GetFirmware() string {
	if sensor.Track && sensor.Firmware != "" {
		return sensor.Firmware
	}
	sensor.write(fwCmd, nil)
	response := sensor.read()
	for response[0] != head || response[1] != ack || response[2] != fwCmd || checksum(response[2:8]) != response[8] {
		response = sensor.read()
	}
	sensor.Id = binary.LittleEndian.Uint16(response[6:8])
	sensor.Firmware = fmt.Sprintf("20%02d.%02d.%02d", response[3], response[4], response[5])
	return sensor.Firmware
}

func (sensor *sensor) GetDuty() byte {
	if sensor.Track && sensor.Duty != ff {
		return sensor.Duty
	}
	return sensor.Get(dutyCmd, nil)
}

func (sensor *sensor) SetDuty(minutes byte) {
	if minutes > 30 {
		fmt.Println("Setting Duty values greater than 30 get ignored")
		os.Exit(3)
	}
	sensor.Set(dutyCmd, []byte{set, minutes})
}
