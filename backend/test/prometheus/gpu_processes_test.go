package prometheus_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"k8s-gpu-monitoring/internal/models"
	"k8s-gpu-monitoring/internal/prometheus"
)

// TestPrometheusClient_GetGPUProcesses tests the actual parsing of Prometheus responses into GPUProcess slice
func TestPrometheusClient_GetGPUProcesses(t *testing.T) {
	tests := []struct {
		name              string
		responseQueries   map[string]prometheus.PrometheusResponse
		expectedProcesses []models.GPUProcess
		expectError       bool
	}{
		{
			name: "successful parsing with single process",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "python",
									"user":         "keito",
									"command":      "python train.py",
								},
								Value: []interface{}{1640995200.0, "2048"},
							},
						},
					},
				},
			},
			expectedProcesses: []models.GPUProcess{
				{
					NodeName:    "node1",
					GPUIndex:    0,
					PID:         1234,
					ProcessName: "python",
					User:        "keito",
					Command:     "python train.py",
					GPUMemory:   2048,
				},
			},
			expectError: false,
		},
		{
			name: "multiple processes on different nodes and GPUs",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "python",
									"user":         "keito",
									"command":      "python train.py",
								},
								Value: []interface{}{1640995200.0, "2048"},
							},
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "1",
									"pid":          "5678",
									"process_name": "jupyter",
									"user":         "researcher",
									"command":      "jupyter notebook",
								},
								Value: []interface{}{1640995200.0, "1024"},
							},
							{
								Metric: map[string]string{
									"hostname":     "node2",
									"gpu_id":       "0",
									"pid":          "9999",
									"process_name": "torch",
									"user":         "ml_team",
									"command":      "python inference.py",
								},
								Value: []interface{}{1640995200.0, "4096"},
							},
						},
					},
				},
			},
			expectedProcesses: []models.GPUProcess{
				{
					NodeName:    "node1",
					GPUIndex:    0,
					PID:         1234,
					ProcessName: "python",
					User:        "keito",
					Command:     "python train.py",
					GPUMemory:   2048,
				},
				{
					NodeName:    "node1",
					GPUIndex:    1,
					PID:         5678,
					ProcessName: "jupyter",
					User:        "researcher",
					Command:     "jupyter notebook",
					GPUMemory:   1024,
				},
				{
					NodeName:    "node2",
					GPUIndex:    0,
					PID:         9999,
					ProcessName: "torch",
					User:        "ml_team",
					Command:     "python inference.py",
					GPUMemory:   4096,
				},
			},
			expectError: false,
		},
		{
			name: "processes with same PID on different GPUs",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "multi_gpu_app",
									"user":         "keito",
									"command":      "python multi_gpu_train.py",
								},
								Value: []interface{}{1640995200.0, "3072"},
							},
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "1",
									"pid":          "1234",
									"process_name": "multi_gpu_app",
									"user":         "keito",
									"command":      "python multi_gpu_train.py",
								},
								Value: []interface{}{1640995200.0, "3072"},
							},
						},
					},
				},
			},
			expectedProcesses: []models.GPUProcess{
				{
					NodeName:    "node1",
					GPUIndex:    0,
					PID:         1234,
					ProcessName: "multi_gpu_app",
					User:        "keito",
					Command:     "python multi_gpu_train.py",
					GPUMemory:   3072,
				},
				{
					NodeName:    "node1",
					GPUIndex:    1,
					PID:         1234,
					ProcessName: "multi_gpu_app",
					User:        "keito",
					Command:     "python multi_gpu_train.py",
					GPUMemory:   3072,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP server that returns appropriate responses
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				queryParams, _ := url.ParseQuery(r.URL.RawQuery)
				query := queryParams.Get("query")

				// Map actual queries to mock response keys
				var responseKey string
				switch query {
				case "gpu_process_gpu_memory":
					responseKey = "gpu_process_gpu_memory"
				default:
					http.Error(w, "Unknown query", http.StatusBadRequest)
					return
				}

				response, exists := tt.responseQueries[responseKey]
				if !exists {
					// Return empty result for queries not defined in test
					response = prometheus.PrometheusResponse{
						Status: "success",
						Data: struct {
							ResultType string `json:"resultType"`
							Result     []struct {
								Metric map[string]string `json:"metric"`
								Value  []interface{}     `json:"value"`
							} `json:"result"`
						}{
							ResultType: "vector",
							Result: []struct {
								Metric map[string]string `json:"metric"`
								Value  []interface{}     `json:"value"`
							}{},
						},
					}
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create Prometheus client with mock server URL
			client := prometheus.NewClient(server.URL)

			// Call GetGPUProcesses
			processes, err := client.GetGPUProcesses(context.Background())

			// Check for expected error
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// If no error expected, verify the results
			if !tt.expectError {
				if len(processes) != len(tt.expectedProcesses) {
					t.Errorf("expected %d processes, got %d", len(tt.expectedProcesses), len(processes))
					return
				}

				// Create maps for easier comparison (processes are sorted by node, gpu, pid)
				expectedMap := make(map[string]models.GPUProcess)
				for _, p := range tt.expectedProcesses {
					key := fmt.Sprintf("%s:%d:%d", p.NodeName, p.GPUIndex, p.PID)
					expectedMap[key] = p
				}

				for _, actualProcess := range processes {
					key := fmt.Sprintf("%s:%d:%d", actualProcess.NodeName, actualProcess.GPUIndex, actualProcess.PID)
					expectedProcess, exists := expectedMap[key]
					if !exists {
						t.Errorf("unexpected process for %s", key)
						continue
					}

					// Compare fields (excluding Timestamp as it's generated dynamically)
					if actualProcess.NodeName != expectedProcess.NodeName ||
						actualProcess.GPUIndex != expectedProcess.GPUIndex ||
						actualProcess.PID != expectedProcess.PID ||
						actualProcess.ProcessName != expectedProcess.ProcessName ||
						actualProcess.User != expectedProcess.User ||
						actualProcess.Command != expectedProcess.Command ||
						actualProcess.GPUMemory != expectedProcess.GPUMemory {

						t.Errorf("process mismatch for %s:\nexpected: %+v\nactual:   %+v",
							key, expectedProcess, actualProcess)
					}

					// Verify timestamp is set
					if actualProcess.Timestamp == "" {
						t.Errorf("timestamp should be set for process %s", key)
					}
				}

				// Verify sorting order
				for i := 1; i < len(processes); i++ {
					prev := processes[i-1]
					curr := processes[i]

					// Should be sorted by NodeName, then GPUIndex, then PID
					if prev.NodeName > curr.NodeName ||
						(prev.NodeName == curr.NodeName && prev.GPUIndex > curr.GPUIndex) ||
						(prev.NodeName == curr.NodeName && prev.GPUIndex == curr.GPUIndex && prev.PID > curr.PID) {
						t.Errorf("processes not sorted correctly at index %d: %+v should come before %+v", i, prev, curr)
					}
				}
			}
		})
	}
}

// TestPrometheusClient_parseGPUProcesses_EdgeCases tests the private parseGPUProcesses method indirectly
func TestPrometheusClient_parseGPUProcesses_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		responseQueries map[string]prometheus.PrometheusResponse
		expectedCount   int
		expectError     bool
	}{
		{
			name:            "empty responses",
			responseQueries: map[string]prometheus.PrometheusResponse{},
			expectedCount:   0,
			expectError:     false,
		},
		{
			name: "missing required fields - no hostname",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								// Missing hostname
								Metric: map[string]string{
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "python",
								},
								Value: []interface{}{1640995200.0, "2048"},
							},
						},
					},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "missing required fields - no gpu_id",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								// Missing gpu_id
								Metric: map[string]string{
									"hostname":     "node1",
									"pid":          "1234",
									"process_name": "python",
								},
								Value: []interface{}{1640995200.0, "2048"},
							},
						},
					},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "missing required fields - no pid",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								// Missing pid
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"process_name": "python",
								},
								Value: []interface{}{1640995200.0, "2048"},
							},
						},
					},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "invalid value format",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "python",
									"user":         "keito",
									"command":      "python train.py",
								},
								Value: []interface{}{1640995200.0, "invalid_number"},
							},
						},
					},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "missing value array",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_process_gpu_memory": {
					Status: "success",
					Data: struct {
						ResultType string `json:"resultType"`
						Result     []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						} `json:"result"`
					}{
						ResultType: "vector",
						Result: []struct {
							Metric map[string]string `json:"metric"`
							Value  []interface{}     `json:"value"`
						}{
							{
								Metric: map[string]string{
									"hostname":     "node1",
									"gpu_id":       "0",
									"pid":          "1234",
									"process_name": "python",
									"user":         "keito",
									"command":      "python train.py",
								},
								Value: []interface{}{1640995200.0}, // Missing second element
							},
						},
					},
				},
			},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				queryParams, _ := url.ParseQuery(r.URL.RawQuery)
				query := queryParams.Get("query")

				response, exists := tt.responseQueries[query]
				if !exists {
					// Return empty result
					response = prometheus.PrometheusResponse{
						Status: "success",
						Data: struct {
							ResultType string `json:"resultType"`
							Result     []struct {
								Metric map[string]string `json:"metric"`
								Value  []interface{}     `json:"value"`
							} `json:"result"`
						}{
							ResultType: "vector",
							Result: []struct {
								Metric map[string]string `json:"metric"`
								Value  []interface{}     `json:"value"`
							}{},
						},
					}
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := prometheus.NewClient(server.URL)
			processes, err := client.GetGPUProcesses(context.Background())

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(processes) != tt.expectedCount {
				t.Errorf("expected %d processes, got %d", tt.expectedCount, len(processes))
			}
		})
	}
}

// TestPrometheusClient_GetGPUProcesses_Integration tests integration with actual HTTP calls
func TestPrometheusClient_GetGPUProcesses_Integration(t *testing.T) {
	// Test connection failure
	client := prometheus.NewClient("http://invalid-url:9999")
	_, err := client.GetGPUProcesses(context.Background())
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
