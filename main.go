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
	version     = "0.2.2"
	active byte = 0
	defdevice   = "/dev/ttyUSB0"
	defspinup   = 10
)

var (
	self string
	red         = "\033[1m\033[31m"
	green       = "\033[1m\033[32m"
	yellow      = "\033[1m\033[33m"
	cyan        = "\033[1m\033[36m"
	def         = "\033[0m"
)

func usage(msg string) {
  help := green + self + " v" + version + def + ` - Manage SDS011 particulate matter sensors
* ` + cyan + `Repo` + def + `:      ` + yellow + `github.com/pepa65/sds011` + def + ` <pepa65@passchier.net>
* ` + cyan + `Usage` + def + `:     ` + self + ` [` + green + `ARGUMENT` + def + `...] [` + green + `COMMAND` + def + `]
  ` + green + `COMMAND` + def + `:   ` + green + `set` + def + `  ` + yellow + `active` + def + ` | ` + yellow + `query` + def + `  |  ` + yellow + `wake` + def + ` | ` + yellow + `sleep` + def + `  |  ` + yellow + `duty ` + cyan + `MINUTES` + def + `  |  ` + yellow + `id ` + cyan + `ID` + def + `
             ` + green + `get` + def + `  [ ` + yellow + `pm` + def + ` | ` + yellow + `mode` + def + ` | ` + yellow + `duty` + def + ` | ` + yellow + `id` + def + ` | ` + yellow + `firmware` + def + ` ]
               (All subcommands can be shortened by cutting of a tail end.)
               For ` + green + `get` + def + `, ` + yellow + `pm` + def + ` is the default and can be omitted.
               In Active mode, ` + green + `get ` + yellow + `pm` + def + ` will wait for a measurement.
               In Query mode, ` + green + `get ` + yellow + `pm` + def + ` will get a measurement after ` + cyan + strconv.Itoa(defspinup) + def + ` seconds
                 (unless ` + yellow + `-s` + def + `/` + yellow + `--spinup` + def + ` is used) and then put the sensor to Sleep
                 (to preserver the service life of the laser: 8000 hours).
             ` + yellow + `help` + def + `                  Only show this help text (default command)
  ` + yellow + `ARGUMENT` + def + `:  ` + yellow + `-h` + def + `|` + yellow + `--help` + def + `             Only show this help text
             ` + yellow + `-d` + def + `|` + yellow + `--device ` + cyan + `DEVICE    DEVICE` + def + ` is ` + cyan + defdevice + def + ` by default
             ` + yellow + `-s` + def + `|` + yellow + `--spinup ` + cyan + `SECONDS` + def + `   Fan spinning before a measurement (` + cyan + `0` + def + `..` + cyan + `30` + def + `)
             ` + yellow + `-n` + def + `|` + yellow + `--nocolor` + def + `          No ANSI color codes in output
             ` + yellow + `-v` + def + `|` + yellow + `--verbose` + def + `          Human-readable output
             ` + yellow + `-D` + def + `|` + yellow + `--debug` + def + `            Show message passing to/from sensors
      Environment variables ` + green + `SDS011_VERBOSE` + def + ` and ` + green + `SDS011_DEBUG` + def + ` can be set to ` + cyan + `1` + def + `,
      and ` + green + `SDS011_DEVICE` + def + ` to the targetted ` + cyan + `DEVICE` + def + ` alternatively as well.`
	fmt.Println(help)
	if msg != "" {
		fmt.Printf("%sERROR%s: %s\n", red, def, msg)
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
				usage("DEVICE " + cyan + arg + def + " not accessible")
			}
			deviceval = arg
			expect = ""
			continue
		case "minutes":
			dutyval, err = strconv.Atoi(arg)
			if err != nil || dutyval < 0 || dutyval > 30 {
				usage(cyan + "MINUTES " + def + "must be " +cyan + "0" + def + ".." + cyan + "30" + def)
			}
			expect = ""
			continue
		case "spinup":
			spinupval, err = strconv.Atoi(arg)
			if err != nil || spinupval < 0 || spinupval > 30 {
				usage(cyan + "SECONDS" + def + " must be " + cyan + "0" + def + ".." + cyan + "30" + def)
			}
			expect = ""
			continue
		case "id":
			idval, err = strconv.ParseUint(arg, 0, 0)
			if err != nil || idval > 0xFFFF {
				usage(green + "ID" + def + " must be " + green + "0x0000" + def + ".." + green + "0xFFFF" + def)
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
		if arg == "-n" || arg == "--nocolor" {
			red, cyan, yellow, green, def = "", "", "", "", ""
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
			usage(green + "COMMAND" + def + " should be " + green + "get" + def + " or " + green + "set" + def + ", not " + red + arg + def)
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
					usage(yellow + "active" + def + " can only be used after " + green + "set" + def)
				}
				subcmd = "active"
				continue
			}
			if strings.HasPrefix("query", arg) {
				if cmd != "set" {
					usage(yellow + "query" + def + " can only be used after " + green + "set" + def)
				}
				subcmd = "query"
				continue
			}
			if strings.HasPrefix("wake", arg) {
				if cmd != "set" {
					usage(yellow + "wake" + def + " can only be used after " + green + "set" + def)
				}
				subcmd = "wake"
				continue
			}
			if strings.HasPrefix("sleep", arg) {
				if cmd != "set" {
					usage(yellow + "sleep" + def + " can only be used after " + green + "set" + def)
				}
				subcmd = "sleep"
				continue
			}
			// Subcommands for get
			if strings.HasPrefix("pm", arg) {
				if cmd != "get" {
					usage(yellow + "pm" + def + " can only be used after " + green + "get" +def)
				}
				subcmd = "pm"
				continue
			}
			if strings.HasPrefix("mode", arg) {
				if cmd != "get" {
					usage(yellow + "mode" + def + " can only be used after " + green + "get" +def)
				}
				subcmd = "mode"
				continue
			}
			if strings.HasPrefix("firmware", arg) {
				if cmd != "get" {
					usage(yellow + "firmware" + def + " can only be used after " + green + "get" + def)
				}
				subcmd = "firmware"
				continue
			}
			usage("Unrecognized subcommand or argument: " + red + arg + def)
		} else { // cmd and subcommand given
			usage("Unrecognised argument: " + red + arg + def)
		}
	} // end for

	if cmd == "" {
		usage("")
	}
	if cmd == "set" && subcmd == "" {
		usage("Can't use " + green + "set" + def + " without " + yellow + "subcommand" + def)
	}
	if cmd == "get" && subcmd == "" {
		subcmd = "pm"
	}
	if device && deviceval == "" {
		usage(yellow + "-d" + def + "/" + yellow + "--device" + def + " must be followed by a " + cyan + "DEVICE" + def)
	}
	if spinup && spinupval == -1 {
		usage(yellow + "-s" + def + "/" + yellow + "--spinup" + def + " must be followed by " + cyan + "0" + def + ".." + cyan + "30 SECONDS" + def)
	}
	if spinupval == -1 {
		spinupval = defspinup
	}
	if duty && dutyval == -1 {
		usage(green + "set " + yellow + "duty" + def + " must be followed by " + cyan + "0" + def + ".." + cyan + "30 MINUTES" + def)
	}
	if id && cmd == "set" && idval == 0xffff {
		usage(green + "set " + yellow + "id" + def + " must be followed by " + green + "ID" + def)
	}
	if !device {
		deviceval = os.Getenv("SDS011_DEVICE")
	}
	if deviceval == "" {
		deviceval = defdevice
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
					fmt.Printf("%sActive%s mode, cycle length %s%d%s minutes\n", yellow, def, cyan, def, d)
				}
				m = sensor.Poll()
			} else {
				if verbose {
					fmt.Printf("%sQuery%s mode, spinup length %s%d%s seconds\n", yellow, def, cyan, def, spinupval)
				}
				secs := time.Duration(spinupval) * time.Second
				time.Sleep(secs)
				m = sensor.Query()
				sensor.Sleep()
			}
			if verbose {
				fmt.Printf("ID: %s%04X%s  pm2.5: %s%.1f%s  pm10: %s%.1f%s  [μg/m³]\n", green, m.ID, def, yellow, m.PM2_5, def, yellow, m.PM10, def)
			} else {
				fmt.Printf("%04X,%.1f,%.1f\n", m.ID, m.PM2_5, m.PM10)
			}
		case "mode":
			m := sensor.GetMode()
			sensor.Sleep()
			if verbose {
				if m == active {
					fmt.Printf("Sensor %s%04X%s is in %sActive%s mode\n", green, sensor.Id, def, yellow, def)
				} else {
					fmt.Printf("Sensor %s%04X%s is in %sQuery%s mode\n", green, sensor.Id, def, yellow, def)
				}
			} else {
				fmt.Printf("%04X,%d\n", sensor.Id, m)
			}
		case "duty":
			m := sensor.GetDuty()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %s%04X%s has Duty cycle length %s%d%s\n", green, sensor.Id, def, cyan, m, def)
			} else {
				fmt.Printf("%04X,%d\n", sensor.Id, m)
			}
		case "id":
			m := sensor.GetId()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor at %s%s%s has ID %s%04X%s\n", cyan, deviceval, def, green, m, def)
			} else {
				fmt.Printf("%04X\n", m)
			}
		case "firmware":
			m := sensor.GetFirmware()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %s%04X%s has Firmware version %s%s%s\n", green, sensor.Id, def, yellow, m, def)
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
