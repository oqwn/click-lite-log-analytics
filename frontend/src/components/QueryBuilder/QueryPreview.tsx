import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  IconButton,
  CircularProgress,
  Alert,
} from '@mui/material';
import { Refresh, ContentCopy, CheckCircle } from '@mui/icons-material';
import { QueryBuilder } from '../../types/api';

interface QueryPreviewProps {
  queryBuilder: QueryBuilder;
  generatedSQL: string;
  onGenerate: () => void;
  loading: boolean;
}

export const QueryPreview: React.FC<QueryPreviewProps> = ({
  queryBuilder,
  generatedSQL,
  onGenerate,
  loading,
}) => {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatedSQL);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      // Failed to copy text
    }
  };

  const hasRequiredFields = () => {
    return queryBuilder.fields.length > 0;
  };

  const getQuerySummary = () => {
    const parts = [];
    
    if (queryBuilder.fields.length > 0) {
      parts.push(`${queryBuilder.fields.length} field${queryBuilder.fields.length > 1 ? 's' : ''}`);
    }
    
    if (queryBuilder.filters.length > 0) {
      parts.push(`${queryBuilder.filters.length} filter${queryBuilder.filters.length > 1 ? 's' : ''}`);
    }
    
    if (queryBuilder.aggregations.length > 0) {
      parts.push(`${queryBuilder.aggregations.length} aggregation${queryBuilder.aggregations.length > 1 ? 's' : ''}`);
    }
    
    if (queryBuilder.group_by.length > 0) {
      parts.push(`grouped by ${queryBuilder.group_by.length} field${queryBuilder.group_by.length > 1 ? 's' : ''}`);
    }
    
    if (queryBuilder.order_by.length > 0) {
      parts.push(`sorted by ${queryBuilder.order_by.length} field${queryBuilder.order_by.length > 1 ? 's' : ''}`);
    }
    
    if (queryBuilder.limit) {
      parts.push(`limited to ${queryBuilder.limit} rows`);
    }

    return parts.length > 0 ? parts.join(', ') : 'No query configuration';
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">SQL Preview</Typography>
        <Button
          variant="outlined"
          startIcon={loading ? <CircularProgress size={16} /> : <Refresh />}
          onClick={onGenerate}
          disabled={loading || !hasRequiredFields()}
        >
          Generate SQL
        </Button>
      </Box>

      <Box sx={{ mb: 3 }}>
        <Typography variant="subtitle2" gutterBottom>
          Query Summary:
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {getQuerySummary()}
        </Typography>
      </Box>

      {!hasRequiredFields() && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          Please select at least one field to generate SQL query.
        </Alert>
      )}

      {generatedSQL ? (
        <Box>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
            <Typography variant="subtitle2">Generated SQL:</Typography>
            <IconButton
              size="small"
              onClick={handleCopy}
              color={copied ? 'success' : 'default'}
              title="Copy SQL"
            >
              {copied ? <CheckCircle /> : <ContentCopy />}
            </IconButton>
          </Box>
          
          <Paper 
            variant="outlined" 
            sx={{ 
              p: 2, 
              bgcolor: 'grey.50',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
              maxHeight: '400px',
              overflow: 'auto',
            }}
          >
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
              {generatedSQL}
            </pre>
          </Paper>
          
          {copied && (
            <Alert severity="success" sx={{ mt: 1 }}>
              SQL copied to clipboard!
            </Alert>
          )}
        </Box>
      ) : (
        <Paper 
          variant="outlined" 
          sx={{ 
            p: 3, 
            textAlign: 'center',
            bgcolor: 'grey.50',
            color: 'text.secondary'
          }}
        >
          <Typography>
            {hasRequiredFields() 
              ? 'Click "Generate SQL" to see the query preview'
              : 'Configure your query to see SQL preview'
            }
          </Typography>
        </Paper>
      )}

      <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
        <Typography variant="caption" color="text.secondary">
          <strong>Note:</strong> The generated SQL will be executed against your ClickHouse database. 
          Make sure to review the query before execution, especially when working with large datasets.
        </Typography>
      </Box>
    </Paper>
  );
};