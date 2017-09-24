package conductivity

import (
	"errors"
	"fmt"
	"github.com/idahoakl/go-atlasScientific"
	"github.com/idahoakl/go-i2c"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Conductivity struct {
	atlasScientific.AtlasScientific
	DefaultMeasurement ConductivityMeasurement
}

type ConductivityMeasurement int

const (
	EC ConductivityMeasurement = iota
	TDS
	Salinity
	SpecificGravity
)

type CalibrationPoint string

const (
	Dry  CalibrationPoint = "dry"
	One  CalibrationPoint = "one"
	Low  CalibrationPoint = "low"
	High CalibrationPoint = "high"
)

var (
	outputParamRegex = regexp.MustCompile(`\?O,(?P<outputParams>.*)`)
	probeTypeRegex   = regexp.MustCompile(`\?K,(?P<probeType>\d+\.?\d*)`)

	conductivityMeasurementToOutputParam = map[ConductivityMeasurement]string{
		EC:              "EC",
		TDS:             "TDS",
		Salinity:        "S",
		SpecificGravity: "SG",
	}
	outputParamToConductivityMeasurement = map[string]ConductivityMeasurement{
		"EC":  EC,
		"TDS": TDS,
		"S":   Salinity,
		"SG":  SpecificGravity,
	}
)

func New(address uint8, connection *i2c.I2C, defaultMeasurement ConductivityMeasurement) (*Conductivity, error) {
	return &Conductivity{
		DefaultMeasurement: defaultMeasurement,
		AtlasScientific: atlasScientific.AtlasScientific{
			Connection: connection,
			Address:    address,
		},
	}, nil
}

func (this *Conductivity) Init() error {
	return this.defaultOutputParameters()
}

func (this *Conductivity) GetValue() (float32, error) {
	if valMap, e := this.GetAllValues(); e != nil {
		return atlasScientific.ERROR_VALUE, e
	} else {
		return valMap[this.DefaultMeasurement], nil
	}
}

func (this *Conductivity) GetAllValues() (map[ConductivityMeasurement]float32, error) {
	if outputParams, e := this.GetOutputParameters(); e != nil {
		return nil, e
	} else if rawValue, e := this.GetRawValue(); e != nil {
		return nil, e
	} else {
		data := strings.Split(rawValue, ",")

		if len(data) != len(outputParams) {
			return nil,
				errors.New(
					fmt.Sprintf("Output param count mis-match.  Output params: %v\tData values: %v\tRaw string: %s",
						outputParams, data, rawValue))
		}

		values := make(map[ConductivityMeasurement]float32)

		for i, k := range outputParams {
			if f, e := strconv.ParseFloat(data[i], 32); e != nil {
				return nil, e
			} else {
				values[k] = float32(f)
			}
		}

		return values, nil
	}
}

//Example instruction sequence:
//	Write: O,?
//	Wait: 300ms
//	Read: ?O,EC,TDS,S,SG
func (this *Conductivity) GetOutputParameters() ([]ConductivityMeasurement, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("O,?", 300*time.Millisecond, outputParamRegex); e != nil {
		return nil, e
	} else {
		split := strings.Split(valMap["outputParams"], ",")

		var outputParams []ConductivityMeasurement

		for i, s := range split {
			p, ok := outputParamToConductivityMeasurement[s]

			if ok {
				outputParams = append(outputParams, p)
			} else {
				return nil,
					errors.New(
						fmt.Sprintf("Unable to parse output param '%s' at index %d.  Raw string: %s",
							s, i, valMap["outputParams"]))
			}
		}

		return outputParams, nil
	}
}

//Example instruction sequence:
//	Write: O,EC,1
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *Conductivity) OutputParameters(outputParams map[ConductivityMeasurement]bool) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	for key, value := range outputParams {
		p, ok := conductivityMeasurementToOutputParam[key]

		if !ok {
			return errors.New(
				fmt.Sprintf("Unable to find string output param for ConductivityMeasurement: %v",
					key))
		}

		valStr := "0"

		if value {
			valStr = "1"
		}

		if _, e := this.Write(fmt.Sprintf("O,%s,%s", p, valStr)); e != nil {
			return e
		}

		if _, e := this.PerformRead(300 * time.Millisecond); e != nil {
			return e
		}
	}

	return nil
}

//Example instruction sequence:
//	Write: K,?
//	Wait: 300ms
//	Read: ?K,0.66
func (this *Conductivity) GetProbeType() (float32, error) {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if valMap, e := this.WriteReadParse("K,?", 300*time.Millisecond, probeTypeRegex); e != nil {
		return atlasScientific.ERROR_VALUE, e
	} else {
		if tempComp, err := strconv.ParseFloat(valMap["probeType"], 32); err != nil {
			return atlasScientific.ERROR_VALUE, err
		} else {
			return float32(tempComp), nil
		}
	}
}

//Example instruction sequence:
//	Write: K,0.66
//	Wait: 300ms
//	Read: <successful read, no data>
func (this *Conductivity) ProbeType(probeType float32) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	if probeType < 0.1 || probeType > 10 {
		return errors.New(fmt.Sprintf("Invalid probe type '%f'.  Must be between 0.1 and 10.", probeType))
	}

	if _, e := this.Write(fmt.Sprintf("K,%f", probeType)); e != nil {
		return e
	}

	if _, e := this.PerformRead(300 * time.Millisecond); e != nil {
		return e
	}

	return nil
}

//Example instruction sequence:
//	Write: CAL,low,210
//	Wait: 1300ms (2000ms for dry)
//	Read: <successful read, no data>
func (this *Conductivity) Calibration(calPoint CalibrationPoint, ecValue float32) error {
	this.Mtx.Lock()
	defer this.Mtx.Unlock()

	var calStr string
	var calTime time.Duration

	if calPoint == Dry {
		calStr = "CAL,dry"
		calTime = 2000 * time.Millisecond
	} else {
		calStr = fmt.Sprintf("CAL,%s,%d", calPoint, int(ecValue))
		calTime = 1500 * time.Millisecond
	}

	if _, e := this.Write(calStr); e != nil {
		return e
	}

	if _, e := this.PerformRead(calTime); e != nil {
		return e
	}

	return nil
}

func (this *Conductivity) defaultOutputParameters() error {
	allOn := map[ConductivityMeasurement]bool{
		EC:              true,
		TDS:             true,
		Salinity:        true,
		SpecificGravity: true,
	}

	return this.OutputParameters(allOn)
}
