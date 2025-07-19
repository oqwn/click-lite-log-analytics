import React from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Box,
  Chip,
} from '@mui/material';

interface TableWidgetProps {
  data: unknown;
  config: Record<string, unknown>;
  height: number;
}

export const TableWidget: React.FC<TableWidgetProps> = ({ data, height }) => {
  if (!data || !Array.isArray(data)) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', height }}>
        <Typography color="text.secondary">No data available</Typography>
      </Box>
    );
  }

  if (data.length === 0) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', height }}>
        <Typography color="text.secondary">No results found</Typography>
      </Box>
    );
  }

  const columns = Object.keys(data[0]);
  const maxRows = Math.floor((height - 100) / 40); // Estimate row height
  const displayData = data.slice(0, maxRows);

  const formatCellValue = (value: unknown): React.ReactNode => {
    if (value === null || value === undefined) {
      return <Typography variant="body2" color="text.secondary">NULL</Typography>;
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

    if (typeof value === 'number') {
      return value.toLocaleString();
    }

    const stringValue = String(value);
    if (stringValue.length > 50) {
      return (
        <Typography variant="body2" title={stringValue}>
          {stringValue.substring(0, 50)}...
        </Typography>
      );
    }

    return stringValue;
  };

  return (
    <TableContainer sx={{ height, overflow: 'auto' }}>
      <Table stickyHeader size="small">
        <TableHead>
          <TableRow>
            {columns.map((column) => (
              <TableCell key={column}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
                  {column}
                </Typography>
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {displayData.map((row, index) => (
            <TableRow key={index} hover>
              {columns.map((column) => (
                <TableCell key={column}>
                  {formatCellValue(row[column])}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
      
      {data.length > maxRows && (
        <Box sx={{ p: 1, textAlign: 'center', bgcolor: 'grey.50' }}>
          <Typography variant="caption" color="text.secondary">
            Showing {maxRows} of {data.length} rows
          </Typography>
        </Box>
      )}
    </TableContainer>
  );
};