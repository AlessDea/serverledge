package metrics

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v3"
)

// Config structure to parse the metrics configuration file
type Config struct {
	NodeExporter    []string `yaml:"node_exporter"`
	ProcessExporter []string `yaml:"process_exporter"`
}

// Reads metrics from the configuration file
func readMetricsConfig(filePath string) (*Config, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// log.Println("File content:") // Print content for debugging
	// content, err := os.ReadFile(filePath)
	// fmt.Println(string(content))
	log.Println("File content:")
	log.Println(string(content)) // Stampa il contenuto del file

	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding YAML file: %w", err)
	}

	return &config, nil
}

// Registers the metrics from the configuration into a gauge map
func registerMetrics(config *Config) (map[string]prometheus.Gauge, error) {
	gaugeMap := make(map[string]prometheus.Gauge)

	// Register Node Exporter metrics
	for _, metric := range config.NodeExporter {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metric,
			Help: fmt.Sprintf("Gauge for %s", metric),
		})
		if err := prometheus.Register(gauge); err != nil {
			return nil, fmt.Errorf("error registering metric %s: %w", metric, err)
		}
		gaugeMap[metric] = gauge
	}

	// Register Process Exporter metrics
	for _, metric := range config.ProcessExporter {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metric,
			Help: fmt.Sprintf("Gauge for %s", metric),
		})
		if err := prometheus.Register(gauge); err != nil {
			return nil, fmt.Errorf("error registering metric %s: %w", metric, err)
		}
		gaugeMap[metric] = gauge
	}

	return gaugeMap, nil
}

// Collect metrics from a given exporter URL and update the gauge map
func collectMetrics(exporterURL string, gaugeMap map[string]prometheus.Gauge) {
	resp, err := http.Get(exporterURL)
	if err != nil {
		log.Printf("Error retrieving metrics from %s: %v", exporterURL, err)
		return
	}
	defer resp.Body.Close()

	// Parse metrics and update gauges
	parser := expfmt.TextParser{}
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		log.Printf("Error parsing metrics from %s: %v", exporterURL, err)
		return
	}

	// Iterate over the gaugeMap and update corresponding metrics
	for metricName, gauge := range gaugeMap {
		if mf, exists := metrics[metricName]; exists {
			for _, m := range mf.Metric {
				gauge.Set(m.GetGauge().GetValue())
				log.Printf("Updated metric %s with value %f", metricName, m.GetGauge().GetValue())
			}
		} else {
			log.Printf("Metric %s not found in response from %s", metricName, exporterURL)
		}
	}
}

// Main function
// func main() {
// 	// Path to the configuration file
// 	configPath := "metrics-config.yml"

// 	// Read metrics from configuration file
// 	config, err := readMetricsConfig(configPath)
// 	if err != nil {
// 		log.Fatalf("Error reading metrics configuration: %v", err)
// 	}

// 	// Register metrics and get the gauge map
// 	gaugeMap, err := registerMetrics(config)
// 	if err != nil {
// 		log.Fatalf("Error registering metrics: %v", err)
// 	}

// 	// Start a goroutine to collect metrics periodically
// 	go func() {
// 		for {
// 			collectMetrics("http://localhost:9100/metrics", gaugeMap) // Node Exporter
// 			collectMetrics("http://localhost:9256/metrics", gaugeMap) // Process Exporter
// 			time.Sleep(10 * time.Second)
// 		}
// 	}()

// 	// Expose metrics to Prometheus on /metrics
// 	http.Handle("/metrics", promhttp.Handler())
// 	log.Println("Exposing metrics on http://localhost:2112/metrics")
// 	log.Fatal(http.ListenAndServe(":2112", nil))
// }
