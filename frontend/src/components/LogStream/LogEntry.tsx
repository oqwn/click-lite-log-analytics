import React, { useState } from 'react';
import {
  Box,
  Chip,
  Typography,
  Collapse,
  IconButton,
  Paper,
} from '@mui/material';
import {
  ExpandMore,
  ExpandLess,
  ContentCopy,
} from '@mui/icons-material';
import { format } from 'date-fns';
import { Log, LOG_LEVEL_COLORS } from '../../types/log';

interface LogEntryProps {
  log: Log;
}

const LogEntry: React.FC<LogEntryProps> = ({ log }) => {
  const [expanded, setExpanded] = useState(false);

  const handleCopy = () => {
    const logText = JSON.stringify(log, null, 2);
    navigator.clipboard.writeText(logText);
  };

  const levelColor = LOG_LEVEL_COLORS[log.level as keyof typeof LOG_LEVEL_COLORS] || '#666';

  return (
    <Paper
      elevation={1}
      sx={{
        mb: 1,
        p: 1.5,
        borderLeft: 4,
        borderColor: levelColor,
        '&:hover': {
          bgcolor: 'action.hover',
        },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        {/* Timestamp */}
        <Typography
          variant="caption"
          sx={{
            color: 'text.secondary',
            fontFamily: 'monospace',
            whiteSpace: 'nowrap',
          }}
        >
          {format(new Date(log.timestamp), 'HH:mm:ss.SSS')}
        </Typography>

        {/* Level */}
        <Chip
          label={log.level.toUpperCase()}
          size="small"
          sx={{
            bgcolor: levelColor,
            color: 'white',
            fontSize: 10,
            height: 20,
            minWidth: 50,
          }}
        />

        {/* Service */}
        <Chip
          label={log.service}
          size="small"
          variant="outlined"
          sx={{
            fontSize: 10,
            height: 20,
          }}
        />

        {/* Message */}
        <Typography
          variant="body2"
          sx={{
            flexGrow: 1,
            fontFamily: 'monospace',
            wordBreak: 'break-word',
          }}
        >
          {log.message}
        </Typography>

        {/* Actions */}
        <Box sx={{ display: 'flex', gap: 0.5 }}>
          <IconButton size="small" onClick={handleCopy}>
            <ContentCopy fontSize="small" />
          </IconButton>
          <IconButton
            size="small"
            onClick={() => setExpanded(!expanded)}
            disabled={!log.attributes || Object.keys(log.attributes).length === 0}
          >
            {expanded ? <ExpandLess fontSize="small" /> : <ExpandMore fontSize="small" />}
          </IconButton>
        </Box>
      </Box>

      {/* Trace ID */}
      {log.trace_id && (
        <Box sx={{ mt: 0.5, display: 'flex', gap: 1 }}>
          <Typography variant="caption" color="text.secondary">
            Trace ID:
          </Typography>
          <Typography
            variant="caption"
            sx={{ fontFamily: 'monospace', color: 'primary.main', cursor: 'pointer' }}
            onClick={() => navigator.clipboard.writeText(log.trace_id!)}
          >
            {log.trace_id}
          </Typography>
        </Box>
      )}

      {/* Expanded Attributes */}
      <Collapse in={expanded}>
        {log.attributes && Object.keys(log.attributes).length > 0 && (
          <Box
            sx={{
              mt: 1,
              p: 1,
              bgcolor: 'grey.100',
              borderRadius: 1,
              fontFamily: 'monospace',
              fontSize: 12,
            }}
          >
            <Typography variant="caption" color="text.secondary" gutterBottom>
              Attributes:
            </Typography>
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
              {JSON.stringify(log.attributes, null, 2)}
            </pre>
          </Box>
        )}
      </Collapse>
    </Paper>
  );
};

export default LogEntry;