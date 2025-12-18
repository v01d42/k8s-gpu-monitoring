// API response types for backend endpoints

export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

export interface GPUMetrics {
  node_name: string;
  gpu_index: number;
  gpu_name: string;
  gpu_utilization: number;
  gpu_memory_used: number;
  gpu_memory_total: number;
  memory_free: number;
  temperature: number;
  cpu_utilization: number;
  memory_utilization: number;
  timestamp: string; // ISO8601
}

export interface GPUProcess {
  node_name: string;
  gpu_index: number;
  pid: number;
  process_name: string;
  user: string;
  command: string;
  gpu_memory: number;
  timestamp: string;
}
