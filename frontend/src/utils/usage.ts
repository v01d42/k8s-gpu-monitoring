const columnsWithPercent = [
  "gpu_utilization",
  "cpu_utilization",
  "memory_utilization",
  "cpu",
  "memory",
];
const columnsWithTmp = ["temperature"];

export const isHighUsage = (id: string, value: string | number) => {
  if (typeof value === "string") {
    return false;
  }
  if (columnsWithPercent.includes(id)) {
    // 使用率80%以上であればtrue
    return value > 80;
  } else if (columnsWithTmp.includes(id)) {
    // 温度70度以上であればtrue
    return value > 70;
  }
  return false;
};
