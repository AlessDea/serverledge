package ms

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func printStructures(config Config, thresholds Thresholds) {
	fmt.Printf("Config:\nIdle: %+v\nDisabled: %+v\nPartialPerformance: %+v\nFullPerformance: %+v\n", config.Idle, config.Disabled, config.PartialPerformance, config.FullPerformance)
	fmt.Printf("Thresholds:\nDegraded: %+v\nCritical: %+v\nInactive: %+v\n", thresholds.Degraded, thresholds.Critical, thresholds.Inactive)
}

// isConfigEmpty checks if the Config structure is empty
func isThresholdsEmpty(thresholds Thresholds) bool {
	return thresholds == Thresholds{}
}

// setDefaultConfig sets default values for the Config structure
func setDefaultThresholds(thresholds *Thresholds) {
	thresholds.Degraded = State{
		Node: ResourceThreshold{
			CPU: 70.0,
			RAM: 70.0,
		},
		MonitoringSystem: ResourceThreshold{
			CPU: 70.0,
			RAM: 70.0,
		},
	}
	thresholds.Critical = State{
		Node: ResourceThreshold{
			CPU: 80.0,
			RAM: 80.0,
		},
		MonitoringSystem: ResourceThreshold{
			CPU: 80.0,
			RAM: 80.0,
		},
	}
	thresholds.Inactive = State{
		Node: ResourceThreshold{
			CPU: 95.0,
			RAM: 95.0,
		},
		MonitoringSystem: ResourceThreshold{
			CPU: 90.0,
			RAM: 90.0,
		},
	}
}

// isConfigEmpty checks if the Config structure is empty
func isConfigEmpty(config Config) bool {
	return config == Config{}
}

// setDefaultConfig sets default values for the Config structure
func setDefaultConfig(config *Config) {
	config.Idle = Actions{
		NodeExporter:    "pause",
		ProcessExporter: "pause",
		OtelCollector:   "pause",
		Prometheus:      "pause",
	}
	config.Disabled = Actions{
		NodeExporter:    "stop",
		ProcessExporter: "stop",
		OtelCollector:   "stop",
		Prometheus:      "stop",
	}
	config.PartialPerformance = Actions{
		NodeExporter:    "run",
		ProcessExporter: "pause",
		OtelCollector:   "pause",
		Prometheus:      "run",
	}
	config.FullPerformance = Actions{
		NodeExporter:    "run",
		ProcessExporter: "run",
		OtelCollector:   "run",
		Prometheus:      "run",
	}
}

// TODO: check if the percentages are increasing, if not return error
func loadThresholdsConfig(filename string) error {
	// Open the YAML file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Parse the YAML file
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&thresholds); err != nil {
		return fmt.Errorf("failed to decode YAML: %v", err)
	}

	// set default values if file is empty
	if isThresholdsEmpty(thresholds) {
		setDefaultThresholds(&thresholds)
	}
	// Output the parsed structure
	fmt.Printf("Parsed YAML: %+v\n", thresholds)
	return nil
}

func loadActionsConfig(filename string) error {
	// Open the YAML file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Parse the YAML file
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("failed to decode YAML: %v", err)
	}

	// set default values if file is empty
	if isConfigEmpty(config) {
		setDefaultConfig(&config)
	}

	// Output the parsed structure
	fmt.Printf("Parsed YAML: %+v\n", config)
	return nil
}
