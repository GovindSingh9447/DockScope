package handlers

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

var influxClient influxdb2.Client
var writeAPI api.WriteAPIBlocking

const (
	influxURL    = "http://localhost:8086"
	influxToken  = "admintoken123"
	influxOrg    = "dockscope"
	influxBucket = "dockscope-bucket"
)

func InitInflux() {
	influxClient = influxdb2.NewClient(influxURL, influxToken)
	writeAPI = influxClient.WriteAPIBlocking(influxOrg, influxBucket)
	fmt.Println("✅ InfluxDB client initialized")
}

func WriteMetricToInflux(hostID, containerID, name, image string, cpu float64, mem float64, restartCount int, ts time.Time) error {
	point := influxdb2.NewPointWithMeasurement("container_metrics").
		AddTag("host_id", hostID).
		AddTag("container_id", containerID).
		AddTag("name", name).
		AddTag("image", image).
		AddField("cpu", cpu).
		AddField("memory", mem).
		AddField("restart_count", restartCount).
		SetTime(ts)

	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		fmt.Printf("❌ Failed to write metric to InfluxDB (container %s): %v\n", containerID, err)
	} else {
		fmt.Printf("✅ Wrote metric to InfluxDB for container: %s\n", containerID)
	}
	return err
}

func WriteLogToInflux(hostID, containerID, level, logLine string, ts time.Time) error {
	point := influxdb2.NewPointWithMeasurement("container_logs").
		AddTag("host_id", hostID).
		AddTag("container_id", containerID).
		AddTag("level", level).
		AddField("log", logLine).
		SetTime(ts)

	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		fmt.Printf("❌ Failed to write log to InfluxDB (container %s): %v\n", containerID, err)
	} else {
		fmt.Printf("✅ Wrote log to InfluxDB for container: %s\n", containerID)
	}
	return err
}



func QueryAverageMetric(metricType, containerID, hostID string, duration time.Duration) (float64, error) {
	queryAPI := influxClient.QueryAPI(influxOrg)

	field := ""
	switch metricType {
	case "high_cpu":
		field = "cpu"
	case "high_memory":
		field = "memory"
	default:
		return 0, fmt.Errorf("unsupported metric type: %s", metricType)
	}

	query := fmt.Sprintf(`
	from(bucket: "%s")
	|> range(start: -%dm)
	|> filter(fn: (r) => r._measurement == "container_metrics" and r._field == "%s")
	|> filter(fn: (r) => r.host_id == "%s" and r.container_id == "%s")
	|> mean()
	`, influxBucket, int(duration.Minutes()), field, hostID, containerID)

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return 0, err
	}

	for result.Next() {
		return result.Record().Value().(float64), nil
	}

	if result.Err() != nil {
		return 0, result.Err()
	}

	return 0, fmt.Errorf("no data found for container %s", containerID)
}

