import React from 'react';
import {
  Box,
  Typography,
} from '@mui/material';

interface TextWidgetProps {
  data: unknown;
  config: Record<string, unknown>;
}

export const TextWidget: React.FC<TextWidgetProps> = ({ data, config }) => {
  let content: string = '';

  if (typeof data === 'string') {
    content = data;
  } else if (typeof data === 'object' && data?.text) {
    content = data.text;
  } else if (Array.isArray(data) && data.length > 0) {
    // If it's an array, join the values or stringify the first object
    if (typeof data[0] === 'string') {
      content = data.join('\n');
    } else {
      content = JSON.stringify(data[0], null, 2);
    }
  } else if (data) {
    content = JSON.stringify(data, null, 2);
  } else {
    content = 'No data available';
  }

  const fontSize = config?.font_size || 'body1';
  const textAlign = (config?.text_align as string) || 'left';
  const fontWeight = config?.font_weight || 'normal';

  return (
    <Box
      sx={{
        p: 2,
        height: '100%',
        overflow: 'auto',
        textAlign: textAlign as any,
      }}
    >
      <Typography
        variant={fontSize}
        component="div"
        sx={{
          fontWeight,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}
      >
        {content}
      </Typography>
    </Box>
  );
};