import type { GPUMetrics, GPUProcess } from "../types/api";

const convertBytestoMiB = (bytes: number) => Math.round(bytes / 2 ** 20);

export const convertGPUMetrics = (ms: GPUMetrics[]) => {
  const convertKeys = [
    "gpu_memory_used",
    "gpu_memory_total",
    "memory_free",
  ] as const;
  for (const m of ms) {
    for (const key of convertKeys) {
      m[key] = convertBytestoMiB(m[key]);
    }
  }
};

export const convertGPUProcesses = (ps: GPUProcess[]) => {
  const convertKeys = ["gpu_memory"] as const;
  for (const p of ps) {
    for (const key of convertKeys) {
      p[key] = convertBytestoMiB(p[key]);
    }
  }
};
