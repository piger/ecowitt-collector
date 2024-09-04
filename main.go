package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/bcicen/go-units"
	gorillaSchema "github.com/gorilla/schema"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/piger/ecowitt-collector/internal/config"
)

var (
	MilesPerHour    = units.NewUnit("MilesPerHour", "mph")
	MetersPerSecond = units.NewUnit("MetersPerSecond", "ms")

	WindDirections = []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}

	ColumnNames = []string{
		"time",
		"station",
		"pressure_absolute",
		"pressure_relative",
		"frequency",
		"heap",
		"daily_rain",
		"event_rain",
		"hourly_rain",
		"monthly_rain",
		"rain_rate",
		"total_rain",
		"weekly_rain",
		"yearly_rain",
		"humidity_outdoor",
		"humidity_indoor",
		"interval",
		"model",
		"runtime",
		"solar_radiation",
		"station_type",
		"temperature_outdoor",
		"temperature_indoor",
		"uv",
		"battery",
		"wind_max_daily_gust",
		"wind_direction",
		"wind_gust",
		"wind_speed",
	}

	formDecoder = gorillaSchema.NewDecoder()
)

// convertDateTime is an helper for the Gorilla schema package that decodes
// time.Time objects formatted in the DateTime layout.
func convertDateTime(v string) reflect.Value {
	dt, err := time.Parse(time.DateTime, v)
	if err != nil {
		return reflect.Value{}
	}

	return reflect.ValueOf(dt)
}

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
	// See: https://github.com/home-assistant-libs/aioecowitt/blob/9ff160146619126c59efed8b21d27faa1c65f1d4/aioecowitt/sensor.py#L270
	Wh65Batt float64 // or int?

	// degrees
	WindDir int

	// mph
	WindGustMph  float64
	WindSpeedMph float64

	YearlyRainIn float64
	// YearlyRainIn units.Value `unit:"in"`
}

func init() {
	formDecoder.RegisterConverter(time.Time{}, convertDateTime)
}

func (msg *Payload) ParseValues(v url.Values) error {
	return formDecoder.Decode(msg, v)
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
		MaxDailyGust:       units.NewValue(msg.MaxDailyGust, MilesPerHour),
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
	MaxDailyGust       units.Value
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

func windDegreesToName(d int) (string, error) {
	if d < 0 || d > 360 {
		return "", fmt.Errorf("invalid wind degrees %d", d)
	}

	idx := (float64(d) / 22.5) + 0.5 // 22.5 = 360 degrees / 16 directions
	idx = math.Floor(idx)
	return WindDirections[int(idx)%len(WindDirections)], nil
}

func sendMetrics(payload Payload, pool *pgxpool.Pool, table string) error {
	wd := payload.ToWeatherData()
	tempOut, err := wd.OutdoorTemperature.Convert(units.Celsius)
	if err != nil {
		return err
	}

	tempIn, err := wd.IndoorTemperature.Convert(units.Celsius)
	if err != nil {
		return err
	}

	relPressure, err := wd.RelativePressure.Convert(units.HectoPascal)
	if err != nil {
		return err
	}

	absPressure, err := wd.AbsolutePressure.Convert(units.HectoPascal)
	if err != nil {
		return err
	}

	windMaxDailyGust, err := wd.MaxDailyGust.Convert(MetersPerSecond)
	if err != nil {
		return err
	}

	windGust, err := wd.WindGust.Convert(MetersPerSecond)
	if err != nil {
		return err
	}

	windSpeed, err := wd.WindSpeed.Convert(MetersPerSecond)
	if err != nil {
		return err
	}

	dailyRain, err := wd.DailyRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	eventRain, err := wd.EventRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	hourlyRain, err := wd.HourlyRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	monthlyRain, err := wd.MonthlyRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	rainRate, err := wd.RainRate.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	totalRain, err := wd.TotalRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	weeklyRain, err := wd.WeeklyRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	yearlyRain, err := wd.YearlyRain.Convert(units.MilliMeter)
	if err != nil {
		return err
	}

	columns := makeColumnString(ColumnNames)
	values := makeValuesString(ColumnNames)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", table, columns, values),
		wd.Timestamp,
		wd.StationType,
		absPressure.Float(),
		relPressure.Float(),
		wd.Frequency,
		wd.Heap,
		dailyRain.Float(),
		eventRain.Float(),
		hourlyRain.Float(),
		monthlyRain.Float(),
		rainRate.Float(),
		totalRain.Float(),
		weeklyRain.Float(),
		yearlyRain.Float(),
		wd.OutdoorHumidity,
		wd.IndoorHumidity,
		wd.Interval.Seconds(),
		wd.Model,
		wd.Runtime,
		wd.SolarRadiation,
		wd.StationType,
		tempOut.Float(),
		tempIn.Float(),
		wd.UV,
		wd.BatteryLevel,
		windMaxDailyGust.Float(),
		wd.WindDirection,
		windGust.Float(),
		windSpeed.Float(),
	); err != nil {
		return err
	}

	return nil
}

func makeHandler(logger *slog.Logger, conf config.Config, pool *pgxpool.Pool, windOffset int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With("client", r.RemoteAddr)

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Warn("error parsing form data", "err", err)
			return
		}

		var payload Payload
		if err := payload.ParseValues(r.Form); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("error deserializing payload", "err", err)
			return
		}

		if windOffset != 0 {
			payload.WindDir = offsetDegrees(payload.WindDir, windOffset)
		}

		if err := sendMetrics(payload, pool, conf.Database.Table); err != nil {
			logger.Error("error sending metrics", "err", err)
		}
	})
}

func run(logger *slog.Logger, conf config.Config) error {
	ctx := context.Background()

	pgConfig, err := pgxpool.ParseConfig(conf.Database.DSN)
	if err != nil {
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return err
	}

	http.Handle("POST /data/report/", makeHandler(logger, conf, pool, -90))

	logger.Info("starting server", "addr", conf.HTTP.Address)
	if err := http.ListenAndServe(conf.HTTP.Address, nil); err != nil {
		return err
	}

	return nil
}

func setupUnits() {
	units.NewRatioConversion(MetersPerSecond, MilesPerHour, 0.447)
}

func makeColumnString(names []string) string {
	return strings.Join(names, ",")
}

func makeValuesString(names []string) string {
	result := make([]string, len(names))
	for i := range names {
		result[i] = fmt.Sprintf("$%d", i+1)
	}

	return strings.Join(result, ",")
}

func main() {
	var flagConfigFilename string
	flag.StringVar(&flagConfigFilename, "config", "config.yml", "Path to the configuration file")
	flag.Parse()

	conf, err := config.Load(flagConfigFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to load configuration file %s: %s\n", flagConfigFilename, err)
		os.Exit(1)
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(conf.LogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing log-level: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	setupUnits()

	if err := run(logger, conf); err != nil {
		logger.Error("fatal error", "err", err)
		os.Exit(1)
	}
}
