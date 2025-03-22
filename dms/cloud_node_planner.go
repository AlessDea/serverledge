package dms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LeaderInfo contiene le informazioni del nodo master ricevute dal nodo Edge
type LeaderInfo struct {
	IPAddress string `json:"ip_address"`
	Timestamp int64  `json:"timestamp"`
}

var (
	mu          sync.Mutex
	masterNode  string
	metricsPort = "3113"
)

type NodeMetrics struct {
	CPUUsage  float64  // Uso CPU del nodo
	MemTotal  float64  // Memoria totale del nodo
	MemUsed   float64  // Memoria usata del nodo
	Functions []string // Funzioni in esecuzione
}

var (
	nodeMetricsMap = make(map[string]NodeMetrics)
	mutex          sync.Mutex
	metricsURL     = "http://192.168.1.50:8080/metrics" // Modifica con l'IP del nodo master
)

// get Leader Election updates
func updateLeaderHandler(w http.ResponseWriter, r *http.Request) {
	var leader LeaderInfo
	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &leader)
	if err != nil {
		http.Error(w, "Errore nel parsing JSON", http.StatusBadRequest)
		return
	}

	mu.Lock()
	masterNode = leader.IPAddress
	mu.Unlock()

	log.Printf("New leader (Collector Agent) node: %s\n", masterNode)
	w.WriteHeader(http.StatusOK)
}

// Get metrics from Collector Agent
func getMetricsFromMaster() {
	mu.Lock()
	master := masterNode
	mu.Unlock()

	if master == "" {
		log.Println("Error: no leader recorded")
		return
	}

	url := fmt.Sprintf("http://%s/metrics", master)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error retrieving metrics from the master:", err)
		return
	}
	defer resp.Body.Close()

	metrics, _ := ioutil.ReadAll(resp.Body)
	log.Println("Received Metrics:\n", string(metrics))

	var metricsData map[string]json.RawMessage
	if err := json.Unmarshal(metrics, &metricsData); err != nil {
		log.Println("Errore nella deserializzazione JSON:", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for nodeName, rawMetrics := range metricsData {
		metricsInfo := NodeMetrics{}
		functionSet := make(map[string]struct{}) // Evita duplicati

		// Decodifica il JSON grezzo in una lista di Metric
		var metrics []Metric
		if err := json.Unmarshal(rawMetrics, &metrics); err != nil {
			log.Printf("❌ Errore nel parsing delle metriche per il nodo %s: %v\n", nodeName, err)
			continue
		}

		// Itera sulle metriche decodificate
		for _, metric := range metrics {
			value, err := strconv.ParseFloat(metric.Value, 64)
			if err != nil {
				log.Printf("⚠️ Errore nella conversione del valore di %s per %s\n", metric.Name, nodeName)
				continue
			}

			switch metric.Name {
			case "node_cpu_seconds_total":
				metricsInfo.CPUUsage = value
			case "node_memory_MemTotal_bytes":
				metricsInfo.MemTotal = value
			case "node_memory_MemAvailable_bytes":
				metricsInfo.MemUsed = metricsInfo.MemTotal - value // Calcola la memoria usata
			}

			// 📌 Controlla se la metrica riguarda una funzione
			if strings.HasPrefix(metric.Name, "sedge_") {
				if funcName, exists := metricFunctionName(metric.Name); exists {
					functionSet[funcName] = struct{}{}
				}
			}
		}

		// Converte la mappa in lista di funzioni
		for funcName := range functionSet {
			metricsInfo.Functions = append(metricsInfo.Functions, funcName)
		}

		// Salva i dati analizzati nella mappa
		nodeMetricsMap[nodeName] = metricsInfo
	}
	log.Println("📊 Analisi completata! Metriche aggiornate.")
}

func metricFunctionName(metricName string) (string, bool) {
	if strings.HasPrefix(metricName, "sedge_") {
		parts := strings.Split(metricName, "_")
		if len(parts) > 2 {
			return parts[2], true // Il nome della funzione è la terza parte
		}
	}
	return "", false
}

func printNodeMetrics() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Println("\nNodes:")
	for node, metrics := range nodeMetricsMap {
		fmt.Printf("🔹 Nodo: %s\n", node)
		fmt.Printf("   🔸 CPU Usage: %.2f sec\n", metrics.CPUUsage)
		fmt.Printf("   🔸 Mem Usage: %.2f MB\n", metrics.MemUsed/1e6)
		fmt.Printf("   🔸 Functions: %v\n\n", metrics.Functions)
	}
}

func InitCloudPlanner() {

	http.HandleFunc("/update-leader", updateLeaderHandler)

	go func() {
		for {
			time.Sleep(10 * time.Second)
			getMetricsFromMaster()
			printNodeMetrics()
		}
	}()

	log.Println("Cloud node listening for Elected Leader on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
