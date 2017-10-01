package atlasScientific

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/idahoakl/go-i2c"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var (
	statusRegex     = regexp.MustCompile(`\?STATUS,(?P<restartCode>\D),(?P<vccVolt>\d+\.?\d*)`)
	deviceInfoRegex = regexp.MustCompile(`\?I,(?P<deviceType>\w+),(?P<firmwareVersion>\d+\.?\d*)`)
	tempCompRegex   = regexp.MustCompile(`\?T,(?P<tempCompensation>\d+\.?\d*)`)
	ledStatRegex    = regexp.MustCompile(`\?L,(?P<ledStatus>[01])`)
	calRegex        = regexp.MustCompile(`\?CAL,(?P<calCount>\d)`)

	errParseResponse = errors.New("Response could not be parsed")
)

const ERROR_VALUE = -1

type AtlasScientific struct {
	Connection *i2c.I2C
	Address    uint8
	Mtx        sync.Mutex
}

type Status struct {
	RestartCode string
	VccVoltage  float32
}

type DeviceInfo struct {
	Type            string
	FirmwareVersion float32
}

type AtlasScientificSensor interface {
	Init() error
	GetRawValue() (string, error)
	GetValue() (float32, error)
	GetStatus() (*Status, error)
	GetDeviceInfo() (*DeviceInfo, error)
	GetTempCompensation() (float32, error)
	TempCompensation(tempC float32) error
	GetLedStatus() (bool, error)
	LedStatus(isLedOn bool) error
	ClearCalibration() error
	GetCalibrationCount() (int, error)
}

type ReadError struct {
	status  int
	message string
}

func (this *ReadError) Error() string {
	return this.message
}

func (this *AtlasScientific) Init() error {
	return nil
}

//Example instruction sequence:
//	Write: R
//	Wait: 1000ms
//	Read: <value>
func (this *AtlasScientific) GetRawValue() (string, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if _, e := this.Write("R"); e != nil {
		return "", e
	}

	if data, e := this.PerformRead(1000 * time.Millisecond); e != nil {
		return "", e
	} else {
		return data, nil
	}
}

func (this *AtlasScientific) GetValue() (float32, error) {
	return 0, errors.New("Not implemented")
}

//GetStatus retrieves the status of a device
//Example instruction sequence:
//	Write: STATUS
//	Wait: 300ms
//	Read: ?STATUS,P,5.038
func (this *AtlasScientific) GetStatus() (*Status, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("STATUS", 300*time.Millisecond, statusRegex); e != nil {
		return nil, e
	} else {
		if f, e := strconv.ParseFloat(valMap["vccVolt"], 32); e != nil {
			return nil, e
		} else {
			return &Status{
				RestartCode: valMap["restartCode"],
				VccVoltage:  float32(f),
			}, nil
		}
	}
}

//Example instruction sequence:
//	Write: I
//	Wait: 300ms
//	Read: ?I,PH,1.0
func (this *AtlasScientific) GetDeviceInfo() (*DeviceInfo, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("I", 300*time.Millisecond, deviceInfoRegex); e != nil {
		return nil, e
	} else {
		if f, e := strconv.ParseFloat(valMap["firmwareVersion"], 32); e != nil {
			return nil, e
		} else {
			return &DeviceInfo{
				Type:            valMap["deviceType"],
				FirmwareVersion: float32(f),
			}, nil
		}
	}
}

//Example instruction sequence:
//	Write: T,?
//	Wait: 300ms
//	Read: ?T,19.5
func (this *AtlasScientific) GetTempCompensation() (float32, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("T,?", 300*time.Millisecond, tempCompRegex); e != nil {
		return 0, e
	} else {
		if tempComp, err := strconv.ParseFloat(valMap["tempCompensation"], 32); err != nil {
			return 0, err
		} else {
			return float32(tempComp), nil
		}
	}
}

//Example instruction sequence:
//	Write: T,19.5
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *AtlasScientific) TempCompensation(tempC float32) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if _, e := this.Write(fmt.Sprintf("T,%f", tempC)); e != nil {
		return e
	}

	if _, e := this.PerformRead(300 * time.Millisecond); e != nil {
		return e
	}

	return nil
}

//Example instruction sequence:
//	Write: L,?
//	Wait: 300ms
//	Read: ?L,1
func (this *AtlasScientific) GetLedStatus() (bool, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("L,?", 300*time.Millisecond, ledStatRegex); e != nil {
		return false, e
	} else {
		if isLedOn, err := strconv.ParseBool(valMap["ledStatus"]); err != nil {
			return false, err
		} else {
			return isLedOn, nil
		}
	}
}

//Example instruction sequence:
//	Write: L,1
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *AtlasScientific) LedStatus(isLedOn bool) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	writeCmd := "L,0"

	if isLedOn {
		writeCmd = "L,1"
	}

	if _, e := this.Write(writeCmd); e != nil {
		return e
	}

	if _, e := this.PerformRead(300 * time.Millisecond); e != nil {
		return e
	}

	return nil
}

//Example instruction sequence:
//	Write: CAL,clear
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *AtlasScientific) ClearCalibration() error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if _, e := this.Write("CAL,clear"); e != nil {
		return e
	}

	if _, e := this.PerformRead(1300 * time.Millisecond); e != nil {
		return e
	}

	return nil
}

//Example instruction sequence:
//	Write: CAL,?
//	Wait: 300ms
//	Read: ?CAL,2
func (this *AtlasScientific) GetCalibrationCount() (int, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("CAL,?", 300*time.Millisecond, calRegex); e != nil {
		return 0, e
	} else {
		if i, e := strconv.ParseInt(valMap["calCount"], 10, 0); e != nil {
			return 0, e
		} else {
			return int(i), nil
		}
	}
}

func (this *AtlasScientific) PerformRead(waitTime time.Duration) (string, error) {
	time.Sleep(waitTime)

	data := make([]byte, 64)
	if _, e := this.Connection.Read(this.Address, data); e != nil {
		return "", e
	}

	//this.GetContextLogger().WithField("data", data).Debug("Raw data read from device")

	e := checkReadError(data)
	if e != nil {
		if e.status == 254 {
			this.GetContextLogger().WithField("waitTime", waitTime).Warn("Attempting re-read after additional wait time")
			//If read wasn't ready try once more
			time.Sleep(waitTime)
			if _, e := this.Connection.Read(this.Address, data); e != nil {
				return "", e
			}

			//this.GetContextLogger().WithField("data", data).Debug("Raw data read from device")

			if e := checkReadError(data); e != nil {
				return "", e
			}

		} else {
			return "", e
		}
	}

	trimData := bytes.Trim(data, "\x00")

	//this.GetContextLogger().WithField("trimmedData", trimData).Debug("Trimmed data")

	return string(trimData[1:]), nil
}

func (this *AtlasScientific) WriteReadParse(writeCommand string, waitTime time.Duration, parseRegex *regexp.Regexp) (map[string]string, error) {
	if _, e := this.Write(writeCommand); e != nil {
		return nil, e
	}

	if data, e := this.PerformRead(waitTime); e != nil {
		return nil, e
	} else {
		if valMap, e := FindStringSubmatchMap(parseRegex, data); e != nil {
			return nil, e
		} else {
			return valMap, nil
		}
	}
}

func (this *AtlasScientific) Write(data string) (int, error) {
	byteData := []byte(data)

	/* this.GetContextLogger().WithFields(log.Fields{
		"data":     data,
		"byteData": byteData,
	}).Debug("Writing to device") */

	return this.Connection.Write(this.Address, byteData)
}

func (this *AtlasScientific) GetContextLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"i2cBus":        this.Connection.Bus,
		"deviceAddress": this.Address,
	})
}

func FindStringSubmatchMap(r *regexp.Regexp, s string) (map[string]string, error) {
	captures := make(map[string]string)

	match := r.FindStringSubmatch(s)
	if match == nil {
		return nil, errParseResponse
	}

	for i, name := range r.SubexpNames() {
		if i == 0 {
			continue
		}
		captures[name] = match[i]

	}
	return captures, nil
}

func checkReadError(data []byte) *ReadError {
	switch data[0] {
	case 1:
		return nil
	case 2:
		return &ReadError{
			status:  2,
			message: "Read error",
		}
	case 254:
		return &ReadError{
			status:  254,
			message: "Pending",
		}
	case 255:
		return &ReadError{
			status:  255,
			message: "No Data",
		}
	}

	return nil
}
