package ms

import (
	"log"
)

func perform_on_exporter(exporter string, command CommandType) {
	switch exportersState[exporter] {
	case paused:
		if command == run {
			_ = unpauseExporter(exporter)
		} else if command == stop {
			_ = stopExporter(exporter)
		}

	case running:
		if command == pause {
			_ = pauseExporter(exporter)
		} else if command == stop {
			_ = stopExporter(exporter)
		}
	case stopped:
		if command == run {
			_ = startExporter(exporter)
		} else if command == pause {
			_ = pauseExporter(exporter)
		}
	default:
		log.Printf("unexpected main.ExporterState")
	}
}

func execute_policy() {
	switch state {
	case MsFullPerf:
		// run all
		log.Println("Node in Full Performance State")
		perform_on_exporter(ProcessExporter, config.FullPerformance.ProcessExporter)
		perform_on_exporter(NodeExporter, config.FullPerformance.NodeExporter)
		perform_on_exporter(OtelCollector, config.FullPerformance.OtelCollector)
		perform_on_exporter(Prometheus, config.FullPerformance.Prometheus)

	case MsPartialPerf:
		// for now stop Process and Otel
		log.Println("Node in Partial Performance State")
		perform_on_exporter(ProcessExporter, config.PartialPerformance.ProcessExporter)
		perform_on_exporter(OtelCollector, config.PartialPerformance.OtelCollector)
		perform_on_exporter(NodeExporter, config.PartialPerformance.NodeExporter)
		perform_on_exporter(Prometheus, config.PartialPerformance.Prometheus)

	case MsDisabled:
		// stop All
		log.Println("Node in Disabled State")
		perform_on_exporter(ProcessExporter, config.Disabled.ProcessExporter)
		perform_on_exporter(NodeExporter, config.Disabled.NodeExporter)
		perform_on_exporter(OtelCollector, config.Disabled.OtelCollector)
		perform_on_exporter(Prometheus, config.Disabled.Prometheus)

	case MsIdle:
		// pause all
		log.Println("Node in Idle State")
		perform_on_exporter(ProcessExporter, config.Idle.ProcessExporter)
		perform_on_exporter(NodeExporter, config.Idle.NodeExporter)
		perform_on_exporter(OtelCollector, config.Idle.OtelCollector)
		perform_on_exporter(Prometheus, config.Idle.Prometheus)
	}
}

func change_state() {
	execute_policy()
}

func check_change_state(nodeCPUUsage float64, nodeRAMUsage float64) {
	checkNodeThreshold(nodeCPUUsage, nodeRAMUsage)
	if state != prevState {
		change_state()
	}
}

// func checkDefaultPolicy(nodeCPUUsage float64, nodeRAMUsage float64) {
// 	if nodeCPUUsage > 95 && nodeRAMUsage > 95 {
// 		nodeState = StateInactive
// 		prevState = state
// 		state = MsDisabled
// 	} else if nodeCPUUsage > 80 && nodeRAMUsage > 80 {
// 		nodeState = StateCritical
// 		prevState = state
// 		state = MsIdle
// 	} else if nodeCPUUsage > 70 && nodeRAMUsage > 70 {
// 		nodeState = StateDegraded
// 		prevState = state
// 		state = MsPartialPerf
// 	} else {
// 		nodeState = StateNormal
// 		prevState = state
// 		state = MsFullPerf
// 	}

// 	if state != prevState {
// 		change_state()
// 	}
// }

func checkNodeThreshold(nodeCPUUsage float64, nodeRAMUsage float64) {
	if nodeCPUUsage > thresholds.Inactive.Node.CPU && nodeRAMUsage > thresholds.Inactive.Node.RAM {
		nodeState = StateInactive
		prevState = state
		state = MsDisabled
	} else if nodeCPUUsage > thresholds.Critical.Node.CPU && nodeRAMUsage > thresholds.Critical.Node.RAM {
		nodeState = StateCritical
		prevState = state
		state = MsIdle
	} else if nodeCPUUsage > thresholds.Degraded.Node.CPU && nodeRAMUsage > thresholds.Degraded.Node.RAM {
		nodeState = StateDegraded
		prevState = state
		state = MsPartialPerf
	} else {
		nodeState = StateNormal
		prevState = state
		state = MsFullPerf
	}
}
