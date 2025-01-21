package ms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Struct per rappresentare le informazioni sui container con diverse metriche
type ContainerInfo struct {
	Name  string `json:"name"`
	Stats []struct {
		CPU struct {
			Usage struct {
				Total  int64 `json:"total"`
				User   int64 `json:"user"`
				System int64 `json:"system"`
			} `json:"usage"`
		} `json:"cpu"`
		Memory struct {
			Usage      int64 `json:"usage"`
			RSS        int64 `json:"rss"`
			Cache      int64 `json:"cache"`
			WorkingSet int64 `json:"working_set"`
		} `json:"memory"`
		Network struct {
			Interfaces []struct {
				Name     string `json:"name"`
				RxBytes  int64  `json:"rx_bytes"`
				TxBytes  int64  `json:"tx_bytes"`
				RxErrors int64  `json:"rx_errors"`
				TxErrors int64  `json:"tx_errors"`
			} `json:"interfaces"`
		} `json:"network"`
		Filesystem []struct {
			Device    string `json:"device"`
			Usage     int64  `json:"usage"`
			Capacity  int64  `json:"capacity"`
			Available int64  `json:"available"`
		} `json:"filesystem"`
	} `json:"stats"`
}

func main() {
	// URL of the cAdvisor endpoint
	url := "http://localhost:8080/api/v1.3/docker"

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var containers []ContainerInfo
	err = json.Unmarshal(body, &containers)
	if err != nil {
		panic(err)
	}

	// Print the container information
	for _, container := range containers {
		fmt.Printf("Nome Container: %s\n", container.Name)
		if len(container.Stats) > 0 {
			// CPU Metrics
			fmt.Printf("CPU Usage Total: %d\n", container.Stats[0].CPU.Usage.Total)
			fmt.Printf("CPU Usage User: %d\n", container.Stats[0].CPU.Usage.User)
			fmt.Printf("CPU Usage System: %d\n", container.Stats[0].CPU.Usage.System)

			// Memory Metrics
			fmt.Printf("Memory Usage: %d\n", container.Stats[0].Memory.Usage)
			fmt.Printf("Memory RSS: %d\n", container.Stats[0].Memory.RSS)
			fmt.Printf("Memory Cache: %d\n", container.Stats[0].Memory.Cache)
			fmt.Printf("Memory Working Set: %d\n", container.Stats[0].Memory.WorkingSet)

			// Network Metrics
			for _, iface := range container.Stats[0].Network.Interfaces {
				fmt.Printf("Network Interface: %s\n", iface.Name)
				fmt.Printf("RxBytes: %d, TxBytes: %d\n", iface.RxBytes, iface.TxBytes)
				fmt.Printf("RxErrors: %d, TxErrors: %d\n", iface.RxErrors, iface.TxErrors)
			}

			// Filesystem Metrics
			for _, fs := range container.Stats[0].Filesystem {
				fmt.Printf("Filesystem Device: %s\n", fs.Device)
				fmt.Printf("Usage: %d, Capacity: %d, Available: %d\n", fs.Usage, fs.Capacity, fs.Available)
			}
		}
	}
}
