package ms

import (
	"log"
	"time"
)

func changeScrapingInterval(interval int64) {
	timerMu.Lock()
	ScrapingInterval = interval * int64(time.Second)
	timerMu.Unlock()
}

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
	log.Println("Executing policy")
	switch state {
	case MsFullPerf:
		log.Println("Node in Full Performance State")
		perform_on_exporter(ProcessExporter, config.FullPerformance.ProcessExporter)
		perform_on_exporter(NodeExporter, config.FullPerformance.NodeExporter)
		perform_on_exporter(CAdvisor, config.FullPerformance.CAdvisor)
		perform_on_exporter(Prometheus, config.FullPerformance.Prometheus)
		changeScrapingInterval(config.FullPerformance.ScrapeInterval)

	case MsPartialPerf:
		log.Println("Node in Partial Performance State")
		perform_on_exporter(ProcessExporter, config.PartialPerformance.ProcessExporter)
		perform_on_exporter(CAdvisor, config.PartialPerformance.CAdvisor)
		perform_on_exporter(NodeExporter, config.PartialPerformance.NodeExporter)
		perform_on_exporter(Prometheus, config.PartialPerformance.Prometheus)
		changeScrapingInterval(config.PartialPerformance.ScrapeInterval)

	case MsDisabled:
		log.Println("Node in Disabled State")
		perform_on_exporter(ProcessExporter, config.Disabled.ProcessExporter)
		perform_on_exporter(NodeExporter, config.Disabled.NodeExporter)
		perform_on_exporter(CAdvisor, config.Disabled.CAdvisor)
		perform_on_exporter(Prometheus, config.Disabled.Prometheus)
		changeScrapingInterval(config.Disabled.ScrapeInterval)

	case MsIdle:
		log.Println("Node in Idle State")
		perform_on_exporter(ProcessExporter, config.Idle.ProcessExporter)
		perform_on_exporter(NodeExporter, config.Idle.NodeExporter)
		perform_on_exporter(CAdvisor, config.Idle.CAdvisor)
		perform_on_exporter(Prometheus, config.Idle.Prometheus)
		changeScrapingInterval(config.Idle.ScrapeInterval)

	}
}

func change_state() {
	execute_policy()
}

func check_change_state(nodeCPUUsage float64, nodeRAMUsage float64, msCPUUsage float64, msRAMUsage float64) {
	checkNodeThreshold(nodeCPUUsage, nodeRAMUsage)
	if state != prevState {
		log.Println("Changing state due to Node policy")
		log.Printf("policy_executor::state::change [%d] change state: %s -> %s\n", time.Now().UnixMilli(), prevState, state)

		change_state()
		return
	}

	checkMSThreshold(msCPUUsage, msRAMUsage)
	if state != prevState {
		log.Println("Changing state due to MS Policy")
		log.Printf("policy_executor::state::change [%d] change state: %s -> %s\n", time.Now().UnixMilli(), prevState, state)

		change_state()
		return
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
	log.Printf("policy_executor::state::usage nodeCPUUsage: %4.f | nodeRAMUsage: %4.f\n", nodeCPUUsage, nodeRAMUsage)
	if nodeCPUUsage > thresholds.Inactive.Node.CPU || nodeRAMUsage > thresholds.Inactive.Node.RAM {
		nodeState = StateInactive
		prevState = state
		state = MsDisabled
	} else if nodeCPUUsage > thresholds.Critical.Node.CPU || nodeRAMUsage > thresholds.Critical.Node.RAM {
		nodeState = StateCritical
		prevState = state
		state = MsIdle
	} else if nodeCPUUsage > thresholds.Degraded.Node.CPU || nodeRAMUsage > thresholds.Degraded.Node.RAM {
		nodeState = StateDegraded
		prevState = state
		state = MsPartialPerf
	} else {
		nodeState = StateNormal
		prevState = state
		state = MsFullPerf
	}
}

func checkMSThreshold(MSCPUUsage float64, MSRAMUsage float64) {
	log.Printf("policy_executor::state::usage MSCPUUsage: %4.f | MSRAMUsage: %4.f\n", MSCPUUsage, MSRAMUsage)
	if MSCPUUsage > thresholds.Inactive.MonitoringSystem.CPU || MSRAMUsage > thresholds.Inactive.MonitoringSystem.RAM {
		nodeState = StateInactive
		prevState = state
		state = MsDisabled
	} else if MSCPUUsage > thresholds.Critical.MonitoringSystem.CPU || MSRAMUsage > thresholds.Critical.MonitoringSystem.RAM {
		nodeState = StateCritical
		prevState = state
		state = MsIdle
	} else if MSCPUUsage > thresholds.Degraded.MonitoringSystem.CPU || MSRAMUsage > thresholds.Degraded.MonitoringSystem.RAM {
		nodeState = StateDegraded
		prevState = state
		state = MsPartialPerf
	} else {
		nodeState = StateNormal
		prevState = state
		state = MsFullPerf
	}
}
