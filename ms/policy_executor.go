package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func loadPolicyConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&policy); err != nil {
		return fmt.Errorf("failed to decode YAML: %v", err)
	}

	return nil
}

func checkCustomPolicy(nodeCPUUsage float64, nodeRAMUsage float64) {
	for _, exporter := range policy.Exporters {
		fmt.Printf("Exporter: %s\n", exporter.Name)
		if nodeCPUUsage < exporter.StartConditions.CPUUsageBelow && nodeRAMUsage < exporter.StartConditions.RAMUsageBelow {
			// start exporter
			_ = startExporter(exporter.Name)
		} else if nodeCPUUsage > exporter.StopConditions.CPUUsageAbove && nodeRAMUsage > exporter.StopConditions.RAMUsageAbove {
			// stop exporter
			_ = stopExporter(exporter.Name)
		} else {
			// if exporter is stopped run it
			if isStopped(exportersState[exporter.Name]) {
				_ = startExporter(exporter.Name)
			} else if isPaused(exportersState[exporter.Name]) {
				_ = unpauseExporter(exporter.Name)
			} else {
				// already running
			}
		}

		fmt.Printf("  Start Conditions: CPU < %.2f%%, RAM < %.2f%%\n",
			exporter.StartConditions.CPUUsageBelow, exporter.StartConditions.RAMUsageBelow)
		fmt.Printf("  Stop Conditions: CPU > %.2f%%, RAM > %.2f%%\n",
			exporter.StopConditions.CPUUsageAbove, exporter.StopConditions.RAMUsageAbove)
		fmt.Printf("  Idle Conditions: Idle for %d seconds\n", exporter.IdleConditions.IdleDurationSeconds)
	}
}

func checkDefaultPolicy(nodeCPUUsage float64, nodeRAMUsage float64) {
	if nodeCPUUsage > 95 && nodeRAMUsage > 95 {
		nodeState = StateInactive
		prevState = state
		state = MsDisabled
	} else if nodeCPUUsage > 80 && nodeRAMUsage > 80 {
		nodeState = StateCritical
		prevState = state
		state = MsIdle
	} else if nodeCPUUsage > 70 && nodeRAMUsage > 70 {
		nodeState = StateDegraded
		prevState = state
		state = MsPartialPerf
	} else {
		nodeState = StateNormal
		prevState = state
		state = MsFullPerf
	}

	if state != prevState {
		change_state()
	}
}
