package ph

import (
	"github.com/idahoakl/go-atlasScientific"
	"github.com/idahoakl/go-i2c"
	"strconv"
	"regexp"
	"time"
	"errors"
	"fmt"
)

var (
	slopeRegex = regexp.MustCompile(`\?SLOPE,(?P<acidSlope>\d+\.?\d*),(?P<baseSlope>\d+\.?\d*)`)
	calRegex = regexp.MustCompile(`\?CAL,(?P<calCount>\d)`)
)

type PH struct {
	atlasScientific.AtlasScientific
}

type CalibrationSlope struct {
	AcidSlope float32
	BaseSlope float32
}

func New(address uint8, connection *i2c.I2C) (*PH, error) {
	ph := &PH{
		atlasScientific.AtlasScientific {
			Connection: connection,
			Address: address,
		},
	}


	return ph, nil
}

func (this *PH) GetValue() (float32, error) {
	if rawValue, e := this.GetRawValue(); e != nil {
		return 0, e
	} else {
		if ph, e := strconv.ParseFloat(rawValue, 32); e != nil {
			return 0, e
		} else {
			return float32(ph), nil
		}
	}
}

//Example instruction sequence:
//	Write: SLOPE,?
//	Wait: 300ms
//	Read: ?SLOPE,99.7,100.3
func (this *PH) GetCalibrationSlope() (*CalibrationSlope, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse([]byte("SLOPE"), 300 * time.Millisecond, slopeRegex); e != nil {
		return nil, e
	} else {
		var calSlope CalibrationSlope

		if f, e := strconv.ParseFloat(valMap["acidSlope"], 32); e != nil {
			return nil, e
		} else {
			calSlope.AcidSlope = float32(f)
		}

		if f, e := strconv.ParseFloat(valMap["baseSlope"], 32); e != nil {
			return nil, e
		} else {
			calSlope.BaseSlope = float32(f)
		}

		return &calSlope, nil
	}
}

//Example instruction sequence:
//	Write: CAL,?
//	Wait: 300ms
//	Read: ?CAL,2
func (this *PH) GetCalibrationCount() (int, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse([]byte("CAL,?"), 300 * time.Millisecond, calRegex); e != nil {
		return 0, e
	} else {
		if i, e := strconv.ParseInt(valMap["calCount"], 10, 0); e != nil {
			return 0, e
		} else {
			return int(i), nil
		}
	}
}

//Example instruction sequence:
//	Write: CAL,mid, 7.00
//	Wait: 1600ms
//	Read: <successful read, no data>
func (this *PH) Calibration(calPoint string, phValue float32) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if calPoint != "high" && calPoint != "mid" && calPoint != "low" {
		return errors.New("Invalid calPoint value.  Valid values: high, mid low")
	}

	if _, e := this.Connection.Write(this.Address, []byte(fmt.Sprintf("CAL,%s,%f", calPoint, phValue))); e != nil {
		return e
	}

	if _, e := this.PerformRead(1600 * time.Millisecond); e != nil {
		return e;
	}

	return nil
}


//Example instruction sequence:
//	Write: CAL,clear
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *PH) ClearCalibration() error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if _, e := this.Connection.Write(this.Address, []byte("CAL,clear")); e != nil {
		return e
	}

	if _, e := this.PerformRead(300 * time.Millisecond); e != nil {
		return e;
	}

	return nil
}