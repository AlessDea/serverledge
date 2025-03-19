package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/container"

	// "github.com/grussorusso/serverledge/internal/container"
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

// var cpuProcModeStats = make(map[string][]float64) // Mappa per memorizzare i valori per ogni modalità per il processo
var cpuProcModeStats = make(map[string]map[string]float64)

// Mappa per memorizzare i valori precedenti per ogni modalità
var previousValues = make(map[string]float64)

// Mappa per memorizzare i tempi precedenti per calcolo differenziale
var previousTimestamps = make(map[string]time.Time)

var prevProcessValues = make(map[string]float64)
var previousProcTimestamps = make(map[string]time.Time)

// Struttura per rappresentare le metriche di un container
type ContainerMetrics struct {
	CPUUsage    float64
	MemoryUsage float64
}

// Node Exporter and Process Exporter endpoints
const (
	nodeExporterURL     = "http://localhost:9100/metrics"
	processExporterURL  = "http://localhost:9256/metrics"
	cAdvisorExporterURL = "http://localhost:8080/metrics"
)

func Init() {
	mux := http.NewServeMux()

	if config.GetBool(config.METRICS_ENABLED, false) {
		log.Println("Metrics enabled.")
		Enabled = true
	} else {
		Enabled = false
		return
	}

	Enabled = true
	nodeIdentifier = node.NodeIdentifier

	// metrics registration
	registerNodeMetrics()
	registerProcessMetrics()
	registerGlobalMetrics()

	go func() {
		for {
			log.Println("COLLECTING")
			metricsCollector()
			time.Sleep(10 * time.Second)
		}
	}()

	// Espone le metriche su /metrics per Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	// Esponi le metriche in formato JSON (prometheus non supporta JSON)
	mux.HandleFunc("/metrics/json", func(w http.ResponseWriter, r *http.Request) {
		metricFamilies, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			http.Error(w, "Errore nella raccolta delle metriche", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		jsonMetrics := make(map[string]interface{})

		for _, mf := range metricFamilies {
			metricName := mf.GetName()
			jsonMetrics[metricName] = mf
		}

		if err := json.NewEncoder(w).Encode(jsonMetrics); err != nil {
			http.Error(w, "Errore nella codifica JSON", http.StatusInternalServerError)
		}
	})

	port := ":" + strconv.Itoa(config.GetInt(config.METRICS_EXPORT_PORT, 2112)) //":2112" // Porta per Prometheus
	// Creazione del server con timeout configurati
	Server := &http.Server{
		Addr:              port,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,  // Timeout per la lettura della richiesta
		WriteTimeout:      10 * time.Second, // Timeout per la scrittura della risposta
		IdleTimeout:       15 * time.Second, // Timeout per connessioni inattive
		ReadHeaderTimeout: 2 * time.Second,  // Timeout per la lettura degli header
	}

	log.Println("Esportazione delle metriche su localhost" + port)
	log.Fatal(Server.ListenAndServe())

	// add functions to be monitored
	container.ContainersPerFunctions["lif"] = 0
	container.ContainersPerFunctions["kmeans"] = 0
	container.ContainersPerFunctions["rsa"] = 0
	container.ContainersPerFunctions["imgp"] = 0
	container.ContainersPerFunctions["msort"] = 0
	container.ContainersPerFunctions["dd"] = 0

	// handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
	// 	EnableOpenMetrics: true})
	// http.Handle("/metrics", handler)
	// err := http.ListenAndServe("127.0.0.1:2112", nil)
	// if err != nil {
	// 	log.Printf("Listen and serve terminated with error: %s\n", err)
	// 	return
	// }
}

// Functions metrics
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
	// Duration
	ExecutionTimes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_exectime",
		Help: "Function duration",
	}, []string{"node", "function"})

	IsColdStart = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_is_cold_start",
		Help: "Whether the container was cold or not",
	}, []string{"node", "function"})

	InitTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_init_time",
		Help: "Container initialization time",
	}, []string{"node", "function"})

	ResponseTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_response_time",
		Help: "Container response time",
	}, []string{"node", "function"})

	FuncCPUTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_func_cpu_total",
		Help: "Function's container total cpu time",
	}, []string{"node", "function"})

	FuncMemTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sedge_func_mem_total",
		Help: "Function's container total mem time",
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
		[]string{"groupnames", "mode"},
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

func AddFunctionWarmStart(funcName string, isWarm bool) {
	if !isWarm {
		IsColdStart.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(1)
	} else {
		IsColdStart.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(0)
	}
}

func AddFunctionInitTime(funcName string, time float64) {
	//ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
	InitTime.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(time)
}

func AddFunctionResponseTime(funcName string, time float64) {
	//ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
	ResponseTime.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(time)
}

func AddFunctionCPUTotal(funcName string, usage float64) {
	//ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
	FuncCPUTotal.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(usage)
}

func AddFunctionMemTotal(funcName string, usage float64) {
	//ExecutionTimes.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Observe(duration)
	FuncMemTotal.With(prometheus.Labels{"function": funcName, "node": nodeIdentifier}).Set(usage)
}

func registerGlobalMetrics() {
	prometheus.Register(CompletedInvocations)
	prometheus.Register(ExecutionTimes)
	prometheus.Register(IsColdStart)
	prometheus.Register(InitTime)
	prometheus.Register(ResponseTime)
	prometheus.Register(FuncCPUTotal)
	prometheus.Register(FuncMemTotal)
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
	collectCAdvisorMetrics()
}

func collectNodeMetrics() {
	// resp, err := http.Get(nodeExporterURL)
	client := http.Client{
		Timeout: 5 * time.Second, // Timeout totale della richiesta
	}
	resp, err := client.Get(nodeExporterURL)
	if err != nil {
		log.Printf("Error retrieving metrics from Node Exporter: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Println("parsing NODE metrics")
	parseMetrics(resp, []string{
		"node_cpu_seconds_total",
		"node_cpu_core",
		"node_memory_MemAvailable_bytes",
		"node_memory_MemFree_bytes",
		"node_memory_MemTotal_bytes",
	})
}

func collectProcessMetrics() {
	// resp, err := http.Get(processExporterURL)
	client := http.Client{
		Timeout: 5 * time.Second, // Timeout totale della richiesta
	}
	resp, err := client.Get(processExporterURL)
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di Process Exporter: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Println("parsing PROCESS metrics")
	parseMetrics(resp, []string{
		"namedprocess_namegroup_cpu_seconds_total",
		"namedprocess_namegroup_memory_bytes",
	})
}

func collectCAdvisorMetrics() {
	// TODO: read function names from registry
	rsa, err := GetContainerMetrics("rsa")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("rsa", 0)
		AddFunctionMemTotal("rsa", 0)
	} else {
		AddFunctionCPUTotal("rsa", rsa.CPUUsage)
		AddFunctionMemTotal("rsa", rsa.MemoryUsage)
		log.Printf("cAdvisor: %2.f", rsa.CPUUsage)
	}
	kmc, err := GetContainerMetrics("kmeans")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("kmeans", 0)
		AddFunctionMemTotal("kmeans", 0)
	} else {
		AddFunctionCPUTotal("kmeans", kmc.CPUUsage)
		AddFunctionMemTotal("kmeans", kmc.MemoryUsage)
	}
	lif, err := GetContainerMetrics("lif")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("lif", 0)
		AddFunctionMemTotal("lif", 0)
	} else {
		AddFunctionCPUTotal("lif", lif.CPUUsage)
		AddFunctionMemTotal("lif", lif.MemoryUsage)
	}
	dd, err := GetContainerMetrics("dd")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("dd", 0)
		AddFunctionMemTotal("dd", 0)
	} else {
		AddFunctionCPUTotal("dd", dd.CPUUsage)
		AddFunctionMemTotal("dd", dd.MemoryUsage)
	}
	msort, err := GetContainerMetrics("msort")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("msort", 0)
		AddFunctionMemTotal("msort", 0)
	} else {
		AddFunctionCPUTotal("msort", msort.CPUUsage)
		AddFunctionMemTotal("msort", msort.MemoryUsage)
	}
	imgp, err := GetContainerMetrics("imgp")
	if err != nil {
		log.Printf("Errore nel recuperare le metriche di cAdvisor: %v", err)
		AddFunctionCPUTotal("imgp", 0)
		AddFunctionMemTotal("imgp", 0)
	} else {
		AddFunctionCPUTotal("imgp", imgp.CPUUsage)
		AddFunctionMemTotal("imgp", imgp.MemoryUsage)
	}
	log.Println("parsing CADVISOR metrics")
	// ...

}

func GetContainerMetrics(containerName string) (ContainerMetrics, error) {

	// resp, err := http.Get(cAdvisorExporterURL)
	client := http.Client{
		Timeout: 5 * time.Second, // Timeout totale della richiesta
	}

	resp, err := client.Get(cAdvisorExporterURL)
	if err != nil {
		return ContainerMetrics{}, fmt.Errorf("Errore richiesta cAdvisor: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ContainerMetrics{}, fmt.Errorf("Errore lettura risposta cAdvisor: %v", err)
	}

	lines := strings.Split(string(body), "\n")
	var cpuUsage, memUsage float64
	cpuUsage = 0.0
	memUsage = 0.0

	cpuRegex := regexp.MustCompile(fmt.Sprintf(`container_cpu_usage_seconds_total\{.*name="%s[0-9]*".*\} ([0-9.]+)`, containerName))
	memRegex := regexp.MustCompile(fmt.Sprintf(`container_memory_usage_bytes\{.*name="%s[0-9]*".*\} ([0-9.]+)`, containerName))

	for _, line := range lines {
		if match := cpuRegex.FindStringSubmatch(line); match != nil {
			tmp := cpuUsage
			cpuUsage, _ = strconv.ParseFloat(match[1], 64)
			cpuUsage += tmp
		}

		if match := memRegex.FindStringSubmatch(line); match != nil {
			tmp := memUsage
			memUsage, _ = strconv.ParseFloat(match[1], 64)
			memUsage += tmp

		}
	}

	return ContainerMetrics{
		CPUUsage:    cpuUsage,
		MemoryUsage: memUsage,
	}, nil
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
					log.Printf("node_memory_MemTotal_bytes %.5f", value)
					nodeMemoryMemTotal.Set(value)
					break
				case "namedprocess_namegroup_cpu_seconds_total":
					mode := m.GetLabel()[1].GetValue()
					proc := m.GetLabel()[0].GetValue()
					//log.Printf("metric: %v\n", m)
					if mode == "" {
						log.Printf("Mode label not found for metric: %v\n", m)
						continue
					}

					value := *m.Counter.Value

					// Inizializza il vettore per la modalità se non esiste
					if _, exists := cpuModeStats[proc]; !exists {
						cpuModeStats[proc] = []float64{}
					}
					//log.Printf("Values for mode %s: %v", mode, value)

					cpuProcModeStats[proc][mode] = value //= append(cpuProcModeStats[mode], value)
					processCPUSecondsTotalVec.WithLabelValues(proc, mode).Set(value)

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
				log.Printf("mode: %s: Current value %.5f - Prev value %.5f\ndelta time %.5f\n", mode, currentValue, prevValue, deltaTime)
			}

			if deltaTime > 0 && deltaValue > 0 {
				percentage := (deltaValue) // / deltaTime)
				nodeCPUSecondsTotalVec.WithLabelValues(mode).Set(percentage)
			}
			// else if deltaTime > 0 {
			// 	nodeCPUSecondsTotalVec.WithLabelValues(mode).Set(currentValue)
			// }
		}

		if currentValue != 0 {
			previousValues[mode] = currentValue
		}
		previousTimestamps[mode] = currentTime
	}

	// calculate and set average for the process (monitoring system) cpu usage
	// for _, mode := range proc_modes {
	// average := calculateAverage(cpuProcModeStats[mode])
	// // currentValue := m.GetCounter().GetValue()
	// currentValue := average

	// if prevValue, exists := prevProcessValues[mode]; exists {
	// 	prevTime := previousProcTimestamps[mode]
	// 	deltaValue := currentValue - prevValue
	// 	deltaTime := currentTime.Sub(prevTime).Seconds()

	// 	if currentValue > prevValue {
	// 		// log.Printf("mode: %s: Current value %.5f - Prev value %.5f\ndelta time %.5f\n", mode, currentValue, prevValue, deltaTime)
	// 	}

	// 	if deltaTime > 0 && deltaValue > 0 {
	// 		percentage := (deltaValue) // / deltaTime)
	// 		processCPUSecondsTotalVec.WithLabelValues(mode).Set(percentage)
	// 	} else if deltaTime > 0 {
	// 		processCPUSecondsTotalVec.WithLabelValues(mode).Set(currentValue)
	// 	}
	// }

	// // if currentValue != 0 {
	// prevProcessValues[mode] = currentValue
	// // }
	// previousProcTimestamps[mode] = currentTime
	// }
}
