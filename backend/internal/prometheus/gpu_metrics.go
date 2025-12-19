package prometheus

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"k8s-gpu-monitoring/internal/models"
	"k8s-gpu-monitoring/internal/timeutil"
)

// GetGPUMetrics retrieves GPU metrics from Prometheus with concurrent queries.
func (c *Client) GetGPUMetrics(ctx context.Context) ([]models.GPUMetrics, error) {
	// Execute multiple queries concurrently
	queries := map[string]string{
		"gpu_mem_free":       `gpu_metrics_free_memory`,
		"gpu_mem_used":       `gpu_metrics_used_memory`,
		"gpu_mem_total":      `gpu_metrics_total_memory`,
		"gpu_utilization":    `gpu_metrics_utilization_percent`,
		"gpu_temperature":    `gpu_metrics_temperature`,
		"cpu_utilization":    `gpu_metrics_cpu_utilization`,
		"memory_utilization": `gpu_metrics_memory_utilization`,
	}

	results := make(map[string]*PrometheusResponse)
	errors := make(chan error, len(queries))
	var mu sync.Mutex

	// Execute queries concurrently
	for name, query := range queries {
		go func(name, query string) {
			resp, err := c.Query(ctx, query)
			if err != nil {
				errors <- fmt.Errorf("query %s failed: %w", name, err)
				return
			}
			mu.Lock()
			results[name] = resp
			mu.Unlock()
			errors <- nil
		}(name, query)
	}

	// Wait for all queries to complete
	for i := 0; i < len(queries); i++ {
		if err := <-errors; err != nil {
			return nil, err
		}
	}

	return c.parseGPUMetrics(results)
}

// parseGPUMetrics parses Prometheus response into GPUMetrics.
func (c *Client) parseGPUMetrics(results map[string]*PrometheusResponse) ([]models.GPUMetrics, error) {
	// Group metrics by node and GPU index
	metricsMap := make(map[string]models.GPUMetrics) // key: "node_name:gpu_index"
	// Store node-level CPU/Memory utilization
	nodeUtilization := make(map[string]struct {
		cpuUtilization    float64
		memoryUtilization float64
	})

	for metricType, response := range results {
		for _, result := range response.Data.Result {
			nodeName := result.Metric["hostname"]
			gpuIndex := result.Metric["gpu_id"]
			gpuName := result.Metric["gpu_name"]

			if nodeName == "" {
				continue
			}

			// Parse and extract value
			if len(result.Value) < 2 {
				continue
			}

			valueStr, ok := result.Value[1].(string)
			if !ok {
				continue
			}

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			// Handle node-level metrics (cpu_utilization, memory_utilization)
			if metricType == "cpu_utilization" || metricType == "memory_utilization" {
				util := nodeUtilization[nodeName]
				if metricType == "cpu_utilization" {
					util.cpuUtilization = value
				} else {
					util.memoryUtilization = value
				}
				nodeUtilization[nodeName] = util
				continue
			}

			if gpuIndex == "" {
				continue
			}

			key := fmt.Sprintf("%s:%s", nodeName, gpuIndex)

			metricsEntry, exists := metricsMap[key]
			if !exists {
				idx, _ := strconv.Atoi(gpuIndex)
				metricsEntry = models.GPUMetrics{
					NodeName:  nodeName,
					GPUIndex:  idx,
					GPUName:   gpuName,
					Timestamp: timeutil.NowJST(),
				}
			}

			// Set value based on metric type
			switch metricType {
			case "gpu_mem_free":
				metricsEntry.GPUMemoryFree = int(value)
			case "gpu_mem_used":
				metricsEntry.GPUMemoryUsed = int(value)
			case "gpu_mem_total":
				metricsEntry.GPUMemoryTotal = int(value)
			case "gpu_utilization":
				metricsEntry.GPUUtilization = int(value)
			case "gpu_temperature":
				metricsEntry.GPUTemperature = int(value)
			}

			metricsMap[key] = metricsEntry
		}
	}

	// Apply node-level utilization to all GPUs on that node
	for key, metricsEntry := range metricsMap {
		nodeName := metricsEntry.NodeName
		if util, exists := nodeUtilization[nodeName]; exists {
			metricsEntry.CPUUtilization = int(util.cpuUtilization)
			metricsEntry.MemoryUtilization = int(util.memoryUtilization)
			metricsMap[key] = metricsEntry
		}
	}

	// Convert to slice
	var gpuMetrics []models.GPUMetrics
	for _, metrics := range metricsMap {
		gpuMetrics = append(gpuMetrics, metrics)
	}

	return gpuMetrics, nil
}
