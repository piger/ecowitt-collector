package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/schema"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/piger/ecowitt-collector/internal/config"
)

var (
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
)

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

func sendMetrics(wd *WeatherData, pool *pgxpool.Pool, table string) error {
	columns := makeColumnString(ColumnNames)
	values := makeValuesString(ColumnNames)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx,
		fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", table, columns, values),
		wd.Timestamp,
		wd.StationType,
		wd.AbsolutePressure,
		wd.RelativePressure,
		wd.Frequency,
		wd.Heap,
		wd.DailyRain,
		wd.EventRain,
		wd.HourlyRain,
		wd.MonthlyRain,
		wd.RainRate,
		wd.TotalRain,
		wd.WeeklyRain,
		wd.YearlyRain,
		wd.OutdoorHumidity,
		wd.IndoorHumidity,
		wd.Interval.Seconds(),
		wd.Model,
		wd.Runtime,
		wd.SolarRadiation,
		wd.StationType,
		wd.OutdoorTemperature,
		wd.IndoorTemperature,
		wd.UV,
		wd.BatteryLevel,
		wd.MaxDailyGust,
		wd.WindDirection,
		wd.WindGust,
		wd.WindSpeed,
	); err != nil {
		return err
	}

	return nil
}

func makeHandler(logger *slog.Logger, conf config.Config, pool *pgxpool.Pool, windOffset int) http.Handler {
	formDecoder := schema.NewDecoder()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With("client", r.RemoteAddr)
		logger.Debug("station sent request")

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Warn("error parsing form data", "err", err)
			return
		}

		var p payload
		if err := formDecoder.Decode(&p, r.Form); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("error deserializing payload", "err", err)
			return
		}

		if windOffset != 0 {
			p.WindDir = offsetDegrees(p.WindDir, windOffset)
		}

		wd, err := NewWeatherData(p)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Warn("error converting payload to WeatherData", "err", err)
			return
		}

		if err := sendMetrics(wd, pool, conf.Database.Table); err != nil {
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

	if err := run(logger, conf); err != nil {
		logger.Error("fatal error", "err", err)
		os.Exit(1)
	}
}
