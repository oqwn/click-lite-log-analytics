import React, { useState } from 'react';
import {
  Button,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  CircularProgress,
} from '@mui/material';
import {
  Download,
  Description,
  TableChart,
  Code,
} from '@mui/icons-material';
import api from '../../services/api';

interface ExportButtonProps {
  query?: string;
  filters?: any[];
  startTime?: Date;
  endTime?: Date;
  fields?: string[];
}

export const ExportButton: React.FC<ExportButtonProps> = ({
  query,
  filters,
  startTime,
  endTime,
  fields,
}) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [loading, setLoading] = useState(false);

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleExport = async (format: 'csv' | 'json' | 'xlsx') => {
    try {
      setLoading(true);
      
      const response = await api.post(
        '/export/logs',
        {
          format,
          query,
          filters,
          start_time: startTime?.toISOString(),
          end_time: endTime?.toISOString(),
          fields,
          include_headers: true,
        },
        {
          responseType: 'blob',
        }
      );

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      
      // Set filename from response headers or default
      const contentDisposition = response.headers['content-disposition'];
      const filenameMatch = contentDisposition?.match(/filename="(.+)"/);
      const filename = filenameMatch ? filenameMatch[1] : `logs_export.${format}`;
      
      link.setAttribute('download', filename);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      
      handleClose();
    } catch (error) {
      console.error('Export failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const exportFormats = [
    {
      format: 'csv' as const,
      label: 'CSV',
      icon: <TableChart />,
      description: 'Comma-separated values',
    },
    {
      format: 'json' as const,
      label: 'JSON',
      icon: <Code />,
      description: 'JavaScript Object Notation',
    },
    {
      format: 'xlsx' as const,
      label: 'Excel',
      icon: <Description />,
      description: 'Microsoft Excel',
    },
  ];

  return (
    <>
      <Button
        variant="outlined"
        startIcon={loading ? <CircularProgress size={16} /> : <Download />}
        onClick={handleClick}
        disabled={loading}
      >
        Export
      </Button>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {exportFormats.map((item) => (
          <MenuItem
            key={item.format}
            onClick={() => handleExport(item.format)}
            disabled={loading}
          >
            <ListItemIcon>{item.icon}</ListItemIcon>
            <ListItemText
              primary={item.label}
              secondary={item.description}
            />
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};