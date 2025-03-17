package ms

// metrics are collected and exported by servergledge itself
// while this component is the orchestrator of the Monitorying System,
// so its aim is to take decision on the metrics collection

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	bully "github.com/grussorusso/serverledge/dms/bully_algorithm"
	"gopkg.in/yaml.v2"
)

var StatesLogFile *os.File

func readMetricsConfig(filePath string) (map[string]float64, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// log.Println("File content:") // Print content for debugging
	// content, err := os.ReadFile(filePath)
	// log.Println(string(content))
	log.Println("File content:")
	log.Println(string(content)) // Stampa il contenuto del file

	var metricNames []string
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&metricNames); err != nil {
		return nil, fmt.Errorf("error decoding YAML file: %w", err)
	}

	metrics := make(map[string]float64)
	for _, name := range metricNames {
		metrics[name] = 0
	}

	return metrics, nil
}

// startExporter starts a Docker container for an exporter
func startExporter(args ...string) error {
	cmd := exec.Command(startScriptPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		log.Printf("Error running script %s: %v", startScriptPath, err)
		return err
	}
	log.Printf("Successfully ran container %s", args[len(args)-1])

	exportersState[args[len(args)-1]] = running

	return nil
}

// stopExporter stops a Docker container for an exporter
func stopExporter(args ...string) error {
	cmd := exec.Command(stopScriptPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		log.Printf("Error stopping container: %v", err)
		return err
	}
	log.Printf("Successfully stopped container %s", args[len(args)-1])

	exportersState[args[len(args)-1]] = stopped

	return nil
}

// pauseExporter pauses a Docker container for an exporter
func pauseExporter(args ...string) error {
	cmd := exec.Command(pauseScriptPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		log.Printf("Error pausing container: %v", err)
		return err
	}
	log.Printf("Successfully paused container %s", args[len(args)-1])

	exportersState[args[len(args)-1]] = paused

	return nil
}

// pauseExporter pauses a Docker container for an exporter
func unpauseExporter(args ...string) error {
	cmd := exec.Command(unpauseScriptPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		log.Printf("Error unpausing container: %v", err)
		return err
	}
	log.Printf("Successfully unpaused container %s", args[len(args)-1])

	exportersState[args[len(args)-1]] = running

	return nil
}

// fetchMetric fetches a specific metric from an exporter endpoint
func fetchMetric(endpoint, metric string) (float64, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %v", err)
	}

	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, metric) {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				return strconv.ParseFloat(parts[1], 64)
			}
		}
	}

	return 0, fmt.Errorf("metric %s not found", metric)
}

// calculateCPUUsage calculates the CPU usage percentage
func calculateCPUUsage(URL string) (float64, error) {
	// Fetch idle CPU time
	idle, err := fetchMetric(URL, `node_cpu_seconds_total{mode="idle"}`)
	if err != nil {
		return 0, err
	}

	// Fetch total CPU time (sum of all states)
	total := 0.0
	states := []string{"user", "system", "nice", "iowait", "irq", "softirq", "steal", "idle"}
	for _, state := range states {
		val, err := fetchMetric(URL, fmt.Sprintf(`node_cpu_seconds_total{mode="%s"}`, state))
		if err == nil {
			total += val
		}
	}

	if total == 0 {
		return 0, fmt.Errorf("total CPU time is zero")
	}

	// Calculate CPU usage percentage
	cpuUsage := 100 * (1 - (idle / total))
	return cpuUsage, nil
}

// calculateProcessCPU calculates the CPU usage of a specific process
func calculateProcessCPU(URL string) (float64, error) {
	// Fetch process CPU time
	proc_total := 0.0
	states := []string{"user", "system"}
	for _, state := range states {
		processCPU, err := fetchMetric(URL, fmt.Sprintf(`namedprocess_namegroup_cpu_seconds_total{mode="%s"}`, state))
		if err == nil {
			proc_total += processCPU
		}
	}

	// Fetch total CPU time (sum of all states)
	total := 0.0
	states = []string{"user", "system", "nice", "iowait", "irq", "softirq", "steal", "idle"}
	for _, state := range states {
		val, err := fetchMetric(URL, fmt.Sprintf(`node_cpu_seconds_total{mode="%s"}`, state))
		if err == nil {
			total += val
		}
	}

	if total == 0 {
		return 0, fmt.Errorf("total CPU time is zero")
	}

	processCPUUsage := (proc_total / total) * 100
	return processCPUUsage, nil
}

// calculateRAMUsage calculates the RAM usage percentage
func calculateRAMUsage(URL string) (float64, error) {
	totalRAM, err := fetchMetric(URL, "node_memory_MemTotal_bytes")
	if err != nil {
		return 0, err
	}

	availableRAM, err := fetchMetric(URL, "node_memory_MemAvailable_bytes")
	if err != nil {
		return 0, err
	}

	if totalRAM == 0 {
		return 0, fmt.Errorf("total RAM is zero")
	}

	ramUsage := (1 - (availableRAM / totalRAM)) * 100
	return ramUsage, nil
}

// calculateProcessRAM calculates the RAM usage of a specific process
func calculateProcessRAM(URL string) (float64, error) {
	processMemory, err := fetchMetric(URL, `namedprocess_namegroup_memory_bytes`)
	if err != nil {
		return 0, err
	}

	totalRAM, err := fetchMetric(URL, "node_memory_MemTotal_bytes")
	if err != nil {
		return 0, err
	}

	if totalRAM == 0 {
		return 0, fmt.Errorf("total RAM is zero")
	}

	processRAMUsage := (processMemory / totalRAM) * 100
	return processRAMUsage, nil
}

func analyzer(uc chan bully.NodeInfo) {
	url := "http://localhost:2112/metrics" // read metrics from
	for {
		// Calculate CPU and RAM usage for the node
		nodeCPUUsage, err := calculateCPUUsage(url)
		if err != nil {
			log.Printf("Error calculating node CPU usage: %v", err)
		} else {
			log.Printf("Node CPU Usage: %.2f%%\n", nodeCPUUsage)
		}

		nodeRAMUsage, err := calculateRAMUsage(url)
		if err != nil {
			log.Printf("Error calculating node RAM usage: %v", err)
		} else {
			log.Printf("Node RAM Usage: %.2f%%\n", nodeRAMUsage)
		}

		// Calculate CPU and RAM usage for a specific process: serverledge
		processCPUUsage, err := calculateProcessCPU(url)
		if err != nil {
			log.Printf("Error calculating process CPU usage: %v", err)
		} else {
			log.Printf("Process CPU Usage: %.2f%%\n", processCPUUsage)
		}

		processRAMUsage, err := calculateProcessRAM(url)
		if err != nil {
			log.Printf("Error calculating process RAM usage: %v", err)
		} else {
			log.Printf("Process RAM Usage: %.2f%%\n", processRAMUsage)
		}

		check_change_state(nodeCPUUsage, nodeRAMUsage)

		if uc != nil {
			uc <- bully.NodeInfo{Status: string(nodeState), AvailableRsrc: (nodeCPUUsage*0.4 + nodeRAMUsage*0.6)}
		}
		// Wait before next iteration
		log.Printf("Waiting")
		time.Sleep(10 * time.Second)
	}
}

func Init(wg *sync.WaitGroup, uc chan bully.NodeInfo) {
	defer wg.Done() // Segnala che la goroutine ha finito

	// Apri/crea il file di log
	logFile, err := os.OpenFile("serverledge_ms.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Errore nell'apertura del file di log: %v", err)
	}

	// apri/crea file raccolta cambio stato
	StatesLogFile, err := os.OpenFile("statesChangeLogFile.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer StatesLogFile.Close()

	// Reindirizza i log al file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Formato log con data, ora e file sorgente

	// load thresholds
	err = loadThresholdsConfig(thresholdsConfigPath)
	if err != nil {
		log.Printf("Error loading thresholds: %v\n", err)
	}
	log.Println("loaded thresholds\n")

	// load change state policy actions
	err = loadActionsConfig(policyConfigPath)
	if err != nil {
		log.Printf("Error loading thresholds: %v\n", err)
	}
	log.Println("loaded thresholds\n")

	printStructures(config, thresholds)

	// start in a Full Performance state
	log.Println("Node in Normal state, run exporters:")
	_ = startExporter(NodeExporter)
	_ = startExporter(ProcessExporter)
	_ = startExporter(Prometheus)
	_ = startExporter(CAdvisor)
	//_ = startExporter(OtelCollector)
	// implement communication with other server nodes

	// run go routin for continuose metrics analyses in order to take decision
	analyzer(uc)
}
