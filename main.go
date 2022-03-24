package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/pepa65/sds011/lib"
)

const (
	version     = "0.2.0"
	red         = "\033[1m\033[31m"
	green       = "\033[1m\033[32m"
	yellow      = "\033[1m\033[33m"
	blue        = "\033[1m\033[34m"
	def         = "\033[0m"
	get    byte = 0
	set    byte = 1
	sleep  byte = 0
	wake   byte = 1
	active byte = 0
	query  byte = 1
	defspinup   = 10
)

var (
	self        string
	dev                = "/dev/ttyUSB0"
)

func usage(msg string) {
  help := self + " v" + version + ` - Manage SDS011 particulate matter sensors
* Repo:      github.com/pepa65/sds011 <pepa65@passchier.net>
* Usage:     ` + self + ` [ARGUMENT...] [COMMAND]
  COMMAND:   set  active | query  |  wake | sleep  |  duty MINUTES  |  id ID
             get  [ pm | mode | duty | id | firmware ]
               (All subcommands can be shortened by cutting of a tail end.)
               For 'get', 'pm' is the default and can be omitted.
               In Active mode, 'get pm' will wait for a measurement.
               In Query mode, 'get pm' will get a measurement after ` + strconv.Itoa(defspinup) + ` seconds
                 (unless -s/--spinup is used) and then put the sensor to Sleep
                 (to preserver the service life of the laser: 8000 hours).
             help                  Only show this help text (default command)
  ARGUMENT:  -h|--help             Only show this help text
             -d|--device DEVICE    DEVICE is '` + dev + `' by default
             -s|--spinup SECONDS   Fan spinning before a measurement (0..30)
             -v|--verbose          Human-readable output
             -D|--debug            Show message passing to/from sensors
      Environment variables SDS011_VERBOSE and SDS011_DEBUG can be set to '1',
      and SDS011_DEVICE to the targetted DEVICE alternatively as well.`
	fmt.Println(help)
	if msg != "" {
		fmt.Printf("ERROR: %s\n", msg)
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	var err error
	var idval uint64 = 0xFFFF
	spinupval, dutyval, device, spinup, verbose, debug, id, duty := -1, -1, false, false, false, false, false, false
	cmd, subcmd, expect, deviceval := "", "", "", ""
	for _, arg := range os.Args {
		if self == "" { // Get binary name (arg0)
			selves := strings.Split(arg, "/")
			self = selves[len(selves)-1]
			if len(os.Args) == 1 {
				usage("")
			}
			continue
		}
		switch expect {
		case "device":
			_, err = os.Stat(arg)
			if err != nil {
				usage("DEVICE " + arg + " not accessible")
			}
			deviceval = arg
			expect = ""
			continue
		case "minutes":
			dutyval, err = strconv.Atoi(arg)
			if err != nil || dutyval < 0 || dutyval > 30 {
				usage("MINUTES must be 0..30")
			}
			expect = ""
			continue
		case "spinup":
			spinupval, err = strconv.Atoi(arg)
			if err != nil || spinupval < 0 || spinupval > 30 {
				usage("SECONDS must be 0..30")
			}
			expect = ""
			continue
		case "id":
			idval, err = strconv.ParseUint(arg, 0, 0)
			if err != nil || idval > 0xFFFF {
				usage("ID must be 0x0000..0xFFFF")
			}
			expect = ""
			continue
		}
		if arg == "-h" || arg == "--help" {
			usage("")
		}
		// Flags, more than 1 possible
		if arg == "-d" || arg == "--device" {
			device, expect = true, "device"
			continue
		}
		if arg == "-s" || arg == "--spinup" {
			spinup, expect = true, "spinup"
			continue
		}
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			continue
		}
		if arg == "-D" || arg == "--debug" {
			debug = true
			continue
		}
		// Primary commands
		if cmd == "" {
			if arg == "get" || arg == "set" || arg == "help" {
				if cmd == "help" {
					usage("")
				}
				cmd = arg
				continue
			}
			usage("Command should be 'get' or 'set', not '" + arg + "'")
		}
		if subcmd == "" {
			// Subcommands for either primary command
			if strings.HasPrefix("duty", arg) {
				if cmd == "set" {
					expect,duty = "minutes", true
				}
				subcmd = "duty"
				continue
			}
			if strings.HasPrefix("id", arg) {
				if cmd == "set" {
					expect, id = "id", true
				}
				subcmd = "id"
				continue
			}
			// Subcommands for set
			if strings.HasPrefix("active", arg) {
				if cmd != "set" {
					usage("'active' can only be used after 'set'")
				}
				subcmd = "active"
				continue
			}
			if strings.HasPrefix("query", arg) {
				if cmd != "set" {
					usage("'query' can only be used after 'set'")
				}
				subcmd = "query"
				continue
			}
			if strings.HasPrefix("wake", arg) {
				if cmd != "set" {
					usage("'wake' can only be used after 'set'")
				}
				subcmd = "wake"
				continue
			}
			if strings.HasPrefix("sleep", arg) {
				if cmd != "set" {
					usage("'sleep' can only be used after 'set'")
				}
				subcmd = "sleep"
				continue
			}
			// Subcommands for get
			if strings.HasPrefix("pm", arg) {
				if cmd != "get" {
					usage("'pm' can only be used after 'get'")
				}
				subcmd = "pm"
				continue
			}
			if strings.HasPrefix("mode", arg) {
				if cmd != "get" {
					usage("'mode' can only be used after 'get'")
				}
				subcmd = "mode"
				continue
			}
			if strings.HasPrefix("firmware", arg) {
				if cmd != "get" {
					usage("'firmware' can only be used after 'get'")
				}
				subcmd = "firmware"
				continue
			}
			usage("Unrecognized subcommand or argument: '" + arg + "'")
		} else { // cmd and subcommand given
			usage("Unrecognised argument: '" + arg + "'")
		}
	} // end for

	if cmd == "" {
		usage("")
	}
	if cmd == "set" && subcmd == "" {
		usage("Can't use 'set' without subcommand")
	}
	if cmd == "get" && subcmd == "" {
		subcmd = "pm"
	}
	if device && deviceval == "" {
		usage("-d/--device must be followed by a DEVICE")
	}
	if spinup && spinupval == -1 {
		usage("-s/--spinup must be followed by 0..30 SECONDS")
	}
	if spinupval == -1 {
		spinupval = defspinup
	}
	if duty && dutyval == -1 {
		usage("'set duty' must be followed by 0..30 MINUTES")
	}
	if id && cmd == "set" && idval == 0xffff {
		usage("'set id' must be followed by ID")
	}
	if !device {
		deviceval = os.Getenv("SDS011_DEVICE")
	}
	if deviceval == "" {
		deviceval = dev
	}
	if os.Getenv("SDS011_VERBOSE") == "1" {
		verbose = true
	}
	if os.Getenv("SDS011_DEBUG") == "1" {
		debug = true
	}

	sensor := sds011.Sensor(deviceval)
	if debug {
		sensor.Debug = true
	}

	sensor.Wake()
	switch cmd {
	case "get":
		switch subcmd {
		case "pm":
			mode := sensor.GetMode()
			var m *sds011.Measurement
			if mode == active {
				if verbose {
					d := sensor.GetDuty()
					fmt.Printf("Active mode, cycle length %d minutes\n", d)
				}
				m = sensor.Poll()
			} else {
				if verbose {
					fmt.Printf("Query mode, spinup length %d seconds\n", spinupval)
				}
				secs := time.Duration(spinupval) * time.Second
				time.Sleep(secs)
				m = sensor.Query()
				sensor.Sleep()
			}
			if verbose {
				fmt.Printf("ID: %04X  pm2.5: %.1f  pm10: %.1f  [μg/m³]\n", m.ID, m.PM2_5, m.PM10)
			} else {
				fmt.Printf("%04X,%.1f,%.1f\n", m.ID, m.PM2_5, m.PM10)
			}
		case "mode":
			m := sensor.GetMode()
			sensor.Sleep()
			if verbose {
				if m == query {
					fmt.Printf("Sensor %04X is in Query mode\n", sensor.Id)
				} else {
					fmt.Printf("Sensor %04X is in Active mode\n", sensor.Id)
				}
			} else {
				fmt.Printf("%04X,%d\n", sensor.Id, m)
			}
		case "duty":
			m := sensor.GetDuty()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %04X has Duty cycle length %d\n", sensor.Id, m)
			} else {
				fmt.Printf("%04X,%d\n", sensor.Id, m)
			}
		case "id":
			m := sensor.GetId()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor at %s has ID %04X\n", deviceval, m)
			} else {
				fmt.Printf("%04X\n", m)
			}
		case "firmware":
			m := sensor.GetFirmware()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %04X has Firmware version %s\n", sensor.Id, m)
			} else {
				fmt.Printf("%04X,%s\n", sensor.Id, m)
			}
		}
	case "set":
		switch subcmd {
		case "active":
			sensor.SetActive()
			sensor.Sleep()
		case "query":
			sensor.SetQuery()
			sensor.Sleep()
		case "wake":
		case "sleep":
			sensor.Sleep()
		case "duty":
			sensor.SetDuty(byte(dutyval))
			sensor.Sleep()
		case "id":
			sensor.SetId(uint16(idval))
			sensor.Sleep()
		}
	}
}
