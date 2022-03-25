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
	version        = "0.2.6"
	defDevice      = "/dev/ttyUSB0"
	defSpinup      = 10
	active    byte = 0
)

var (
	self   string
	red           = "\033[1m\033[31m"
	green         = "\033[1m\033[32m"
	yellow        = "\033[1m\033[33m"
	cyan          = "\033[1m\033[36m"
	def           = "\033[0m"
)

func usage(msg string) {
  help := green + self + " v" + version + def + ` - Manage SDS011 particulate matter sensors
* ` + cyan + `Repo` + def + `:      ` + yellow + `github.com/pepa65/sds011` + def + ` <pepa65@passchier.net>
* ` + cyan + `Usage` + def + `:     ` + self + ` [` + yellow + `ARGUMENT` + def + `...] [` + green + `COMMAND` + def + `]
  ` + green + `COMMAND` + def + `:   ` + green + `help` + def + `            Only show this help text (default command)
             ` + green + `wake` + def + `            Set to Wake state (in Query mode: fan & laser on)
             ` + green + `sleep` + def + `           Set to Sleep state (only ` + green + `wake` + def + ` commands received)
             ` + green + `mode` + def + `            Get the sensor's Mode (` + cyan + `0` + def + `: Active, ` + cyan + `1` + def + `: Query)
             ` + green + `active` + def + `          Set to Active mode (each Duty cycle, a measurement
                              happens automatically; ` + green + `pm` + def + ` will poll the next one)
             ` + green + `query` + def + `           Set to Query mode (Wake/Sleep states apply,
                             measurements need to be queried manually)
             ` + green + `pm` + def + `              Get measurement (Active mode: spinup ` + cyan + `30` + def + ` seconds,
                              Query mode: ` + cyan + strconv.Itoa(defSpinup) + def + ` seconds or ` + yellow+ `spinup` + def + ` time)
             ` + green + `duty ` + def + `[` + cyan + `MINUTES` + def + `]  Get or Set the Duty cycle length (` + cyan + `0` + def + `..` + cyan + `30` + def + ` minutes)
                              ` + cyan + `0` + def + `: 1.004 seconds, ` + cyan + `1` + def + `..` + cyan + `30` + def + `: uses 30 second spinup
             ` + green + `id ` + def + `[` + cyan + `ID` + def + `]         Get or Set the sensor's (2-byte) ID
             ` + green + `firmware` + def + `        Get the sensor's firmware version
      All COMMANDs can be shortened by cutting off part of their tail end.
  ` + yellow + `ARGUMENT` + def + `:  ` + yellow + `-h` + def + `|` + yellow + `--help` + def + `            Only show this help text
             ` + yellow + `-d` + def + `|` + yellow + `--device ` + cyan + `DEVICE   ` + def + `The default Device is ` + cyan + defDevice + def + `
             ` + yellow + `-s` + def + `|` + yellow + `--spinup ` + cyan + `SECONDS` + def + `  Fan spinup before a measurement (` + cyan + `0` + def + `..` + cyan + `30` + def + `)
             ` + yellow + `-n` + def + `|` + yellow + `--nocolor` + def + `         No ANSI color codes in output
             ` + yellow + `-v` + def + `|` + yellow + `--verbose` + def + `         Human-readable output
             ` + yellow + `-D` + def + `|` + yellow + `--debug` + def + `           Show message passing to/from sensors
      Environment variables ` + green + `SDS011_NOCOLOR` + def + `, ` + green + `SDS011_VERBOSE` + def + ` and ` + green + `SDS011_DEBUG` + def + ` can
      be set to ` + cyan + `1` + def + `, ` + green + `SDS011_DEVICE` + def + ` to the targetted ` + cyan + `DEVICE` + def + `, and ` + green + `SDS011_SPINUP` + def + ` to
      the intended ` + yellow + `spinup` + def + ` time as an alternative to using the ` + yellow + `ARGUMENT` + def + `s.`
	fmt.Println(help)
	if msg != "" {
		fmt.Printf("%sERROR%s: %s\n", red, def, msg)
		os.Exit(1)
	}
	os.Exit(0)
}

func checkDevice(arg string) string {
	_, err := os.Stat(arg)
	if err != nil {
		usage("DEVICE " + cyan + arg + def + " not accessible")
	}
	return arg
}

func checkSpinup(arg string) int {
	spinup, err := strconv.Atoi(arg)
	if err != nil || spinup < 0 || spinup > 30 {
		usage(cyan + "SECONDS" + def + " must be " + cyan + "0" + def + ".." + cyan + "30" + def)
	}
	return spinup
}

func main() {
	var err error
	var id uint64 = 0x11111
	spinup, duty, getDevice, getSpinup, verbose, debug, spinupSet := -1, -1, false, false, false, false, false
	cmd, expect, device := "", "", ""
	for _, arg := range os.Args {

		if self == "" { // Get binary name (arg0)
			selves := strings.Split(arg, "/")
			self = selves[len(selves)-1]
			continue
		}

		// In case arguments might follow
		switch expect {
		case "device":  // Mandatory argument
			expect = ""
			device = checkDevice(arg)
			continue
		case "spinup":  // Mandatory argument
			expect = ""
			spinup = checkSpinup(arg)
			spinupSet = true
			continue
		case "duty":  // Optional argument
			expect = ""
			if arg[0] >= '0' && arg[0] <= '9' {
				duty, err = strconv.Atoi(arg)
				if err != nil || duty < 0 || duty > 30 {
					usage(cyan + "MINUTES " + def + "must be " +cyan + "0" + def + ".." + cyan + "30" + def + ", not: " + red + arg + def)
				}
				continue
			}
		case "id":  // Optional argument
			expect = ""
			if arg[0] >= '0' && arg[0] <= '9' {
				id, err = strconv.ParseUint(arg, 0, 0)
				if err != nil || id > 0xFFFF {
					usage(cyan + "ID" + def + " must be " + cyan + "0x0000" + def + ".." + cyan + "0xFFFF" + def)
				}
				continue
			}
		}

		// Flags, multiple possible
		if arg == "-h" || arg == "--help" {
			// Override any command
			cmd = "help"
			continue
		}
		if arg == "-d" || arg == "--device" {
			getDevice, expect = true, "device"
			continue
		}
		if arg == "-s" || arg == "--spinup" {
			getSpinup, expect = true, "spinup"
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

		// Commands (abbreviatable)
		if cmd != "" { // Command already filled in
			usage("Unrecognised argument: " + red + arg + def)
		}
		if strings.HasPrefix("help", arg) {
			cmd = "help"
			continue
		}
		if strings.HasPrefix("duty", arg) {
			cmd, expect = "duty", "duty"
			continue
		}
		if strings.HasPrefix("id", arg) {
			cmd, expect = "id", "id"
			continue
		}
		if strings.HasPrefix("active", arg) {
			cmd = "active"
			continue
		}
		if strings.HasPrefix("query", arg) {
			cmd = "query"
			continue
		}
		if strings.HasPrefix("wake", arg) {
			cmd = "wake"
			continue
		}
		if strings.HasPrefix("sleep", arg) {
			cmd = "sleep"
			continue
		}
		if strings.HasPrefix("pm", arg) {
			cmd = "pm"
			continue
		}
		if strings.HasPrefix("mode", arg) {
			cmd = "mode"
			continue
		}
		if strings.HasPrefix("firmware", arg) {
			cmd = "firmware"
			continue
		}
		if cmd == "" {
			usage("Unrecognised command: " + red + arg + def)
		}
	} // end for

	if cmd == "" {
		cmd = "help"
	}
	if getDevice && device == "" {
		usage(yellow + "-d" + def + "/" + yellow + "--device" + def + " must be followed by a " + cyan + "DEVICE" + def)
	}
	if getSpinup && !spinupSet {
		usage(yellow + "-s" + def + "/" + yellow + "--spinup" + def + " must be followed by " + cyan + "0" + def + ".." + cyan + "30 SECONDS" + def)
	}
	if !spinupSet {
		spinupEnv := os.Getenv("SDS011_SPINUP")
		if spinupEnv != "" {
			spinup = checkSpinup(spinupEnv)
		} else {
			spinup = defSpinup
		}
	}
	if getSpinup && cmd != "pm" && verbose {
		fmt.Println("Setting " + yellow + "spinup" + def + " time only relevant for " + green + "pm" + def + " command in Passive mode")
	}
	if os.Getenv("SDS011_NOCOLOR") == "1" {
		red, cyan, yellow, green, def = "", "", "", "", ""
	}
	deviceEnv := os.Getenv("SDS011_DEVICE")
	if !getDevice && deviceEnv != "" {
		device = checkDevice(deviceEnv)
	}
	if cmd == "duty" && duty == -1 {
		cmd = "getduty"
	}
	if cmd == "id" && id == 0x11111 {
		cmd = "getid"
	}
	if device == "" {
		device = defDevice
	}
	if os.Getenv("SDS011_VERBOSE") == "1" {
		verbose = true
	}
	if os.Getenv("SDS011_DEBUG") == "1" {
		debug = true
	}

	sensor := sds011.Sensor(device)
	if debug {
		sensor.Debug = true
	}
	sensor.Wake()
	switch cmd {
	case "help":
		usage("")
	case "pm":
		mode := sensor.GetMode()
		var m *sds011.Measurement
		if mode == active {
			if verbose {
				if getSpinup {
					fmt.Println("Setting " + yellow + "spinup" + def + " time only relevant for " + green + "pm" + def + " command in Passive mode")
				}
				d := sensor.GetDuty()
				fmt.Printf("%sActive%s mode, cycle length %s%d%s minutes\n", yellow, def, cyan, d, def)
			}
			m = sensor.Poll()
		} else {
			if verbose {
				fmt.Printf("%sQuery%s mode, spinup time %s%d%s seconds\n", yellow, def, cyan, spinup, def)
			}
			secs := time.Duration(spinup) * time.Second
			time.Sleep(secs)
			m = sensor.Query()
			sensor.Sleep()
		}
		if verbose {
			fmt.Printf("ID: %s%04X%s  pm2.5: %s%.1f%s  pm10: %s%.1f%s  [μg/m³]\n", cyan, m.ID, def, yellow, m.PM2_5, def, yellow, m.PM10, def)			} else {
			fmt.Printf("%04X,%.1f,%.1f\n", m.ID, m.PM2_5, m.PM10)
		}
	case "mode":
		m := sensor.GetMode()
		sensor.Sleep()
		if verbose {
			if m == active {
				fmt.Printf("Sensor %s%04X%s is in %sActive%s mode\n", cyan, sensor.Id, def, yellow, def)
			} else {
				fmt.Printf("Sensor %s%04X%s is in %sQuery%s mode\n", cyan, sensor.Id, def, yellow, def)
			}
		} else {
			fmt.Printf("%04X,%d\n", sensor.Id, m)
		}
	case "getduty":
			m := sensor.GetDuty()
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %s%04X%s has Duty cycle length %s%d%s\n", cyan, sensor.Id, def, cyan, m, def)
			} else {
				fmt.Printf("%04X,%d\n", sensor.Id, m)
			}
	case "duty":
			sensor.SetDuty(byte(duty))
			sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %s%04X%s is set to Duty cycle length %s%d%s\n", cyan, sensor.Id, def, cyan, duty, def)
			}
	case "getid":
		m := sensor.GetId()
		sensor.Sleep()
		if verbose {
			fmt.Printf("Sensor at %s%s%s has ID %s%04X%s\n", cyan, device, def, cyan, m, def)
		} else {
			fmt.Printf("%04X\n", m)
		}
	case "id":
		sensor.SetId(uint16(id))
		sensor.Sleep()
			if verbose {
				fmt.Printf("Sensor %s%04X%s is set to ID %s%04X%s\n", cyan, sensor.Id, def, cyan, id, def)
			}
	case "firmware":
		m := sensor.GetFirmware()
		sensor.Sleep()
		if verbose {
			fmt.Printf("Sensor %s%04X%s has Firmware version %s%s%s\n", cyan, sensor.Id, def, yellow, m, def)
		} else {
			fmt.Printf("%04X,%s\n", sensor.Id, m)
		}
	case "active":
		sensor.SetActive()
		sensor.Sleep()
		if verbose {
			fmt.Printf("Sensor %s%04X%s is set to %sActive%s mode\n", cyan, sensor.Id, def, yellow, def)
		}
	case "query":
		sensor.SetQuery()
		sensor.Sleep()
		if verbose {
			fmt.Printf("Sensor %s%04X%s is set to %sQuery%s mode\n", cyan, sensor.Id, def, yellow, def)
		}
	case "wake":
		if verbose {
			fmt.Printf("Sensor %s%04X%s is set to %sWake%s mode\n", cyan, sensor.Id, def, yellow, def)
		}
	case "sleep":
		sensor.Sleep()
		if verbose {
			fmt.Printf("Sensor %s%04X%s is set to %sSleep%s mode\n", cyan, sensor.Id, def, yellow, def)
		}
	}
}
