package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	Passkey        string
	BaromAbsIn     float64
	BaromRelIn     float64
	DailyRainIn    float64
	DateUTC        time.Time
	EventRainIn    float64
	Freq           string
	Heap           int
	HourlyRainIn   float64
	Humidity       int
	HumidityIn     int
	Interval       int
	MaxDailyGust   float64
	Model          string
	MonthlyRainIn  float64
	RainRateIn     float64
	Runtime        int
	SolarRadiation float64
	StationType    string
	Tempf          float64
	TempInF        float64
	TotalRainIn    float64
	UV             float64 // or int?
	WeeklyRainIn   float64
	Wh65Batt       float64 // or int?
	WindDir        int
	WindGustMph    float64
	WindSpeedMph   float64
	YearlyRainIn   float64
}

func (msg *Message) ParseValues(v url.Values) error {
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

func run(logger *slog.Logger, addr string) error {
	http.HandleFunc("/data/report/", func(w http.ResponseWriter, r *http.Request) {
		logger = logger.With("client", r.RemoteAddr)

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
		var msg Message
		if err := msg.ParseValues(r.Form); err != nil {
			logger.Error("error parsing form data", "err", err)
		}
		fmt.Printf("msg = %+v\n", msg)

		/*
			for key, values := range r.Form {
				fmt.Printf("Form: %s = %s\n", key, strings.Join(values, ","))
			}
		*/
	})

	logger.Info("starting server", "addr", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		return err
	}

	return nil
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

	if err := run(logger, addr); err != nil {
		log.Fatal(err)
	}
}
