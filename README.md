# ecowitt-collector

A custom collector for weather data sent by the Ecowitt Weather Stations WS2910.

This program is published for documentation purposes only, as it's tailored to my own use.

## Configuration

The configuration for the collector must be provided in YAML with the following format:

```yaml
log_level: "INFO"
database:
  dsn: "postgres://<username>:<password>@<hostname>/<dbname>"
  table: "<table_name>"
http:
  address: ":8080"
```

To configure the station's panel to send weather data to this collector you can use the
[WSView Plus](https://api.ecowitt.net/api/app/download?category=WSView%20Plus) mobile app; the
collector exposes a HTTP endpoint on `/data/report/` accepting POST requests.

## Metrics

The program exposes the following metrics on the `/metrics` endpoint:

- `ecowitt_collector_requests_total`
- `ecowitt_collector_errors_total` with the `error_type` label (`parser`, `decoder`, `converter`, `db`)

## Protocol information

- [Receiving weather information in EcoWitt protocol and writing into InfluxDB and WOW](https://www.bentasker.co.uk/posts/blog/house-stuff/receiving-weather-info-from-ecowitt-weather-station-and-writing-to-influxdb.html)
- [aioecowitt](https://github.com/home-assistant-libs/aioecowitt)
- [Connecting a Weather Station to FME](https://locusglobal.com/connecting-a-weather-station-to-fme/)
