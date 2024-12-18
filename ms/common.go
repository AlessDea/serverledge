package main

type NodeState string // State of the node
type MSState string   // State of the monitoryng system

var startScriptPath string = "../scripts/exporters_controller/start_exporter.sh"
var stopScriptPath string = "../scripts/exporters_controller/start_exporter.sh"
var pauseScriptPath string = "../scripts/exporters_controller/pause_exporter.sh"
var unpauseScriptPath string = "../scripts/exporters_controller/unpause_exporter.sh"

var policyConfigPath string = "policy_config.py"

var state MSState = MsFullPerf // state of the MS
var prevState MSState = ""
var nodeState NodeState = StateNormal // state of the node

type ExporterState string

const (
	NodeExporter    string = "node-exporter"
	ProcessExporter string = "process-exporter"
	OtelCollector   string = "otel-collector"
	Prometheus      string = "prometheus"
)

const (
	running ExporterState = "running"
	paused  ExporterState = "paused"
	stopped ExporterState = "stopped"
)

func isRunning(state ExporterState) bool {
	if state == running {
		return true
	}
	return false
}

func isPaused(state ExporterState) bool {
	if state == paused {
		return true
	}
	return false
}

func isStopped(state ExporterState) bool {
	if state == stopped {
		return true
	}
	return false
}

const (
	StateNormal   NodeState = "Normal"
	StateDegraded NodeState = "Degraded"
	StateCritical NodeState = "Critical"
	StateInactive NodeState = "Inactive"
)

const (
	MsFullPerf    MSState = "FullPerformance"
	MsPartialPerf MSState = "PArtialPerformance"
	MsIdle        MSState = "Idle"
	MsDisabled    MSState = "Disabled"
)

type Policy struct {
	Exporters []Exporter `yaml:"exporters"`
}

type Exporter struct {
	Name            string        `yaml:"name"`
	StartConditions Conditions    `yaml:"start_conditions"`
	StopConditions  Conditions    `yaml:"stop_conditions"`
	IdleConditions  IdleCondition `yaml:"idle_conditions"`
}

type Conditions struct {
	CPUUsageBelow float64 `yaml:"cpu_usage_below"`
	RAMUsageBelow float64 `yaml:"ram_usage_below"`
	CPUUsageAbove float64 `yaml:"cpu_usage_above"`
	RAMUsageAbove float64 `yaml:"ram_usage_above"`
}

type IdleCondition struct {
	IdleDurationSeconds int `yaml:"idle_duration_seconds"`
}

var exportersState = make(map[string]ExporterState)

var policy Policy

var defaultPolicy bool = true
