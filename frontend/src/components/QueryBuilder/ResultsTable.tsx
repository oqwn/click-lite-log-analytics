import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  Button,
  Chip,
  Alert,
  CircularProgress,
  Tooltip,
} from '@mui/material';
import { Download } from '@mui/icons-material';
import { QueryBuilderResponse } from '../../types/api';

interface ResultsTableProps {
  result: QueryBuilderResponse | null;
  loading: boolean;
  onExport: () => void;
}

export const ResultsTable: React.FC<ResultsTableProps> = ({
  result,
  loading,
  onExport,
}) => {
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);

  const handleChangePage = (event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const formatCellValue = (value: unknown, columnType: string): React.ReactNode => {
    if (value === null || value === undefined) {
      return <Typography variant="body2" color="text.secondary">NULL</Typography>;
    }

    if (columnType === 'datetime') {
      return new Date(value).toLocaleString();
    }

    if (columnType === 'number') {
      return typeof value === 'number' ? value.toLocaleString() : value;
    }

    if (typeof value === 'boolean') {
      return (
        <Chip 
          label={value ? 'True' : 'False'} 
          color={value ? 'success' : 'default'}
          size="small"
          variant="outlined"
        />
      );
    }

    if (typeof value === 'object') {
      return JSON.stringify(value);
    }

    const stringValue = String(value);
    if (stringValue.length > 100) {
      return (
        <Tooltip title={stringValue}>
          <Typography variant="body2" sx={{ cursor: 'help' }}>
            {stringValue.substring(0, 100)}...
          </Typography>
        </Tooltip>
      );
    }

    return stringValue;
  };

  const getColumnTypeColor = (type: string) => {
    switch (type.toLowerCase()) {
      case 'string': return 'primary';
      case 'number': return 'success';
      case 'datetime': return 'warning';
      case 'boolean': return 'error';
      default: return 'default';
    }
  };

  if (loading) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <CircularProgress sx={{ mb: 2 }} />
        <Typography>Executing query...</Typography>
      </Paper>
    );
  }

  if (!result) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography color="text.secondary">
          No results yet. Execute a query to see results here.
        </Typography>
      </Paper>
    );
  }

  if (result.error) {
    return (
      <Paper sx={{ p: 3 }}>
        <Alert severity="error">
          <Typography variant="h6">Query Error</Typography>
          <Typography sx={{ mt: 1 }}>{result.error}</Typography>
        </Alert>
      </Paper>
    );
  }

  const paginatedRows = result.rows.slice(
    page * rowsPerPage,
    page * rowsPerPage + rowsPerPage
  );

  return (
    <Paper sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Query Results</Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          {result.rows.length > 0 && (
            <Button
              variant="outlined"
              startIcon={<Download />}
              onClick={onExport}
              size="small"
            >
              Export CSV
            </Button>
          )}
        </Box>
      </Box>

      {/* Query Statistics */}
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, mb: 3 }}>
        <Chip 
          label={`${result.row_count} rows`} 
          variant="outlined" 
          size="small"
        />
        <Chip 
          label={`${result.columns.length} columns`} 
          variant="outlined" 
          size="small"
        />
        <Chip 
          label={`${result.execution_time_ms}ms`} 
          variant="outlined" 
          size="small"
          color="success"
        />
      </Box>

      {result.rows.length === 0 ? (
        <Alert severity="info">
          Query executed successfully but returned no results.
        </Alert>
      ) : (
        <>
          <TableContainer sx={{ maxHeight: 600 }}>
            <Table stickyHeader size="small">
              <TableHead>
                <TableRow>
                  {result.columns.map((column) => (
                    <TableCell key={column.name}>
                      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
                          {column.name}
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 1 }}>
                          <Chip
                            label={column.type}
                            size="small"
                            color={getColumnTypeColor(column.type) as 'primary' | 'secondary' | 'default' | 'error' | 'info' | 'success' | 'warning'}
                            variant="outlined"
                          />
                          {column.is_aggregated && (
                            <Chip
                              label="AGG"
                              size="small"
                              color="secondary"
                              variant="outlined"
                            />
                          )}
                        </Box>
                      </Box>
                    </TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedRows.map((row, index) => (
                  <TableRow key={index} hover>
                    {result.columns.map((column) => (
                      <TableCell key={column.name}>
                        {formatCellValue(row[column.name], column.type)}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>

          <TablePagination
            rowsPerPageOptions={[10, 25, 50, 100]}
            component="div"
            count={result.rows.length}
            rowsPerPage={rowsPerPage}
            page={page}
            onPageChange={handleChangePage}
            onRowsPerPageChange={handleChangeRowsPerPage}
            showFirstButton
            showLastButton
          />
        </>
      )}

      {/* SQL Query Display */}
      {result.sql && (
        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Executed SQL:
          </Typography>
          <Paper 
            variant="outlined" 
            sx={{ 
              p: 2, 
              bgcolor: 'grey.50',
              fontFamily: 'monospace',
              fontSize: '0.75rem',
              maxHeight: '200px',
              overflow: 'auto',
            }}
          >
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
              {result.sql}
            </pre>
          </Paper>
        </Box>
      )}
    </Paper>
  );
};