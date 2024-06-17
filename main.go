package main

import (
	"flag"
	"fmt"
	"github.com/bcicen/go-units"
	"log"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	MilesPerHour    = units.NewUnit("MilesPerHour", "mph")
	MetersPerSecond = units.NewUnit("MetersPerSecond", "ms")

	WindDirections = []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
)

// Payload is a status update in the Ecowitt protocol.
// See also: https://www.bentasker.co.uk/posts/blog/house-stuff/receiving-weather-info-from-ecowitt-weather-station-and-writing-to-influxdb.html
// And: https://locusglobal.com/connecting-a-weather-station-to-fme/
type Payload struct {
	// Passkey is the MD5 of the MAC address (uppercase)
	Passkey string

	// Absolute pressure (inHg)
	BaromAbsIn float64

	// Relative pressure (inHg)
	BaromRelIn float64

	DailyRainIn float64
	DateUTC     time.Time
	EventRainIn float64

	// 868M ?
	Freq string

	// Memory utilisation? https://www.wxforum.net/index.php?topic=46306.0
	Heap int

	HourlyRainIn float64

	Humidity   int
	HumidityIn int

	// in seconds

	Interval     int
	MaxDailyGust float64

	// WS2900_V2.02.03
	Model string

	MonthlyRainIn float64
	RainRateIn    float64
	Runtime       int

	// fc (foot candles, lol), lux or W/m2
	// it's in W/m2
	SolarRadiation float64

	// EasyWeatherPro_V5.1.6
	StationType string

	// Outdoor temperature
	Tempf float64

	// Indoor temperature
	TempInF float64

	TotalRainIn float64

	// UV index
	UV float64 // or int?

	WeeklyRainIn float64

	// 0 = OK, 1 = Low?
	Wh65Batt float64 // or int?

	// degrees
	WindDir int

	// mph
	WindGustMph  float64
	WindSpeedMph float64

	YearlyRainIn float64
}

func (msg *Payload) ParseValues(v url.Values) error {
	structValue := reflect.ValueOf(msg).Elem()

	for key, values := range v {
		// url.Values is a map[string][]string, but we're only interested in the first value
		if len(values) < 1 {
			return fmt.Errorf("value %s have no values", key)
		}
		rawValue := values[0]

		structFieldValue := structValue.FieldByNameFunc(func(s string) bool {
			if strings.ToLower(s) == strings.ToLower(key) {
				return true
			}
			return false
		})
		if !structFieldValue.IsValid() {
			return fmt.Errorf("no such field: %s", key)
		}

		if !structFieldValue.CanSet() {
			return fmt.Errorf("field cannot be set: %s", key)
		}

		switch structFieldValue.Kind() {
		case reflect.Float64:
			value, err := strconv.ParseFloat(rawValue, 64)
			if err != nil {
				return fmt.Errorf("error parsing %s: %w", key, err)
			}

			structFieldValue.SetFloat(value)

		case reflect.Int:
			value, err := strconv.ParseInt(rawValue, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing %s: %w", key, err)
			}

			structFieldValue.SetInt(value)

		case reflect.String:
			structFieldValue.SetString(rawValue)

		case reflect.Struct:
			if structFieldValue.Type() == reflect.TypeOf(time.Time{}) {
				value, err := time.Parse(time.DateTime, rawValue)
				if err != nil {
					return fmt.Errorf("error parsing %s: %w", key, err)
				}

				structFieldValue.Set(reflect.ValueOf(value))
			}

		default:
			return fmt.Errorf("unsupported type %s for field %s", structFieldValue.Kind(), key)
		}
	}

	return nil
}

func (msg *Payload) ToWeatherData() WeatherData {
	msgSI := WeatherData{
		Passkey:            msg.Passkey,
		AbsolutePressure:   units.NewValue(msg.BaromAbsIn, units.InHg),
		RelativePressure:   units.NewValue(msg.BaromRelIn, units.InHg),
		Timestamp:          msg.DateUTC,
		Frequency:          msg.Freq,
		Heap:               msg.Heap,
		DailyRain:          units.NewValue(msg.DailyRainIn, units.Inch),
		EventRain:          units.NewValue(msg.EventRainIn, units.Inch),
		HourlyRain:         units.NewValue(msg.HourlyRainIn, units.Inch),
		MonthlyRain:        units.NewValue(msg.MonthlyRainIn, units.Inch),
		RainRate:           units.NewValue(msg.RainRateIn, units.Inch),
		TotalRain:          units.NewValue(msg.TotalRainIn, units.Inch),
		WeeklyRain:         units.NewValue(msg.WeeklyRainIn, units.Inch),
		YearlyRain:         units.NewValue(msg.YearlyRainIn, units.Inch),
		OutdoorHumidity:    msg.Humidity,
		IndoorHumidity:     msg.HumidityIn,
		Interval:           time.Duration(msg.Interval) * time.Second,
		Model:              msg.Model,
		Runtime:            msg.Runtime,
		SolarRadiation:     msg.SolarRadiation,
		StationType:        msg.StationType,
		OutdoorTemperature: units.NewValue(msg.Tempf, units.Fahrenheit),
		IndoorTemperature:  units.NewValue(msg.TempInF, units.Fahrenheit),
		UV:                 msg.UV,
		BatteryLevel:       msg.Wh65Batt,
		MaxDailyGuts:       units.NewValue(msg.MaxDailyGust, MilesPerHour),
		WindDirection:      msg.WindDir,
		WindGust:           units.NewValue(msg.WindGustMph, MilesPerHour),
		WindSpeed:          units.NewValue(msg.WindSpeedMph, MilesPerHour),
	}

	return msgSI
}

type WeatherData struct {
	Passkey            string
	AbsolutePressure   units.Value
	RelativePressure   units.Value
	Timestamp          time.Time
	Frequency          string
	Heap               int
	DailyRain          units.Value
	EventRain          units.Value
	HourlyRain         units.Value
	MonthlyRain        units.Value
	RainRate           units.Value
	TotalRain          units.Value
	WeeklyRain         units.Value
	YearlyRain         units.Value
	OutdoorHumidity    int
	IndoorHumidity     int
	Interval           time.Duration
	Model              string
	Runtime            int
	SolarRadiation     float64
	StationType        string
	OutdoorTemperature units.Value
	IndoorTemperature  units.Value
	UV                 float64 // or int?
	BatteryLevel       float64
	MaxDailyGuts       units.Value
	WindDirection      int
	WindGust           units.Value
	WindSpeed          units.Value
}

func offsetDegrees(i, offset int) int {
	if offset < 0 {
		offset += 360
	}

	return (i + offset) % 360
}

func WindDegreesToName(d int) (string, error) {
	if d < 0 || d > 360 {
		return "", fmt.Errorf("invalid wind degrees %d", d)
	}

	idx := (float64(d) / 22.5) + 0.5 // 22.5 = 360 degrees / 16 directions
	idx = math.Floor(idx)
	return WindDirections[int(idx)%len(WindDirections)], nil
}

func run(logger *slog.Logger, addr string) error {
	http.HandleFunc("/data/report/", func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With("client", r.RemoteAddr)

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			logger.Warn("invalid method", "method", r.Method)
			return
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("error parsing form data", "err", err)
			return
		}

		// temporary code, so the error here can be just transient
		var payload Payload
		if err := payload.ParseValues(r.Form); err != nil {
			logger.Error("error parsing form data", "err", err)
		}
		// fmt.Printf("payload = %+v\n", payload)
		logger.Info("payload received", "payload", payload)

		// I think I need to offset by -90 because the West indicator on the probe it's currently pointing South
		payload.WindDir = offsetDegrees(payload.WindDir, -90)

		wd := payload.ToWeatherData()
		tempOut, err := wd.OutdoorTemperature.Convert(units.Celsius)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		tempIn, err := wd.IndoorTemperature.Convert(units.Celsius)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		relPressure, err := wd.RelativePressure.Convert(units.HectoPascal)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		absPressure, err := wd.AbsolutePressure.Convert(units.HectoPascal)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		windGust, err := wd.WindGust.Convert(MetersPerSecond)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		windSpeed, err := wd.WindSpeed.Convert(MetersPerSecond)
		if err != nil {
			logger.Error("error converting weather data", "err", err)
			return
		}

		fmt.Printf("T: %.1fc, T (out): %.1fc; Relative pressure: %.1fhPa, Absolute Pressure: %.1fhPa, Wind Gust: %.1f m/s, Wind Speed: %.1f m/s\n",
			tempOut.Float(), tempIn.Float(), relPressure.Float(), absPressure.Float(), windGust.Float(), windSpeed.Float())
	})

	logger.Info("starting server", "addr", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		return err
	}

	return nil
}

func setupUnits() {
	units.NewRatioConversion(MetersPerSecond, MilesPerHour, 0.447)
}

func main() {
	var logLevelName string
	var addr string
	flag.StringVar(&logLevelName, "log-level", "INFO", "Set the log level")
	flag.StringVar(&addr, "addr", ":8080", "Set the bind address and port")
	flag.Parse()

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(logLevelName)); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing log-level: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	setupUnits()

	if err := run(logger, addr); err != nil {
		log.Fatal(err)
	}
}
