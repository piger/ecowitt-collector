package main

import (
	"time"

	"github.com/bcicen/go-units"
)

var (
	MilesPerHour    = units.NewUnit("MilesPerHour", "mph")
	MetersPerSecond = units.NewUnit("MetersPerSecond", "ms")
)

func init() {
	units.NewRatioConversion(MilesPerHour, MetersPerSecond, 0.44704)
}

// Time is a type alias that has helpers to serialize to JSON and to
// deserialize from the time format used by the weather station (which is time.DateTime).
type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	v := time.Time(t).Format(time.RFC3339)
	return []byte(v), nil
}

func (t *Time) UnmarshalText(text []byte) error {
	dt, err := time.Parse(time.DateTime, string(text))
	if err != nil {
		return err
	}

	*t = Time(dt)

	return nil
}

// payload is the POST form data sent by the weather station to a custom endpoint.
type payload struct {
	// Some sort of identifier; seems to be the MD5 hash of the MAC address
	// of the station.
	Passkey string

	// Absolute pressure (inHg)
	BaromAbsIn float64

	// Relative pressure (inHg)
	BaromRelIn float64

	// Total rain recorded today (in)
	DailyRainIn float64

	// Current time from the station
	DateUTC Time

	// Total rain recorded during the last event (in)
	EventRainIn float64

	Freq string

	// Memory heap
	Heap int

	// Total rain recorded in this hour (in)
	HourlyRainIn float64

	// Outdoor humidity (percentage)
	Humidity int

	// Indoor humidity (percentage)
	HumidityIn int

	// How often the station sends data to the collector (seconds)
	Interval int

	// Maximum wind gust speed today (mph)
	MaxDailyGust float64

	// Model name of the station
	Model string

	// Total rain recorded this month (in)
	MonthlyRainIn float64

	// Current rainfall rate (inches per hour?)
	RainRateIn float64

	Runtime int

	// Solar radiation (W/m2)
	SolarRadiation float64

	// Station type
	StationType string

	// Outdoor temperature (f)
	Tempf float64

	// Indoor temperature (f)
	TempInF float64

	// Total rain recorded (in)
	TotalRainIn float64

	// UV index
	UV float64 // or int?

	// Vapour Pressure Deficit
	VPD float64

	// Total rain recorded this week (in)
	WeeklyRainIn float64

	// Battery status (0=OK, 1=LOW, unconfirmed)
	Wh65Batt float64 // or int?

	// Wind direction (degrees)
	WindDir int

	// Wind gust speed (mph)
	WindGustMph float64

	// Wind speed (mph)
	WindSpeedMph float64

	// Total rain recorded this year (in)
	YearlyRainIn float64
}

type WeatherData struct {
	Passkey            string        `db:"-"`
	AbsolutePressure   float64       `db:"pressure_absolute"`
	RelativePressure   float64       `db:"pressure_relative"`
	Timestamp          time.Time     `db:"time"`
	Frequency          string        `db:"frequency"`
	Heap               int           `db:"heap"`
	DailyRain          float64       `db:"daily_rain"`
	EventRain          float64       `db:"event_rain"`
	HourlyRain         float64       `db:"hourly_rain"`
	MonthlyRain        float64       `db:"monthly_rain"`
	RainRate           float64       `db:"rain_rate"`
	TotalRain          float64       `db:"total_rain"`
	WeeklyRain         float64       `db:"weekly_rain"`
	YearlyRain         float64       `db:"yearly_rain"`
	OutdoorHumidity    int           `db:"humidity_outdoor"`
	IndoorHumidity     int           `db:"humidity_indoor"`
	Interval           time.Duration `db:"interval"`
	Model              string        `db:"model"`
	Runtime            int           `db:"runtime"`
	SolarRadiation     float64       `db:"solar_radiation"`
	StationType        string        `db:"station_type"`
	OutdoorTemperature float64       `db:"temperature_outdoor"`
	IndoorTemperature  float64       `db:"temperature_indoor"`
	UV                 float64       `db:"uv"`
	BatteryLevel       float64       `db:"battery"`
	MaxDailyGust       float64       `db:"wind_max_daily_gust"`
	WindDirection      int           `db:"wind_direction"`
	WindGust           float64       `db:"wind_gust"`
	WindSpeed          float64       `db:"wind_speed"`
}

func NewWeatherData(p payload) (*WeatherData, error) {
	absPressure := units.NewValue(p.BaromAbsIn, units.InHg)
	if v, err := absPressure.Convert(units.HectoPascal); err != nil {
		return nil, err
	} else {
		absPressure = v
	}

	relPressure := units.NewValue(p.BaromRelIn, units.InHg)
	if v, err := relPressure.Convert(units.HectoPascal); err != nil {
		return nil, err
	} else {
		relPressure = v
	}

	dailyRain := units.NewValue(p.DailyRainIn, units.Inch)
	if v, err := dailyRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		dailyRain = v
	}

	eventRain := units.NewValue(p.EventRainIn, units.Inch)
	if v, err := eventRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		eventRain = v
	}

	monthlyRain := units.NewValue(p.MonthlyRainIn, units.Inch)
	if v, err := monthlyRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		monthlyRain = v
	}

	rainRate := units.NewValue(p.RainRateIn, units.Inch)
	if v, err := rainRate.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		rainRate = v
	}

	totalRain := units.NewValue(p.TotalRainIn, units.Inch)
	if v, err := totalRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		totalRain = v
	}

	weeklyRain := units.NewValue(p.WeeklyRainIn, units.Inch)
	if v, err := weeklyRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		weeklyRain = v
	}

	yearlyRain := units.NewValue(p.YearlyRainIn, units.Inch)
	if v, err := yearlyRain.Convert(units.MilliMeter); err != nil {
		return nil, err
	} else {
		yearlyRain = v
	}

	outTemp := units.NewValue(p.Tempf, units.Fahrenheit)
	if v, err := outTemp.Convert(units.Celsius); err != nil {
		return nil, err
	} else {
		outTemp = v
	}

	inTemp := units.NewValue(p.TempInF, units.Fahrenheit)
	if v, err := inTemp.Convert(units.Celsius); err != nil {
		return nil, err
	} else {
		inTemp = v
	}

	maxDailyGust := units.NewValue(p.MaxDailyGust, MilesPerHour)
	if v, err := maxDailyGust.Convert(MetersPerSecond); err != nil {
		return nil, err
	} else {
		maxDailyGust = v
	}

	windGust := units.NewValue(p.WindGustMph, MilesPerHour)
	if v, err := windGust.Convert(MetersPerSecond); err != nil {
		return nil, err
	} else {
		windGust = v
	}

	windSpeed := units.NewValue(p.WindSpeedMph, MilesPerHour)
	if v, err := windSpeed.Convert(MetersPerSecond); err != nil {
		return nil, err
	} else {
		windSpeed = v
	}

	wd := WeatherData{
		Passkey:            p.Passkey,
		AbsolutePressure:   absPressure.Float(),
		RelativePressure:   relPressure.Float(),
		Timestamp:          time.Time(p.DateUTC).UTC(),
		Frequency:          p.Freq,
		Heap:               p.Heap,
		DailyRain:          dailyRain.Float(),
		EventRain:          eventRain.Float(),
		MonthlyRain:        monthlyRain.Float(),
		RainRate:           rainRate.Float(),
		TotalRain:          totalRain.Float(),
		WeeklyRain:         weeklyRain.Float(),
		YearlyRain:         yearlyRain.Float(),
		OutdoorHumidity:    p.Humidity,
		IndoorHumidity:     p.HumidityIn,
		Interval:           time.Duration(p.Interval) * time.Second,
		Model:              p.Model,
		Runtime:            p.Runtime,
		SolarRadiation:     p.SolarRadiation,
		StationType:        p.StationType,
		OutdoorTemperature: outTemp.Float(),
		IndoorTemperature:  inTemp.Float(),
		UV:                 p.UV,
		BatteryLevel:       p.Wh65Batt,
		MaxDailyGust:       maxDailyGust.Float(),
		WindDirection:      p.WindDir, // TODO check for offset
		WindGust:           windGust.Float(),
		WindSpeed:          windSpeed.Float(),
	}

	return &wd, nil
}
