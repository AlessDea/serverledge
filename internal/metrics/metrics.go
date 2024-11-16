package metrics

import (
	"log"
	"time"

	"net/http"

	"github.com/grussorusso/serverledge/internal/node"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Enabled bool
var registry = prometheus.NewRegistry()
var nodeIdentifier string

// Node Exporter and Process Exporter endpoints
const (
	nodeExporterURL    = "http://localhost:9100/metrics"
	processExporterURL = "http://localhost:9256/metrics"
)

func Init() {
	// if config.GetBool(config.METRICS_ENABLED, false) {
	// 	log.Println("Metrics enabled.")
	// 	Enabled = true
	// } else {
	// 	Enabled = false
	// 	return
	// }

	Enabled = true
	nodeIdentifier = node.NodeIdentifier

	// --- NEW ---

	// Path to the configuration file
	configPath := "../../metrics-config.yml"

	// Read metrics from configuration file
	config, err := readMetricsConfig(configPath)
	if err != nil {
		log.Fatalf("Error reading metrics configuration: %v", err)
	}

	// Register metrics and get the gauge map
	gaugeMap, err := registerMetrics(config)
	if err != nil {
		log.Fatalf("Error registering metrics: %v", err)
	}

	// register global metrics
	registerGlobalMetrics()

	// Start a goroutine to collect metrics periodically
	go func() {
		for {
			collectMetrics("http://localhost:9100/metrics", gaugeMap) // Node Exporter
			collectMetrics("http://localhost:9256/metrics", gaugeMap) // Process Exporter
			time.Sleep(10 * time.Second)
		}
	}()

	// Expose metrics to Prometheus on /metrics
	http.Handle("/metrics", promhttp.Handler())
	addr := "127.0.0.1:2112" // Port for Prometheus
	log.Println("Exposing metrics on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

	// -----------

	// metrics registration
	// registerNodeMetrics()
	// registerProcessMetrics()

	// go func() {
	// 	for {
	// 		metricsCollector()
	// 		time.Sleep(10 * time.Second)
	// 	}
	// }()

	// Espone le metriche su /metrics per Prometheus
	// http.Handle("/mymetrics", promhttp.Handler())
	// port := ":2112" // Porta per Prometheus
	// log.Println("Esportazione delle metriche su localhost" + port)
	// log.Fatal(http.ListenAndServe(port, nil))

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true})
	http.Handle("/metrics", handler)
	err = http.ListenAndServe("127.0.0.1:2112", nil)
	if err != nil {
		log.Printf("Listen and serve terminated with error: %s\n", err)
		return
	}
}

// Global metrics
var (
	CompletedInvocations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sedge_completed_total",
		Help: "The total number of completed function invocations",
	}, []string{"node", "function"})
	// ExecutionTimes = promauto.NewHistogramVec(prometheus.HistogramOpts{
	// 	Name:    "sedge_exectime",
	// 	Help:    "Function duration",
	// 	Buckets: durationBuckets,
	// },
	ExecutionTimes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_exectime",
		Help: "Function duration",
	}, []string{"node", "function"})
)

// some metrics
// var (
// 	nodeCPUSecondsTotal    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_cpu_seconds_total"})
// 	nodeCPUCore            = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_cpu_core"})
// 	nodeMemoryMemAvailable = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_memory_MemAvailable_bytes"})
// 	nodeMemoryMemFree      = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_memory_MemFree_bytes"})
// 	processCPUSecondsTotal = prometheus.NewGauge(prometheus.GaugeOpts{Name: "namedprocess_namegroup_cpu_seconds_total"})
// 	processMemoryBytes     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "namedprocess_namegroup_memory_bytes"})
// )

var durationBuckets = []float64{0.002, 0.005, 0.010, 0.02, 0.03, 0.05, 0.1, 0.15, 0.3, 0.6, 1.0}

func AddCompletedInvocation(funcName string) {
	CompletedInvocations.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Inc()
}
func AddFunctionDurationValue(funcName string, duration float64) {
	//ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
	ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(duration)

}

func registerGlobalMetrics() {
	prometheus.Register(CompletedInvocations)
	prometheus.Register(ExecutionTimes)
}

// func registerNodeMetrics() {
// 	prometheus.MustRegister(nodeCPUSecondsTotal)
// 	prometheus.MustRegister(nodeCPUCore)
// 	prometheus.MustRegister(nodeMemoryMemAvailable)
// 	prometheus.MustRegister(nodeMemoryMemFree)
// }

// func registerProcessMetrics() {
// 	prometheus.MustRegister(processCPUSecondsTotal)
// 	prometheus.MustRegister(processMemoryBytes)
// }

// func metricsCollector() {
// 	collectNodeMetrics()
// 	collectProcessMetrics()
// }

// func collectNodeMetrics() {
// 	resp, err := http.Get(nodeExporterURL)
// 	if err != nil {
// 		log.Printf("Errore nel recuperare le metriche di Node Exporter: %v", err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	parseMetrics(resp, []string{
// 		"node_cpu_seconds_total",
// 		"node_cpu_core",
// 		"node_memory_MemAvailable_bytes",
// 		"node_memory_MemFree_bytes",
// 	})
// }

// func collectProcessMetrics() {
// 	resp, err := http.Get(processExporterURL)
// 	if err != nil {
// 		log.Printf("Errore nel recuperare le metriche di Process Exporter: %v", err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	parseMetrics(resp, []string{
// 		"namedprocess_namegroup_cpu_seconds_total",
// 		"namedprocess_namegroup_memory_bytes",
// 	})
// }

// Analyze the collected metrics
// func parseMetrics(resp *http.Response, metricNames []string) {
// 	parser := expfmt.TextParser{}
// 	metrics, err := parser.TextToMetricFamilies(resp.Body)
// 	if err != nil {
// 		log.Printf("Errore nel parsing delle metriche: %v", err)
// 		return
// 	}

// 	for _, metricName := range metricNames {
// 		if mf, ok := metrics[metricName]; ok {
// 			for _, m := range mf.Metric {
// 				value := m.GetGauge().GetValue()
// 				switch metricName {
// 				case "node_cpu_seconds_total":
// 					nodeCPUSecondsTotal.Set(value)
// 				case "node_cpu_core":
// 					nodeCPUCore.Set(value)
// 				case "node_memory_MemAvailable_bytes":
// 					nodeMemoryMemAvailable.Set(value)
// 				case "node_memory_MemFree_bytes":
// 					nodeMemoryMemFree.Set(value)
// 				case "namedprocess_namegroup_cpu_seconds_total":
// 					processCPUSecondsTotal.Set(value)
// 				case "namedprocess_namegroup_memory_bytes":
// 					processMemoryBytes.Set(value)
// 				}
// 			}
// 		}
// 	}
// }
