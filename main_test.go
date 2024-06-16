package main

import (
	"net/url"
	"testing"
)

func TestParsePayload(t *testing.T) {
	// to generate a passkey: pwgen 32 | tr 'a-z' 'A-Z'
	payload := `PASSKEY=LA5ZAQUAHNGEDOOW0DAEROOV8VEZIETI&stationtype=EasyWeatherPro_V5.1.3&runtime=1240&dateutc=2024-06-16+16:32:08&tempinf=70.0&humidityin=48&baromrelin=29.920&baromabsin=29.565&tempf=67.8&humidity=47&winddir=196&windspeedmph=0.22&windgustmph=1.12&maxdailygust=4.47&solarradiation=142.55&uv=1&rainratein=0.000&eventrainin=0.000&hourlyrainin=0.000&dailyrainin=0.000&weeklyrainin=0.000&monthlyrainin=0.000&yearlyrainin=0.000&totalrainin=0.000&wh65batt=0&freq=868M&model=WS2900_V2.02.03&interval=60`

	urlValues, err := url.ParseQuery(payload)
	if err != nil {
		t.Fatal(err)
	}

	var msg Message
	if err := msg.ParseValues(urlValues); err != nil {
		t.Error(err)
	}
}
