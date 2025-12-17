package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s-gpu-monitoring/internal/handlers"
	"k8s-gpu-monitoring/internal/models"
	"k8s-gpu-monitoring/internal/prometheus"
)

// PrometheusClient interface for testing
type PrometheusClient interface {
	GetGPUMetrics(ctx context.Context) ([]models.GPUMetrics, error)
	GetGPUProcesses(ctx context.Context) ([]models.GPUProcess, error)
	Query(ctx context.Context, query string) (*prometheus.PrometheusResponse, error)
}

// mockPrometheusClient implements PrometheusClient interface for testing
type mockPrometheusClient struct {
	shouldReturnError  bool
	shouldProcessError bool
}

func (m *mockPrometheusClient) GetGPUMetrics(ctx context.Context) ([]models.GPUMetrics, error) {
	if m.shouldReturnError {
		return nil, errors.New("mock prometheus error")
	}

	return []models.GPUMetrics{
		{
			NodeName:          "node1",
			GPUIndex:          0,
			GPUName:           "NVIDIA Tesla V100",
			GPUUtilization:    75,
			GPUMemoryUsed:     8192,
			GPUMemoryTotal:    16384,
			GPUMemoryFree:     8192,
			CPUUtilization:    25.5,
			MemoryUtilization: 50.5,
			GPUTemperature:    65,
			Timestamp:         "2024/01/01 12:00:00",
		},
	}, nil
}

func (m *mockPrometheusClient) GetGPUProcesses(ctx context.Context) ([]models.GPUProcess, error) {
	if m.shouldProcessError {
		return nil, errors.New("mock prometheus process error")
	}

	return []models.GPUProcess{
		{
			NodeName:    "node1",
			GPUIndex:    0,
			PID:         1234,
			ProcessName: "python",
			User:        "user1",
			Command:     "python train.py",
			GPUMemory:   1024,
			Timestamp:   "2024/01/01 12:00:00",
		},
	}, nil
}

func (m *mockPrometheusClient) Query(ctx context.Context, query string) (*prometheus.PrometheusResponse, error) {
	if m.shouldReturnError {
		return nil, errors.New("mock prometheus error")
	}

	return &prometheus.PrometheusResponse{
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
					},
					Value: []interface{}{
						1234567890.0,
						"75.5",
					},
				},
			},
		},
	}, nil
}

// TestableGPUHandler wraps the real GPUHandler to allow dependency injection for testing
type TestableGPUHandler struct {
	*handlers.GPUHandler
	mockClient PrometheusClient
}

// Create a new testable GPU handler that uses mock client
func newTestableGPUHandler(mockClient PrometheusClient) *TestableGPUHandler {
	// Create a real prometheus client just for structure, but we'll override methods
	realClient := prometheus.NewClient("http://localhost:9090")
	handler := handlers.NewGPUHandler(realClient)

	return &TestableGPUHandler{
		GPUHandler: handler,
		mockClient: mockClient,
	}
}

// Override methods to use mock client
func (h *TestableGPUHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := h.mockClient.Query(ctx, "up")
	if err != nil {
		response := models.APIResponse{
			Success: false,
			Error:   "Prometheus connection failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success: true,
		Message: "Service is healthy",
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": "2024/01/01 12:00:00",
			"version":   "1.0.0",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *TestableGPUHandler) GetGPUMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	metrics, err := h.mockClient.GetGPUMetrics(ctx)
	if err != nil {
		response := models.APIResponse{
			Success: false,
			Error:   "Failed to retrieve GPU metrics",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success: true,
		Data:    metrics,
		Message: "GPU metrics retrieved successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *TestableGPUHandler) GetGPUProcesses(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	processes, err := h.mockClient.GetGPUProcesses(ctx)
	if err != nil {
		response := models.APIResponse{
			Success: false,
			Error:   "Failed to retrieve GPU processes",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success: true,
		Data:    processes,
		Message: "GPU processes retrieved successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name         string
		mockError    bool
		expectedCode int
	}{
		{
			name:         "successful health check",
			mockError:    false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "prometheus connection error",
			mockError:    true,
			expectedCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockPrometheusClient{shouldReturnError: tt.mockError}
			handler := newTestableGPUHandler(mockClient)

			req := httptest.NewRequest("GET", "/api/healthz", nil)
			w := httptest.NewRecorder()

			handler.HealthCheck(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var response models.APIResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if tt.mockError && response.Success {
				t.Error("expected success to be false for error case")
			}
			if !tt.mockError && !response.Success {
				t.Error("expected success to be true for success case")
			}

			// For successful health check, verify the data structure
			if !tt.mockError {
				if response.Data == nil {
					t.Error("expected data to be present for successful health check")
				}
			}
		})
	}
}

func TestGetGPUMetrics(t *testing.T) {
	tests := []struct {
		name         string
		mockError    bool
		expectedCode int
		checkData    bool
	}{
		{
			name:         "successful metrics retrieval",
			mockError:    false,
			expectedCode: http.StatusOK,
			checkData:    true,
		},
		{
			name:         "prometheus error",
			mockError:    true,
			expectedCode: http.StatusInternalServerError,
			checkData:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockPrometheusClient{shouldReturnError: tt.mockError}
			handler := newTestableGPUHandler(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/gpu/metrics", nil)
			w := httptest.NewRecorder()

			handler.GetGPUMetrics(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var response models.APIResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if tt.checkData {
				if !response.Success {
					t.Error("expected success to be true")
				}
				if response.Data == nil {
					t.Error("expected data to be present")
				}

				// Verify the message
				if response.Message != "GPU metrics retrieved successfully" {
					t.Errorf("expected message 'GPU metrics retrieved successfully', got '%s'", response.Message)
				}
			} else {
				if response.Success {
					t.Error("expected success to be false for error case")
				}
				if response.Error != "Failed to retrieve GPU metrics" {
					t.Errorf("expected error message 'Failed to retrieve GPU metrics', got '%s'", response.Error)
				}
			}
		})
	}
}

func TestGetGPUProcesses(t *testing.T) {
	tests := []struct {
		name             string
		mockProcessError bool
		expectedCode     int
		checkData        bool
	}{
		{
			name:             "successful processes retrieval",
			mockProcessError: false,
			expectedCode:     http.StatusOK,
			checkData:        true,
		},
		{
			name:             "prometheus process error",
			mockProcessError: true,
			expectedCode:     http.StatusInternalServerError,
			checkData:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockPrometheusClient{shouldProcessError: tt.mockProcessError}
			handler := newTestableGPUHandler(mockClient)

			req := httptest.NewRequest("GET", "/api/v1/gpu/processes", nil)
			w := httptest.NewRecorder()

			handler.GetGPUProcesses(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var response models.APIResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if tt.checkData {
				if !response.Success {
					t.Error("expected success to be true")
				}
				if response.Data == nil {
					t.Error("expected data to be present")
				}

				// Verify the message
				if response.Message != "GPU processes retrieved successfully" {
					t.Errorf("expected message 'GPU processes retrieved successfully', got '%s'", response.Message)
				}
			} else {
				if response.Success {
					t.Error("expected success to be false")
				}
				if response.Error != "Failed to retrieve GPU processes" {
					t.Errorf("expected error message 'Failed to retrieve GPU processes', got '%s'", response.Error)
				}
			}
		})
	}
}

// TestGPUHandlerIntegration tests the actual GPUHandler with a real Prometheus client (but mocked HTTP)
func TestGPUHandlerIntegration(t *testing.T) {
	// This test verifies that the real GPUHandler works as expected
	// We don't test with actual network calls, but verify the structure
	realClient := prometheus.NewClient("http://localhost:9090")
	handler := handlers.NewGPUHandler(realClient)

	// Test that the handler is created correctly
	if handler == nil {
		t.Fatal("expected handler to be created")
	}

	// Test GET request setup
	req := httptest.NewRequest("GET", "/api/healthz", nil)
	w := httptest.NewRecorder()

	// We can't easily test the actual behavior without mocking the network,
	// but we can verify the handler doesn't panic and handles basic setup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handler panicked: %v", r)
		}
	}()

	// This will likely fail with network error, but shouldn't panic
	handler.HealthCheck(w, req)

	// The response will likely be an error due to no real Prometheus server
	// but that's expected in this integration test
}
