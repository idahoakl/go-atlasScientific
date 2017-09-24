package main

import (
	"bufio"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/idahoakl/go-atlasScientific/conductivity"
	"github.com/idahoakl/go-atlasScientific/utility"
	"github.com/idahoakl/go-i2c"
	"os"
	"strconv"
)

type cmdFunc func(*bufio.Reader, *conductivity.Conductivity)

type cmd struct {
	name string
	desc string
	exec cmdFunc
}

var cmds = []cmd{
	cmd{name: "info", exec: infoCmd, desc: utility.DeviceInfoDesc},
	cmd{name: "stat", exec: statusCmd, desc: utility.DeviceStatDesc},
	cmd{name: "read", exec: readCmd, desc: utility.ReadingDesc},
	cmd{name: "poll", exec: pollCmd, desc: utility.PollDesc},
	cmd{name: "temp", exec: tempCompCmd, desc: utility.TempCompDesc},
	cmd{name: "cal", exec: conductivityCalCmd, desc: "Get/set conductivity calibration"},
	cmd{name: "probe", exec: probeTypeCmd, desc: "Probe type (K value)"},
}

func main() {
	var conn *i2c.I2C
	var probe *conductivity.Conductivity
	var e error

	cmdMap := make(map[string]cmd)

	for _, cmd := range cmds {
		cmdMap[cmd.name] = cmd
	}

	if conn, e = i2c.NewI2C(1); e != nil {
		log.Fatal(e)
	}

	if probe, e = conductivity.New(100, conn, conductivity.EC); e != nil {
		log.Fatal(e)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		printActions()
		fmt.Print("-> ")
		if text, e := utility.ReadAndSanitizeLine(reader); e != nil {
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

func infoCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	utility.InfoCmd(reader, probe)
}

func statusCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	utility.StatusCmd(reader, probe)
}

func readCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	utility.ReadCmd(reader, probe)
}

func pollCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	utility.PollCmd(reader, probe)
}

func tempCompCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	utility.TempCompCmd(reader, probe)
}

func conductivityCalCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	println("\nEC calibration")
	println(fmt.Sprintf("\tget, %s, %s, %s, %s, clear? [get] ->", conductivity.Dry, conductivity.One, conductivity.High, conductivity.Low))

	if text, e := utility.ReadAndSanitizeLine(reader); e != nil {
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
			calPoint := conductivity.CalibrationPoint(text)
			for loop {
				switch calPoint {
				case "clear":
					if utility.CalClearConfirm(reader) {
						if e := probe.ClearCalibration(); e != nil {
							log.Fatal(e)
						} else {
							println("\tConductivity calibration cleared")
						}
					}
					loop = false
				case conductivity.Dry, conductivity.One, conductivity.High, conductivity.Low:
					performConductivityCal(reader, probe, calPoint)
					loop = false
				default:
					fmt.Printf("\t'%s' not recognized as a command.  Please try again\n", text)
				}
			}
		}
	}
}

func performConductivityCal(reader *bufio.Reader, probe *conductivity.Conductivity, calPoint conductivity.CalibrationPoint) {
	fmt.Printf("\tEnter EC value for '%s' ->", calPoint)

	if text, e := utility.ReadAndSanitizeLine(reader); e != nil {
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
			fmt.Printf("\tcalibration point '%s' set to: %f microsiemens\n", calPoint, val)
		}
	}
}

func probeTypeCmd(reader *bufio.Reader, probe *conductivity.Conductivity) {
	println("\nProbe type")
	println("\tget or <value>?  [get] ->")

	if text, e := utility.ReadAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		if text == "" || text == "get" {
			if i, e := probe.GetProbeType(); e != nil {
				log.Fatal(e)
			} else {
				fmt.Printf("\tProbe type (K value): %f\n", i)
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

			if e := probe.ProbeType(val); e != nil {
				log.Fatal(e)
			} else {
				fmt.Printf("\tprobe type (K value) set to: %f\n", val)
			}
		}
	}
}
