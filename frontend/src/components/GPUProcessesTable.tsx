import { Box, CircularProgress, Typography } from "@mui/material";
import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TableSortLabel from "@mui/material/TableSortLabel";
import { useContext, useEffect, useMemo, useState } from "react";
import { searchContext } from "../utils/contexts";
// ...existing code...
import type { ApiResponse, GPUProcess } from "../types/api";
import { mockGPUProcesses } from "../types/api.mock";
import { getConfig } from "../utils/config";
import { convertGPUProcesses } from "../utils/convert";
import { getComparator } from "../utils/sort";
import { isHighUsage } from "../utils/usage";

const config = getConfig();
const API_BASE_URL = config.API_BASE_URL;

const fetchGpuProcessesWithFallback = async (): Promise<
  ApiResponse<GPUProcess[]>
> => {
  try {
    console.log(
      `Fetching GPU processes from: ${API_BASE_URL}/api/v1/gpu/processes`
    );
    const res = await fetch(`${API_BASE_URL}/api/v1/gpu/processes`);
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    const data = (await res.json()) as ApiResponse<GPUProcess[]>;
    return data;
  } catch (e) {
    console.log("Failed to fetch GPU processes:", e);
    console.log("using mock data due to fetch error");
    return mockGPUProcesses;
  }
};

type Order = "asc" | "desc";
type GpuProcessRowKey = keyof GPUProcess;

const columns: {
  id: GpuProcessRowKey;
  label: string;
}[] = [
  { id: "node_name", label: "node_name" },
  { id: "timestamp", label: "timestamp" },
  { id: "gpu_index", label: "gpu_index" },
  { id: "pid", label: "pid" },
  { id: "process_name", label: "process_name" },
  { id: "user", label: "user" },
  { id: "command", label: "command" },
  { id: "gpu_memory", label: "gpu_memory (MiB)" },
];

const GPUProcessesTable = () => {
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const { searchText } = useContext(searchContext);
  const [order, setOrder] = useState<Order>("asc");
  const [orderBy, setOrderBy] = useState<GpuProcessRowKey>("node_name");
  const [rows, setRows] = useState<GPUProcess[]>([]);

  const handleRequestSort = (property: GpuProcessRowKey) => {
    const isAsc = orderBy === property && order === "asc";
    setOrder(isAsc ? "desc" : "asc");
    setOrderBy(property);
  };

  const fetchAndUpdate = () => {
    fetchGpuProcessesWithFallback().then((res) => {
      let data = [];
      if (res.data) {
        data = JSON.parse(JSON.stringify(res.data));
        convertGPUProcesses(data);
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
    return arr.sort(getComparator<GPUProcess>(order, orderBy));
  }, [rows, order, orderBy, searchText]);

  return (
    <Box sx={{ marginTop: 4 }}>
      <Typography variant="h4" sx={{ paddingTop: 2, marginBottom: 2 }}>
        GPU Processes
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
            aria-label="gpu processes table"
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
                <TableRow
                  key={
                    row.node_name +
                    "-" +
                    row.gpu_index +
                    "-" +
                    row.pid +
                    "-" +
                    idx
                  }
                >
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
    </Box>
  );
};
export default GPUProcessesTable;
