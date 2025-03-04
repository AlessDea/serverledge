package ms

type NodeState string // State of the node
type MSState string   // State of the monitoryng system

var startScriptPath string = "./ms/exporters_controller/start_exporter.sh"
var stopScriptPath string = "./ms/exporters_controller/start_exporter.sh"
var pauseScriptPath string = "./ms/exporters_controller/pause_exporter.sh"
var unpauseScriptPath string = "./ms/exporters_controller/unpause_exporter.sh"

var thresholdsConfigPath string = "./ms/thresholds_config.yml"
var policyConfigPath string = "./ms/policy_config.yml"

var state MSState = MsFullPerf // state of the MS
var prevState MSState = ""
var nodeState NodeState = StateNormal // state of the node

type ExporterState string

type CommandType string

const (
	NodeExporter    string = "node-exporter"
	ProcessExporter string = "process-exporter"
	OtelCollector   string = "otel-collector"
	Prometheus      string = "prom"
	CAdvisor        string = "cadvisor"
)

const (
	running ExporterState = "running"
	paused  ExporterState = "paused"
	stopped ExporterState = "stopped"
)

const (
	run   CommandType = "run"
	stop  CommandType = "stop"
	pause CommandType = "pause"
)

func GetNodeState() NodeState {
	return nodeState
}

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
	MsPartialPerf MSState = "PartialPerformance"
	MsIdle        MSState = "Idle"
	MsDisabled    MSState = "Disabled"
)

// Thresholds defines the structure for the YAML configuration
type Thresholds struct {
	Degraded State `yaml:"Degraded"`
	Critical State `yaml:"Critical"`
	Inactive State `yaml:"Inactive"`
}

// State defines the thresholds for node and monitoring system
type State struct {
	Node             ResourceThreshold `yaml:"node"`
	MonitoringSystem ResourceThreshold `yaml:"monitoring_system"`
}

// ResourceThreshold defines CPU and RAM thresholds
type ResourceThreshold struct {
	CPU float64 `yaml:"cpu"`
	RAM float64 `yaml:"ram"`
}

// Config defines the structure for the YAML configuration for the mapping State, Actions
type Config struct {
	Idle               Actions `yaml:"Idle"`
	Disabled           Actions `yaml:"Disabled"`
	PartialPerformance Actions `yaml:"PartialPerformance"`
	FullPerformance    Actions `yaml:"FullPerformance"`
}

// Actions defines the actions (run/stop/pause) for each process in a specific performance state
type Actions struct {
	NodeExporter    CommandType `yaml:"node-exporter"`
	ProcessExporter CommandType `yaml:"process-exporter"`
	OtelCollector   CommandType `yaml:"otel-collector"`
	Prometheus      CommandType `yaml:"prometheus"`
	CAdvisor        CommandType `yaml:"cadvisor"`
}

var exportersState = make(map[string]ExporterState)

var thresholds Thresholds
var config Config

var defaultPolicy bool = true

func between(x float64, e1 float64, e2 float64) bool {
	return (x >= e1 && x < e2)
}
