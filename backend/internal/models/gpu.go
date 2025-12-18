package models

// GPUMetrics represents GPU metrics data structure
type GPUMetrics struct {
	NodeName          string `json:"node_name"`
	GPUIndex          int    `json:"gpu_index"`
	GPUName           string `json:"gpu_name"`
	GPUMemoryUsed     int    `json:"gpu_memory_used"`
	GPUMemoryTotal    int    `json:"gpu_memory_total"`
	GPUMemoryFree     int    `json:"memory_free"`
	GPUUtilization    int    `json:"gpu_utilization"`
	GPUTemperature    int    `json:"temperature"`
	CPUUtilization    int    `json:"cpu_utilization"`
	MemoryUtilization int    `json:"memory_utilization"`
	Timestamp         string `json:"timestamp"`
}

// GPUProcess represents running GPU-related processes and their usage metrics.
type GPUProcess struct {
	NodeName    string `json:"node_name"`
	GPUIndex    int    `json:"gpu_index"`
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	User        string `json:"user"`
	Command     string `json:"command"`
	GPUMemory   int    `json:"gpu_memory"`
	Timestamp   string `json:"timestamp"`
}

// APIResponse represents standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// MetricsQuery represents Prometheus query parameters
type MetricsQuery struct {
	Query     string `json:"query"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Step      string `json:"step,omitempty"`
}
