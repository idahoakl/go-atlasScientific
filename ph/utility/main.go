package main

import (
	"github.com/idahoakl/go-atlasScientific/ph"
	"github.com/idahoakl/go-i2c"
	"bufio"
	"os"
	"log"
	"fmt"
	"strconv"
	"strings"
)

type cmdFunc func(*bufio.Reader, *ph.PH)

type cmd struct {
	name string
	desc string
	exec cmdFunc
}

var cmds = []cmd{
	cmd{name: "info", exec: infoCmd, desc: "Device information"},
	cmd{name: "stat", exec: statusCmd, desc: "Device status"},
	cmd{name: "read", exec: readCmd, desc: "Take reading"},
	cmd{name: "temp", exec: tempCompCmd, desc: "Get/set temperature compensation"},
	cmd{name: "phCal", exec: phCalCmd, desc: "Get/set PH calibration"},
	cmd{name: "slope", exec: slopeCmd, desc: "Probe calibration slope"},
}

func main() {
	var conn *i2c.I2C
	var probe *ph.PH
	var e error

	cmdMap := make(map[string]cmd)

	for _, cmd := range cmds {
		cmdMap[cmd.name] = cmd
	}

	if conn, e = i2c.NewI2C(1); e != nil {
		log.Fatal(e)
	}

	if probe, e = ph.New(99, conn); e != nil {
		log.Fatal(e)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		printActions()
		fmt.Print("-> ")
		if text, e := readAndSanitizeLine(reader); e != nil {
			log.Fatal(e)
		} else {
			if cmd, ok := cmdMap[text]; ok {
				cmd.exec(reader, probe)
			} else {
				fmt.Printf("Unknown command: '%s'\n", text)
			}
		}
	}
}

func printActions() {
	println("Please select a command:")
	println("Command\t\tNote")

	for _, cmd := range cmds {
		fmt.Printf("%s\t\t%s\n", cmd.name, cmd.desc)
	}
}

func readAndSanitizeLine(reader *bufio.Reader) (string, error) {
	if text, e := reader.ReadString('\n'); e != nil {
		return nil, e
	} else {
		text = strings.TrimRight(text, "\n")
		return text, nil
	}
}

func infoCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nDevice Info")
	if i, e := probe.GetDeviceInfo(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tType: %s\n", i.Type)
		fmt.Printf("\tFirmware version: %f\n", i.FirmwareVersion)
	}
}

func statusCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nDevice Status")
	if s, e := probe.GetStatus(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tRestart code: %s\n", s.RestartCode)
		fmt.Printf("\tVCC voltage: %f\n", s.VccVoltage)
	}
}

func readCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nPH Reading")
	if v, e := probe.GetValue(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tPH: %f\n", v)
	}
}

func tempCompCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nTemperature compensation")
	println("\tget or <value>?  [get] ->")

	if text, e := readAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		if text == "" || text == "get" {
			if tc, e := probe.GetTempCompensation(); e != nil {
				log.Fatal(e)
			} else {
				fmt.Printf("\t%f C\n", tc)
			}
		} else {
			var val float32
			for {
				if tc, e := strconv.ParseFloat(text, 32); e != nil {
					fmt.Printf("\tUnable to parse value '%s' as float32.  Please try again.  Error:  %s\n", text, e)
				} else {
					val = float32(tc)
					break
				}
			}

			if e := probe.TempCompensation(val); e != nil {
				log.Fatal(e)
			} else {
				fmt.Printf("\tset value to: %f C\n", val)
			}
		}
	}
}

func phCalCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nPH calibration")
	println("\tget, high, mid, low, clear? [get] ->")

	if text, e := readAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		if text == "" || text == "get" {
			if i, e := probe.GetCalibrationCount(); e != nil {
				log.Fatal(e)
			} else {
				fmt.Printf("\tCalibration point count: %d\n", i)
			}
		} else {
			loop := true
			for loop {
				switch text {
				case "clear":
					if phCalClearConfirm(reader) {
						if e := probe.ClearCalibration(); e != nil {
							log.Fatal(e)
						} else {
							println("\tPH calibration cleared")
						}
					}
					loop = false
					break;
				case "mid":
					if phCalClearConfirm(reader) {
						performPhCal(reader, probe, text)
					}
					loop = false
					break;
				case "low":
				case "high":
					performPhCal(reader, probe, text)
					loop = false
					break;
				default:
					fmt.Printf("\t'%s' not recognized as a command.  Please try again\n", text)
				}
			}
		}
	}
}

func performPhCal(reader *bufio.Reader, probe *ph.PH, calPoint string) {
	fmt.Printf("\tEnter PH value for '%s' ->", calPoint)

	if text, e := readAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		var val float32
		for {
			if tc, e := strconv.ParseFloat(text, 32); e != nil {
				fmt.Printf("\tUnable to parse value '%s' as float32.  Please try again.  Error:  %s\n", text, e)
			} else {
				val = float32(tc)
				break
			}
		}

		if e := probe.Calibration(calPoint, val); e != nil {
			log.Fatal(e)
		} else {
			fmt.Printf("\tcalibration point '%s' set to: %f C\n", calPoint, val)
		}
	}
}

func phCalClearConfirm(reader *bufio.Reader) bool {
	println("\tThis command will clear all existing calibration.  Continue? yes/no [no] ->")

	if text, e := readAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		return text == "yes"
	}

	return false
}



func slopeCmd(reader *bufio.Reader, probe *ph.PH) {
	println("\nCalibration Slope")
	if s, e := probe.GetCalibrationSlope(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tAcid slope: %f\n", s.AcidSlope)
		fmt.Printf("\tBase slope: %f\n", s.BaseSlope)
	}
}