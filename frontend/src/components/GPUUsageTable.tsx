import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TableSortLabel from "@mui/material/TableSortLabel";
import { useContext, useEffect, useMemo, useState } from "react";

import { Box, CircularProgress, Typography } from "@mui/material";
import { styled } from "@mui/material/styles";
import type { ApiResponse, GPUMetrics } from "../types/api";
import { mockGpuMetrics } from "../types/api.mock";
import { getConfig } from "../utils/config";
import { searchContext } from "../utils/contexts";
import { convertGPUMetrics } from "../utils/convert";
import { getComparator } from "../utils/sort";
import { isHighUsage } from "../utils/usage";

const config = getConfig();
const API_BASE_URL = config.API_BASE_URL;
/**
 * Fetch GPU metrics from the API. If the request fails, return mock data.
 */
const fetchGpuMetricsWithFallback = async (): Promise<
  ApiResponse<GPUMetrics[]>
> => {
  try {
    console.log(
      `Fetching GPU metrics from: ${API_BASE_URL}/api/v1/gpu/metrics`
    );
    const res = await fetch(`${API_BASE_URL}/api/v1/gpu/metrics`);
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    const data = (await res.json()) as ApiResponse<GPUMetrics[]>;
    return data;
  } catch (e) {
    console.log("Failed to fetch GPU metrics:", e);
    console.log("use mock data due to fetch error");
    return mockGpuMetrics;
  }
};

type Order = "asc" | "desc";
type GpuRowKey = keyof GPUMetrics;

const columns: {
  id: GpuRowKey;
  label: string;
}[] = [
  { id: "node_name", label: "node_name" },
  { id: "timestamp", label: "timestamp" },
  { id: "gpu_index", label: "index" },
  { id: "gpu_name", label: "name" },
  { id: "utilization", label: "utilization (%)" },
  { id: "memory_used", label: "memory_used (MiB)" },
  { id: "memory_total", label: "memory_total (MiB)" },
  { id: "temperature", label: "temperature (°C)" },
];

const ContentWrapper = styled("div")(({ theme }) => ({
  marginTop: "65px",
  [theme.breakpoints.up("sm")]: {
    marginTop: "70px",
  },
}));

const GPUUsageTable = () => {
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const { searchText } = useContext(searchContext);
  const [order, setOrder] = useState<Order>("asc");
  const [orderBy, setOrderBy] = useState<GpuRowKey>("node_name");

  const handleRequestSort = (property: GpuRowKey) => {
    const isAsc = orderBy === property && order === "asc";
    setOrder(isAsc ? "desc" : "asc");
    setOrderBy(property);
  };

  const [rows, setRows] = useState<GPUMetrics[]>([]);

  const fetchAndUpdate = () => {
    fetchGpuMetricsWithFallback().then((res) => {
      let data = [];
      if (res.data) {
        data = JSON.parse(JSON.stringify(res.data));
        convertGPUMetrics(data);
      }
      setRows(data);
      setIsLoading(false);
    });
  };

  useEffect(() => {
    fetchAndUpdate();
  }, []);

  // ソート
  const sortedRows = useMemo(() => {
    const arr = rows.filter(({ node_name }) => {
      return node_name.indexOf(searchText) > -1;
    });
    if (orderBy === "node_name") {
      // node_nameでソート時は、node_name→gpu_indexの複合ソート
      return arr.sort((a, b) => {
        const nodeComp =
          order === "asc"
            ? a.node_name.localeCompare(b.node_name)
            : b.node_name.localeCompare(a.node_name);
        return nodeComp || a.gpu_index - b.gpu_index;
      });
    }
    // 他のカラムは通常のソート
    return arr.sort(getComparator<GPUMetrics>(order, orderBy));
  }, [rows, order, orderBy, searchText]);

  return (
    <ContentWrapper>
      <Typography variant="h4" sx={{ paddingTop: 2, marginBottom: 2 }}>
        GPU Usage
      </Typography>
      {isLoading ? (
        <Box
          sx={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            minHeight: 200,
          }}
        >
          <CircularProgress size="40px" />
        </Box>
      ) : (
        <TableContainer component={Paper}>
          <Table
            sx={{
              width: "100%",
              borderCollapse: "separate",
              borderSpacing: 0,
            }}
            aria-label="gpu table"
          >
            <TableHead>
              <TableRow>
                {columns.map((col, colIdx) => (
                  <TableCell
                    key={col.id}
                    align="left"
                    sortDirection={orderBy === col.id ? order : false}
                    sx={{
                      height: 12,
                      padding: "8px 4px",
                      borderRight:
                        colIdx !== columns.length - 1
                          ? "1px solid #e0e0e0"
                          : undefined,
                      borderLeft:
                        colIdx === 0 ? "1px solid #e0e0e0" : undefined,
                    }}
                  >
                    <TableSortLabel
                      active={orderBy === col.id}
                      direction={orderBy === col.id ? order : "asc"}
                      onClick={() => handleRequestSort(col.id)}
                    >
                      {col.label}
                    </TableSortLabel>
                  </TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {sortedRows.map((row, idx) => (
                <TableRow key={row.node_name + "-" + row.gpu_index + "-" + idx}>
                  {columns.map((col, colIdx) => (
                    <TableCell
                      key={col.id}
                      align="left"
                      sx={{
                        height: 4,
                        padding: "8px 4px",
                        borderRight:
                          colIdx !== columns.length - 1
                            ? "1px solid #e0e0e0"
                            : undefined,
                        borderLeft:
                          colIdx === 0 ? "1px solid #e0e0e0" : undefined,
                        backgroundColor: isHighUsage(col.id, row[col.id])
                          ? "#ef9a9a"
                          : "white",
                      }}
                    >
                      {row[col.id]}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </ContentWrapper>
  );
};
export default GPUUsageTable;
