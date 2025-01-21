package metrics

import (
	"log"
	"time"

	"net/http"

	"github.com/grussorusso/serverledge/internal/node"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
)

var Enabled bool
var registry = prometheus.NewRegistry()
var nodeIdentifier string

var node_modes = []string{"user", "system", "nice", "iowait", "irq", "softirq", "steal", "idle"}
var proc_modes = []string{"user", "system"}

var cpuModeStats = make(map[string][]float64) // Mappa per memorizzare i valori per ogni modalità

var cpuProcModeStats = make(map[string][]float64) // Mappa per memorizzare i valori per ogni modalità per il processo

// Mappa per memorizzare i valori precedenti per ogni modalità
var previousValues = make(map[string]float64)

// Mappa per memorizzare i tempi precedenti per calcolo differenziale
var previousTimestamps = make(map[string]time.Time)

var prevProcessValues = make(map[string]float64)
var previousProcTimestamps = make(map[string]time.Time)

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

	// // Path to the configuration file
	// configPath := "../../metrics-config.yml"

	// // Read metrics from configuration file
	// config, err := readMetricsConfig(configPath)
	// if err != nil {
	// 	log.Fatalf("Error reading metrics configuration: %v", err)
	// }

	// // Register metrics and get the gauge map
	// gaugeMap, err := registerMetrics(config)
	// if err != nil {
	// 	log.Fatalf("Error registering metrics: %v", err)
	// }

	// register global metrics
	// registerGlobalMetrics()

	// // Start a goroutine to collect metrics periodically
	// go func() {
	// 	for {
	// 		collectMetrics("http://localhost:9100/metrics", gaugeMap) // collect from Node Exporter
	// 		collectMetrics("http://localhost:9256/metrics", gaugeMap) // collect from Process Exporter
	// 		time.Sleep(10 * time.Second)
	// 	}
	// }()

	// // Expose metrics to Prometheus on /metrics
	// http.Handle("/metrics", promhttp.Handler())
	// addr := "127.0.0.1:2112" // Port for Prometheus
	// log.Println("Exposing metrics on %s", addr)
	// log.Fatal(http.ListenAndServe(addr, nil))

	// -----------

	// metrics registration
	registerNodeMetrics()
	registerProcessMetrics()
	registerGlobalMetrics()

	go func() {
		for {
			metricsCollector()
			time.Sleep(10 * time.Second)
		}
	}()

	// Espone le metriche su /metrics per Prometheus
	http.Handle("/metrics", promhttp.Handler())
	port := ":2112" // Porta per Prometheus
	log.Println("Esportazione delle metriche su localhost" + port)
	log.Fatal(http.ListenAndServe(port, nil))

	// handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
	// 	EnableOpenMetrics: true})
	// http.Handle("/metrics", handler)
	// err := http.ListenAndServe("127.0.0.1:2112", nil)
	// if err != nil {
	// 	log.Printf("Listen and serve terminated with error: %s\n", err)
	// 	return
	// }
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
var (
	//nodeCPUSecondsTotal    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_cpu_seconds_total"})
	nodeCPUSecondsTotalVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "node_cpu_seconds_total",
			Help: "Total CPU time spent in different modes",
		},
		[]string{"mode"},
	)

	nodeCPUCore            = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_cpu_core"})
	nodeMemoryMemAvailable = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_memory_MemAvailable_bytes"})
	nodeMemoryMemFree      = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_memory_MemFree_bytes"})
	nodeMemoryMemTotal     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "node_memory_MemTotal_bytes"})
	//processCPUSecondsTotal = prometheus.NewGauge(prometheus.GaugeOpts{Name: "namedprocess_namegroup_cpu_seconds_total"})
	processCPUSecondsTotalVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "namedprocess_namegroup_cpu_seconds_total",
			Help: "Total CPU time spent in different modes for the specific process",
		},
		[]string{"mode"},
	)
	processMemoryBytes = prometheus.NewGauge(prometheus.GaugeOpts{Name: "namedprocess_namegroup_memory_bytes"})
)

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

func registerNodeMetrics() {
	prometheus.MustRegister(nodeCPUSecondsTotalVec)
	prometheus.MustRegister(nodeCPUCore)
	prometheus.MustRegister(nodeMemoryMemAvailable)
	prometheus.MustRegister(nodeMemoryMemFree)
	prometheus.MustRegister(nodeMemoryMemTotal)
}

func registerProcessMetrics() {
	prometheus.MustRegister(processCPUSecondsTotalVec)
	prometheus.MustRegister(processMemoryBytes)
}

func metricsCollector() {
	collectNodeMetrics()
	collectProcessMetrics()
}

func collectNodeMetrics() {
	resp, err := http.Get(nodeExporterURL)
	if err != nil {
		log.Printf("Error retrieving metrics from Node Exporter: %v", err)
		return
	}
	defer resp.Body.Close()

	parseMetrics(resp, []string{
		"node_cpu_seconds_total",
		"node_cpu_core",
		"node_memory_MemAvailable_bytes",
		"node_memory_MemFree_bytes",
		"node_memory_MemTotal_bytes",
	})
}

func collectProcessMetrics() {
	resp, err := http.Get(processExporterURL)
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di Process Exporter: %v", err)
		return
	}
	defer resp.Body.Close()

	parseMetrics(resp, []string{
		"namedprocess_namegroup_cpu_seconds_total",
		"namedprocess_namegroup_memory_bytes",
	})
}

// Funzione di supporto per calcolare la media di un vettore di float64
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	average := sum / float64(len(values))
	return average
}

// Analyze the collected metrics
func parseMetrics(resp *http.Response, metricNames []string) {
	parser := expfmt.TextParser{}
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		log.Printf("Errore nel parsing delle metriche: %v", err)
		return
	}

	currentTime := time.Now()

	for k := range cpuModeStats {
		delete(cpuModeStats, k)
	}

	for k := range cpuProcModeStats {
		delete(cpuProcModeStats, k)
	}

	for _, metricName := range metricNames {
		if mf, ok := metrics[metricName]; ok {
			for _, m := range mf.Metric {
				value := m.GetGauge().GetValue()
				switch metricName {
				case "node_cpu_seconds_total":
					mode := m.GetLabel()[1].GetValue()

					if mode == "" {
						log.Printf("Mode label not found for metric: %v", m)
						continue
					}

					value = *m.Counter.Value
					// Inizializza il vettore per la modalità se non esiste
					if _, exists := cpuModeStats[mode]; !exists {
						cpuModeStats[mode] = []float64{}
					}
					//log.Printf("Values for mode %s: %v", mode, value)

					cpuModeStats[mode] = append(cpuModeStats[mode], value)

					// average := calculateAverage(cpuModeStats[mode])

					// if average > 0 {
					// 	nodeCPUSecondsTotalVec.WithLabelValues(mode).Set(average / 100)
					// }

				case "node_cpu_core":
					nodeCPUCore.Set(value)
				case "node_memory_MemAvailable_bytes":
					nodeMemoryMemAvailable.Set(value)
				case "node_memory_MemFree_bytes":
					nodeMemoryMemFree.Set(value)
				case "node_memory_MemTotal_bytes":
					nodeMemoryMemTotal.Set(value)
				case "namedprocess_namegroup_cpu_seconds_total":
					mode := m.GetLabel()[1].GetValue()
					//log.Printf("metric: %v\n", m)
					if mode == "" {
						log.Printf("Mode label not found for metric: %v\n", m)
						continue
					}

					value := *m.Counter.Value

					// Inizializza il vettore per la modalità se non esiste
					if _, exists := cpuModeStats[mode]; !exists {
						cpuModeStats[mode] = []float64{}
					}
					//log.Printf("Values for mode %s: %v", mode, value)

					cpuProcModeStats[mode] = append(cpuProcModeStats[mode], value)

					// deltavalue := currentValue - prevProcessValues[mode]
					// if deltavalue > 0 {
					// 	processCPUSecondsTotalVec.WithLabelValues(mode).Set(deltavalue)
					// }

					// if currentValue != 0 {
					// 	prevProcessValues[mode] = currentValue
					// }
					// previousProcTimestamps[mode] = currentTime
				case "namedprocess_namegroup_memory_bytes":
					processMemoryBytes.Set(value)
				}
			}
		}
	}

	// calculate and set average for the node cpu usage
	for _, mode := range node_modes {
		average := calculateAverage(cpuModeStats[mode])
		currentValue := average

		if prevValue, exists := previousValues[mode]; exists {
			prevTime := previousTimestamps[mode]
			deltaValue := currentValue - prevValue
			deltaTime := currentTime.Sub(prevTime).Seconds()

			if currentValue > prevValue {
				//log.Printf("mode: %s: Current value %.5f - Prev value %.5f\ndelta time %.5f\n", mode, currentValue, prevValue, deltaTime)
			}

			if deltaTime > 0 && deltaValue > 0 {
				percentage := (deltaValue) // / deltaTime)
				nodeCPUSecondsTotalVec.WithLabelValues(mode).Set(percentage)
			}
		}

		if currentValue != 0 {
			previousValues[mode] = currentValue
		}
		previousTimestamps[mode] = currentTime
	}

	// calculate and set average for the process (monitoring system) cpu usage
	for _, mode := range proc_modes {
		average := calculateAverage(cpuProcModeStats[mode])
		// currentValue := m.GetCounter().GetValue()
		currentValue := average

		if prevValue, exists := prevProcessValues[mode]; exists {
			prevTime := previousProcTimestamps[mode]
			deltaValue := currentValue - prevValue
			deltaTime := currentTime.Sub(prevTime).Seconds()

			if currentValue > prevValue {
				//log.Printf("mode: %s: Current value %.5f - Prev value %.5f\ndelta time %.5f\n", mode, currentValue, prevValue, deltaTime)
			}

			if deltaTime > 0 && deltaValue > 0 {
				percentage := (deltaValue) // / deltaTime)
				processCPUSecondsTotalVec.WithLabelValues(mode).Set(percentage)
			}
		}

		if currentValue != 0 {
			prevProcessValues[mode] = currentValue
		}
		previousProcTimestamps[mode] = currentTime
	}
}
