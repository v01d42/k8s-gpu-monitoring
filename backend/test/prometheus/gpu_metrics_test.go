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

// TestPrometheusClient_GetGPUMetrics tests the actual parsing of Prometheus responses into GPUMetrics
func TestPrometheusClient_GetGPUMetrics(t *testing.T) {
	tests := []struct {
		name            string
		responseQueries map[string]prometheus.PrometheusResponse
		expectedMetrics []models.GPUMetrics
		expectError     bool
	}{
		{
			name: "successful parsing with complete metrics",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_metrics_free_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "8192"},
							},
						},
					},
				},
				"gpu_metrics_used_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "8192"},
							},
						},
					},
				},
				"gpu_metrics_total_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "16384"},
							},
						},
					},
				},
				"gpu_metrics_utilization_percent": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "75"},
							},
						},
					},
				},
				"gpu_metrics_temperature_celsius": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "65"},
							},
						},
					},
				},
				"gpu_metrics_cpu_utilization": {
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
									"hostname": "node1",
								},
								Value: []interface{}{1640995200.0, "25.5"},
							},
						},
					},
				},
				"gpu_metrics_memory_utilization": {
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
									"hostname": "node1",
								},
								Value: []interface{}{1640995200.0, "50.0"},
							},
						},
					},
				},
			},
			expectedMetrics: []models.GPUMetrics{
				{
					NodeName:          "node1",
					GPUIndex:          0,
					GPUName:           "NVIDIA Tesla V100",
					GPUMemoryFree:     8192,
					GPUMemoryUsed:     8192,
					GPUMemoryTotal:    16384,
					GPUUtilization:    75,
					GPUTemperature:    65,
					CPUUtilization:    25,
					MemoryUtilization: 50,
				},
			},
			expectError: false,
		},
		{
			name: "multiple GPUs on different nodes",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_metrics_free_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "4096"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
									"gpu_id":   "1",
									"gpu_name": "NVIDIA Tesla A100",
								},
								Value: []interface{}{1640995200.0, "8192"},
							},
						},
					},
				},
				"gpu_metrics_used_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "12288"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
									"gpu_id":   "1",
									"gpu_name": "NVIDIA Tesla A100",
								},
								Value: []interface{}{1640995200.0, "32768"},
							},
						},
					},
				},
				"gpu_metrics_total_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "16384"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
									"gpu_id":   "1",
									"gpu_name": "NVIDIA Tesla A100",
								},
								Value: []interface{}{1640995200.0, "40960"},
							},
						},
					},
				},
				"gpu_metrics_utilization_percent": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "85"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
									"gpu_id":   "1",
									"gpu_name": "NVIDIA Tesla A100",
								},
								Value: []interface{}{1640995200.0, "95"},
							},
						},
					},
				},
				"gpu_metrics_temperature_celsius": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "70"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
									"gpu_id":   "1",
									"gpu_name": "NVIDIA Tesla A100",
								},
								Value: []interface{}{1640995200.0, "80"},
							},
						},
					},
				},
				"gpu_metrics_cpu_utilization": {
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
									"hostname": "node1",
								},
								Value: []interface{}{1640995200.0, "30.5"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
								},
								Value: []interface{}{1640995200.0, "45.8"},
							},
						},
					},
				},
				"gpu_metrics_memory_utilization": {
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
									"hostname": "node1",
								},
								Value: []interface{}{1640995200.0, "75.0"},
							},
							{
								Metric: map[string]string{
									"hostname": "node2",
								},
								Value: []interface{}{1640995200.0, "80.0"},
							},
						},
					},
				},
			},
			expectedMetrics: []models.GPUMetrics{
				{
					NodeName:          "node1",
					GPUIndex:          0,
					GPUName:           "NVIDIA Tesla V100",
					GPUMemoryFree:     4096,
					GPUMemoryUsed:     12288,
					GPUMemoryTotal:    16384,
					GPUUtilization:    85,
					GPUTemperature:    70,
					CPUUtilization:    30,
					MemoryUtilization: 75,
				},
				{
					NodeName:          "node2",
					GPUIndex:          1,
					GPUName:           "NVIDIA Tesla A100",
					GPUMemoryFree:     8192,
					GPUMemoryUsed:     32768,
					GPUMemoryTotal:    40960,
					GPUUtilization:    95,
					GPUTemperature:    80,
					CPUUtilization:    45,
					MemoryUtilization: 80,
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
				case "gpu_metrics_free_memory":
					responseKey = "gpu_metrics_free_memory"
				case "gpu_metrics_used_memory":
					responseKey = "gpu_metrics_used_memory"
				case "gpu_metrics_total_memory":
					responseKey = "gpu_metrics_total_memory"
				case "gpu_metrics_utilization_percent":
					responseKey = "gpu_metrics_utilization_percent"
				case "gpu_metrics_temperature_celsius":
					responseKey = "gpu_metrics_temperature_celsius"
				case "gpu_metrics_cpu_utilization":
					responseKey = "gpu_metrics_cpu_utilization"
				case "gpu_metrics_memory_utilization":
					responseKey = "gpu_metrics_memory_utilization"
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

			// Call GetGPUMetrics
			metrics, err := client.GetGPUMetrics(context.Background())

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
				if len(metrics) != len(tt.expectedMetrics) {
					t.Errorf("expected %d metrics, got %d", len(tt.expectedMetrics), len(metrics))
					return
				}

				// Sort both slices to ensure consistent comparison
				// (since the order might vary due to concurrent processing)
				expectedMap := make(map[string]models.GPUMetrics)
				for _, m := range tt.expectedMetrics {
					key := fmt.Sprintf("%s:%d", m.NodeName, m.GPUIndex)
					expectedMap[key] = m
				}

				for _, actualMetric := range metrics {
					key := fmt.Sprintf("%s:%d", actualMetric.NodeName, actualMetric.GPUIndex)
					expectedMetric, exists := expectedMap[key]
					if !exists {
						t.Errorf("unexpected metric for %s", key)
						continue
					}

					// Compare fields (excluding Timestamp as it's generated dynamically)
					if actualMetric.NodeName != expectedMetric.NodeName ||
						actualMetric.GPUIndex != expectedMetric.GPUIndex ||
						actualMetric.GPUName != expectedMetric.GPUName ||
						actualMetric.GPUMemoryFree != expectedMetric.GPUMemoryFree ||
						actualMetric.GPUMemoryUsed != expectedMetric.GPUMemoryUsed ||
						actualMetric.GPUMemoryTotal != expectedMetric.GPUMemoryTotal ||
						actualMetric.GPUUtilization != expectedMetric.GPUUtilization ||
						actualMetric.GPUTemperature != expectedMetric.GPUTemperature ||
						actualMetric.CPUUtilization != expectedMetric.CPUUtilization ||
						actualMetric.MemoryUtilization != expectedMetric.MemoryUtilization {

						t.Errorf("metrics mismatch for %s:\nexpected: %+v\nactual:   %+v",
							key, expectedMetric, actualMetric)
					}

					// Verify timestamp is set
					if actualMetric.Timestamp == "" {
						t.Errorf("timestamp should be set for metric %s", key)
					}
				}
			}
		})
	}
}

// TestPrometheusClient_parseGPUMetrics tests the private parseGPUMetrics method indirectly
func TestPrometheusClient_parseGPUMetrics_EdgeCases(t *testing.T) {
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
			name: "missing required fields",
			responseQueries: map[string]prometheus.PrometheusResponse{
				"gpu_metrics_free_memory": {
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
								// Missing hostname or gpu_id
								Metric: map[string]string{
									"gpu_name": "NVIDIA Tesla V100",
								},
								Value: []interface{}{1640995200.0, "8192"},
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
				"gpu_metrics_free_memory": {
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
									"hostname": "node1",
									"gpu_id":   "0",
									"gpu_name": "NVIDIA Tesla V100",
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
			metrics, err := client.GetGPUMetrics(context.Background())

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(metrics) != tt.expectedCount {
				t.Errorf("expected %d metrics, got %d", tt.expectedCount, len(metrics))
			}
		})
	}
}

// TestPrometheusClient_GetGPUMetrics_Integration tests integration with actual HTTP calls
func TestPrometheusClient_GetGPUMetrics_Integration(t *testing.T) {
	// Test connection failure
	client := prometheus.NewClient("http://invalid-url:9999")
	_, err := client.GetGPUMetrics(context.Background())
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
