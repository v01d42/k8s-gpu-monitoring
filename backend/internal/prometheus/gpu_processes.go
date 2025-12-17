package prometheus

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"k8s-gpu-monitoring/internal/models"
	"k8s-gpu-monitoring/internal/timeutil"
)

// GetGPUProcesses retrieves running GPU processes from Prometheus.
func (c *Client) GetGPUProcesses(ctx context.Context) ([]models.GPUProcess, error) {
	queries := map[string]string{
		"gpu_memory": `gpu_process_gpu_memory`,
	}

	results := make(map[string]*PrometheusResponse)
	errors := make(chan error, len(queries))
	var mu sync.Mutex

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

	for i := 0; i < len(queries); i++ {
		if err := <-errors; err != nil {
			return nil, err
		}
	}

	return c.parseGPUProcesses(results)
}

// parseGPUProcesses parses Prometheus response into GPUProcess slice.
func (c *Client) parseGPUProcesses(results map[string]*PrometheusResponse) ([]models.GPUProcess, error) {
	processMap := make(map[string]models.GPUProcess)

	for metricType, response := range results {
		if response == nil {
			continue
		}

		for _, result := range response.Data.Result {
			nodeName := result.Metric["hostname"]
			gpuIndex := result.Metric["gpu_id"]
			pidStr := result.Metric["pid"]

			if nodeName == "" || gpuIndex == "" || pidStr == "" {
				continue
			}

			key := fmt.Sprintf("%s:%s:%s", nodeName, gpuIndex, pidStr)

			proc, exists := processMap[key]
			if !exists {
				idx, _ := strconv.Atoi(gpuIndex)
				pid, _ := strconv.Atoi(pidStr)

				proc = models.GPUProcess{
					NodeName:    nodeName,
					GPUIndex:    idx,
					PID:         pid,
					ProcessName: result.Metric["process_name"],
					User:        result.Metric["user"],
					Command:     result.Metric["command"],
					Timestamp:   timeutil.NowJST(),
				}
			}

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

			switch metricType {
			case "gpu_memory":
				proc.GPUMemory = int(value)
			}

			processMap[key] = proc
		}
	}

	if len(processMap) == 0 {
		return []models.GPUProcess{}, nil
	}

	processes := make([]models.GPUProcess, 0, len(processMap))
	for _, proc := range processMap {
		processes = append(processes, proc)
	}

	sort.Slice(processes, func(i, j int) bool {
		if processes[i].NodeName != processes[j].NodeName {
			return processes[i].NodeName < processes[j].NodeName
		}
		if processes[i].GPUIndex != processes[j].GPUIndex {
			return processes[i].GPUIndex < processes[j].GPUIndex
		}
		return processes[i].PID < processes[j].PID
	})

	return processes, nil
}
