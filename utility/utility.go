package utility

import (
	"fmt"
	"bufio"
	"log"
	"strconv"
	"strings"
	"github.com/idahoakl/go-atlasScientific"
)

const (
	DeviceInfoDesc = "Device information"
	DeviceStatDesc = "Device status"
	ReadingDesc = "Take reading"
	TempCompDesc = "Get/set temperature compensation"
)

func ReadAndSanitizeLine(reader *bufio.Reader) (string, error) {
	if text, e := reader.ReadString('\n'); e != nil {
		return "", e
	} else {
		text = strings.TrimRight(text, "\n")
		return text, nil
	}
}

func InfoCmd(reader *bufio.Reader, probe atlasScientific.AtlasScientificSensor) {
	println("\nDevice Info")
	if i, e := probe.GetDeviceInfo(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tType: %s\n", i.Type)
		fmt.Printf("\tFirmware version: %f\n", i.FirmwareVersion)
	}
}

func StatusCmd(reader *bufio.Reader, probe atlasScientific.AtlasScientificSensor) {
	println("\nDevice Status")
	if s, e := probe.GetStatus(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\tRestart code: %s\n", s.RestartCode)
		fmt.Printf("\tVCC voltage: %f\n", s.VccVoltage)
	}
}

func ReadCmd(reader *bufio.Reader, probe atlasScientific.AtlasScientificSensor) {
	println("\nReading")
	if v, e := probe.GetValue(); e != nil {
		log.Fatal(e)
	} else {
		fmt.Printf("\t%f\n", v)
	}
}

func TempCompCmd(reader *bufio.Reader, probe atlasScientific.AtlasScientificSensor) {
	println("\nTemperature compensation")
	println("\tget or <value>?  [get] ->")

	if text, e := ReadAndSanitizeLine(reader); e != nil {
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

func CalClearConfirm(reader *bufio.Reader) bool {
	println("\tThis command will clear all existing calibration.  Continue? yes/no [no] ->")

	if text, e := ReadAndSanitizeLine(reader); e != nil {
		log.Fatal(e)
	} else {
		return text == "yes"
	}

	return false
}
