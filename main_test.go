package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/schema"
)

func TestParsePayload(t *testing.T) {
	// to generate a passkey: pwgen 32 | tr 'a-z' 'A-Z'
	queryArgs := `PASSKEY=LA5ZAQUAHNGEDOOW0DAEROOV8VEZIETI&stationtype=EasyWeatherPro_V5.1.3&runtime=1240&dateutc=2024-06-16+16:32:08&tempinf=70.0&humidityin=48&baromrelin=29.920&baromabsin=29.565&tempf=67.8&humidity=47&winddir=196&windspeedmph=0.22&windgustmph=1.12&maxdailygust=4.47&solarradiation=142.55&uv=1&rainratein=0.000&eventrainin=0.000&hourlyrainin=0.000&dailyrainin=0.000&weeklyrainin=0.000&monthlyrainin=0.000&yearlyrainin=0.000&totalrainin=0.000&vpd=0.153&wh65batt=0&freq=868M&model=WS2900_V2.02.03&interval=60`

	urlValues, err := url.ParseQuery(queryArgs)
	if err != nil {
		t.Fatal(err)
	}

	formDecoder := schema.NewDecoder()
	var p payload
	if err := formDecoder.Decode(&p, urlValues); err != nil {
		t.Fatalf("error decoding form data: %s", err)
	}

	wantTempInF := 70.0
	if p.TempInF != wantTempInF {
		t.Fatalf("expected %v, got %v", wantTempInF, p.TempInF)
	}

	wantDate, err := time.Parse(time.DateTime, "2024-06-16 16:32:08")
	if err != nil {
		t.Error(err)
	}
	if wantDate.Compare(time.Time(p.DateUTC)) != 0 {
		t.Fatalf("expected %+v, got %+v", wantDate, p.DateUTC)
	}
}

func TestOffsetDegrees(t *testing.T) {
	type args struct {
		i      int
		offset int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "0",
			args: args{i: 0, offset: -90},
			want: 270,
		},
		{
			name: "1",
			args: args{i: 207, offset: -90},
			want: 117,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := offsetDegrees(tt.args.i, tt.args.offset); got != tt.want {
				t.Errorf("offsetDegrees() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindDegreesToName(t *testing.T) {
	tests := []struct {
		Deg int
		Dir string
	}{
		{Deg: 1, Dir: "N"},
		{Deg: 359, Dir: "N"},
		{Deg: 0, Dir: "N"},
		{Deg: 180, Dir: "S"},
		{Deg: 12, Dir: "NNE"},
	}

	for _, tt := range tests {
		t.Run(tt.Dir, func(t *testing.T) {
			got, err := windDegreesToName(tt.Deg)
			if err != nil {
				t.Error(err)
			}

			if got != tt.Dir {
				t.Fatalf("got %q, want %q", got, tt.Dir)
			}
		})
	}
}
