# sds011
**Manage SDS011 particulate matter sensor through CLI or golang librarys**

* CLI app: **v0.2.6**
* Library: **v0.2.1**
* Repo: [github.com/pepa65/sds011](https://github.com/pepa65/sds011)
* After: [github.com/maker-bierzo/sds011](https://github.com/maker-bierzo/sds011)
* Contact: pepa65 <pepa65@passchier.net>
* License: GPLv3 (c) 2022 github.com/pepa65

## CLI app usage
* Install through GoBinaries: `wget -qO- gobinaries.com/pepa65/sds011 |sh`
* Download from [Releases page](https://github.com/pepa65/sds011/releases)

```
sds011 v0.2.6 - Manage SDS011 particulate matter sensors
* Repo:      github.com/pepa65/sds011 <pepa65@passchier.net>
* Usage:     sds011 [ARGUMENT...] [COMMAND]
  COMMAND:   help            Only show this help text (default command)
             wake            Set to Wake state (in Query mode: fan & laser on)
             sleep           Set to Sleep state (only wake commands received)
             mode            Get the sensor's Mode (0: Active, 1: Query)
             active          Set to Active mode (each Duty cycle, a measurement
                              happens automatically; pm will poll the next one)
             query           Set to Query mode (Wake/Sleep states apply,
                             measurements need to be queried manually)
             pm              Get measurement (Active mode: spinup 30 seconds,
                              Query mode: 10 seconds or spinup time)
             duty [MINUTES]  Get or Set the Duty cycle length (0..30 minutes)
                              0: 1.004 seconds, 1..30: uses 30 second spinup
             id [ID]         Get or Set the sensor's (2-byte) ID
             firmware        Get the sensor's firmware version
      All COMMANDs can be shortened by cutting off part of their tail end.
  ARGUMENT:  -h|--help            Only show this help text
             -d|--device DEVICE   The default Device is /dev/ttyUSB0
             -s|--spinup SECONDS  Fan spinup before a measurement (0..30)
             -n|--nocolor         No ANSI color codes in output
             -v|--verbose         Human-readable output
             -D|--debug           Show message passing to/from sensors
      Environment variables SDS011_NOCOLOR, SDS011_VERBOSE and SDS011_DEBUG can
      be set to 1, SDS011_DEVICE to the targetted DEVICE, and SDS011_SPINUP to
      the intended spinup time as an alternative to using the ARGUMENTs.
```

## Build
```shell
# While in the repo root directory:
go build

# Or anywhere:
go get -u github.com/pepa65/sds011

# Smaller binary:
go build -ldflags="-s -w"

# Build for various architectures:
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o sds011
GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o sds011_pi
GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o sds011_bsd
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o sds011_osx
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o sds011.exe

# More extreme shrinking:
upx --brute sds011*
```

## Library usage
Import the library:
```go
import "github.com/pepa65/sds011/lib"
```

Create a sensor object to interact with the sensor:
```go
sensor := sds011.Sensor()
```

Unless you know the device is a Wake state, wake it up:
```go
sensor.Wake()
```
In the Wake state, both the laser and the fan are turned on. Messages are responded to only in the Wake state.

It can be set to Active mode, where measurements happen automatically every second (laser & fan stay on) or every 1..30 minutes (laser & fan turn on for 30 seconds every cycle):
```go
sensor.SetActive()
```

Or it can be set to Query mode, where it will only do a measurement when requested:
```go
sensor.SetQuery()
time.Sleep(10 * time.Second)
m := sensor.Query()
fmt.Printf("ID: %04X  pm2.5: %3.1f  pm10: %3.1f  [μg/m³]\n", m.ID, m.PM2_5, m.PM10)
```
After waking up, let the fan spin for some time to get a good measurement.
The result of `Query()` is a struct `Measurement` with fields `ID` (`uint16`) and `PM2_5` & `PM10` (`float32`).

In Query mode, the sensor should be set to Sleep to preserve the laser (8000 hours service life):
```go
sensor.Sleep()
```
When in Sleep state, the sensor only responds to a Wake message.

Using Active mode, measurement cycle length needs to set to 0 (every 1.004 s) or 1..30 (every 1..30 minutes):
```go
sensor.SetActive()
sensor.SetDuty(0)
```

In Active mode, measurements need to be read from a [channel](https://gobyexample.com/channels):
```go
measurements := sensor.Channel()
for true {
	m := <- measurements
	fmt.Printf("ID: %04X  pm2.5: %.1f  pm10: %.1f  [μg/m³]\n", m.ID, m.PM2_5, m.PM10)
}
```

The firmware version can be retrieved:
```go
fmt.Println("Firmware version:" + sensor.GetFirmware())
```

The sensor ID can be set:
```go
sensor.SetId(0xABCD)
fmt.Printf("Device ID: %x\n", sensor.Id)
```
The sensor object records the last known values of: `Id` (`uint16`), `Firmware` (`string`), `Mode`, `State`, `Duty` (`byte`) and `PM2_5` & `PM10` (`float32`) which can be directly accessed (as in the example above).

When `sensor.Debug` is set to `true`, all the message passing is displayed.
When `sensor.Track` is set to `true`, Get operations first try to access the requested value in the sensor object. (This assumes that there is only 1 sensor device attached to the device interface!)

### Exposed methods of the sensor object
* `NewSensor()` -> `*sensor` - Create a new sensor object; default device interface is `/dev/ttyUSB0`, but a sensor could be created like: `sensor := NewSensor("COM7")` or `sensor2 := NewSensor("/dev/ttyUSB1")`
* `.Query()` -> `*Measurement` - Get a measurement in Query mode
* `.Poll()` -> `*Measurement` - Get a measurement in Active mode
* `.Channel()` -> `chan Measurement` - Initialize a channel for measurements in Active mode
* `.GetMode()` -> `byte` - Get current mode [0:Active, 1:Query]
* `.SetActive()` - Set Active mode
* `.SetQuery()` - Set Query mode
* `.GetId()` -> `uint16` - Get device ID
* `.SetId(uint16)` - Set device ID
* `.GetState()` -> `byte` - Get current State [0:Sleep, 1:Wake] (If it returns, it will always return 1.)
* `.Sleep()` - Set Sleep state
* `.Wake()` - Set Wake state
* `.GetFirmware()` -> `string` - Get firmware version
* `.GetDuty()` -> `byte` - Get Duty cycle length [0:every 1.004 second, 1..30:every n minutes]
* `.SetDuty(byte)` - Set Duty cycle length (values over 30 get ignored)

### Limitations
* There is no effective way to query the Wake/Sleep state, as all messages get ignored in Sleep state, except for setting a Wake state. If there is no response, it might be in Sleep state, or something else is not connecting. Any positive response is always from the Wake state.
* All devices attached to the device interface are targeted for both Set and Get messages. Sensor devices could individually be set to different Device IDs, so their responses can be distinguished by sensor Device ID, but this is currently only supported for measurements, and displayed in Debug mode.

### Complete example
```
package main

import (
  "fmt"
  "github.com/pepa65/sds011/lib"
)

func main() {
  sensor := sds011.Sensor()
  sensor.Wake()
  sensor.SetActive()
  sensor.SetDuty(1)
  fmt.Printf("ID: %04X  Firmware version: %s\n", sensor.Id, sensor.GetFirmware())

  measurements := sensor.Channel()
  var t2_5, t10 float32
  n := 10
  for i := 0 ; i < n; i += 1 {
    m := <- measurements
    t2_5 += m.PM2_5
    t10 += m.PM10
    fmt.Printf("   pm2.5:  %3.1f   pm10: %3.1f   [μg/m³]\n", m.PM2_5, m.PM10)
  }
  sensor.Sleep()
  fmt.Printf("Average:   %3.2f        %3.2f  [μg/m³]\n", t2_5/float32(n), t10/float32(n))
}
```
