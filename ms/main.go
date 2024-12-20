package main

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
	"time"

	"gopkg.in/yaml.v2"
)

func readMetricsConfig(filePath string) (map[string]float64, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// log.Println("File content:") // Print content for debugging
	// content, err := os.ReadFile(filePath)
	// fmt.Println(string(content))
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
	log.Printf("Successfully ran script: %s", startScriptPath)

	exportersState[NodeExporter] = running
	exportersState[ProcessExporter] = running
	exportersState[OtelCollector] = running

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
	log.Printf("Successfully stopped container")

	exportersState[NodeExporter] = stopped
	exportersState[ProcessExporter] = stopped
	exportersState[OtelCollector] = stopped

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
	log.Printf("Successfully paused container")

	exportersState[NodeExporter] = paused
	exportersState[ProcessExporter] = paused
	exportersState[OtelCollector] = paused

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
	log.Printf("Successfully unpaused container")

	exportersState[NodeExporter] = running
	exportersState[ProcessExporter] = running
	exportersState[OtelCollector] = running

	return nil
}

func change_state() {
	switch state {
	case MsFullPerf:
		// run all
		log.Println("Node in Full Performance State")
		if prevState == MsIdle {
			_ = unpauseExporter(ProcessExporter)
			_ = unpauseExporter(NodeExporter)
			_ = unpauseExporter(OtelCollector)
			_ = unpauseExporter(Prometheus)
		}
		_ = startExporter(NodeExporter, ProcessExporter, OtelCollector, Prometheus)

	case MsPartialPerf:
		// for now stop Process and Otel
		log.Println("Node in Partial Performance State")
		_ = stopExporter(ProcessExporter)
		_ = stopExporter(OtelCollector)
		_ = startExporter(NodeExporter, Prometheus)

	case MsDisabled:
		// stop All
		log.Println("Node in Disabled State")
		_ = stopExporter(ProcessExporter)
		_ = stopExporter(NodeExporter)
		_ = stopExporter(OtelCollector)
		_ = stopExporter(Prometheus)

	case MsIdle:
		// stop export the metrics but for now stop all the services except prometheus
		log.Println("Node in Idle State")
		_ = pauseExporter(ProcessExporter)
		_ = pauseExporter(NodeExporter)
		_ = pauseExporter(OtelCollector)
		_ = pauseExporter(Prometheus)
	}
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
	processCPU, err := fetchMetric(URL, `namedprocess_namegroup_cpu_seconds_total{name="serverledge"}`)
	if err != nil {
		return 0, err
	}

	// Fetch total CPU time
	totalCPU, err := fetchMetric(URL, `namedprocess_namegroup_cpu_seconds_total`)
	if err != nil {
		return 0, err
	}

	// Calculate process CPU usage percentage
	if totalCPU == 0 {
		return 0, fmt.Errorf("total CPU time is zero")
	}

	processCPUUsage := (processCPU / totalCPU) * 100
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
	processMemory, err := fetchMetric(URL, `namedprocess_namegroup_memory_bytes{name="your_process_name"}`)
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

func analyzer() {
	url := "http://localhost:2112/metrics"
	for {
		// Calculate CPU and RAM usage for the node
		nodeCPUUsage, err := calculateCPUUsage(url)
		if err != nil {
			log.Printf("Error calculating node CPU usage: %v", err)
		} else {
			fmt.Printf("Node CPU Usage: %.2f%%\n", nodeCPUUsage)
		}

		nodeRAMUsage, err := calculateRAMUsage(url)
		if err != nil {
			log.Printf("Error calculating node RAM usage: %v", err)
		} else {
			fmt.Printf("Node RAM Usage: %.2f%%\n", nodeRAMUsage)
		}

		// Calculate CPU and RAM usage for a specific process: serverledge
		processCPUUsage, err := calculateProcessCPU(url)
		if err != nil {
			log.Printf("Error calculating process CPU usage: %v", err)
		} else {
			fmt.Printf("Process CPU Usage: %.2f%%\n", processCPUUsage)
		}

		processRAMUsage, err := calculateProcessRAM(url)
		if err != nil {
			log.Printf("Error calculating process RAM usage: %v", err)
		} else {
			fmt.Printf("Process RAM Usage: %.2f%%\n", processRAMUsage)
		}

		// Wait before next iteration
		time.Sleep(10 * time.Second)

		// check if change status has to be performed
		if defaultPolicy {
			checkDefaultPolicy(nodeCPUUsage, nodeRAMUsage)
		} else {
			checkCustomPolicy(nodeCPUUsage, nodeRAMUsage)
		}

		// in realtà bisogna implementare le policy
	}
}

func main() {

	// load change state policy
	err := loadPolicyConfig(policyConfigPath)
	if err != nil {
		fmt.Printf("Error loading policy: %v\n", err)
	} else {
		defaultPolicy = false
	}

	// start in a Full Performance state
	log.Println("Node in Normal state, run exporters:")
	_ = startExporter(NodeExporter, ProcessExporter, OtelCollector, Prometheus)

	// implement communication with other server nodes

	// run go routin for continuose metrics analyses in order to take decision
	go analyzer()
}
