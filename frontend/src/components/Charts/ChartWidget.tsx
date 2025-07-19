import React from 'react';
import { Box, Typography, Alert } from '@mui/material';
import { LineChart } from '@mui/x-charts/LineChart';
import { BarChart } from '@mui/x-charts/BarChart';
import { PieChart } from '@mui/x-charts/PieChart';
import { ScatterChart } from '@mui/x-charts/ScatterChart';
import { ChartData } from '../../types/api';

interface ChartWidgetProps {
  data: unknown;
  config: Record<string, unknown>;
  height: number;
}

export const ChartWidget: React.FC<ChartWidgetProps> = ({ data, config, height }) => {
  if (!data) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', height }}>
        <Typography color="text.secondary">No data available</Typography>
      </Box>
    );
  }

  // Transform data to chart format
  const chartData = transformDataToChart(data);
  
  if (!chartData || chartData.labels.length === 0) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', height }}>
        <Typography color="text.secondary">No data to display</Typography>
      </Box>
    );
  }

  const chartType = config?.chart_type || 'line';
  const showLegend = config?.show_legend !== false;
  const showGrid = config?.show_grid !== false;
  const colors = config?.colors || ['#1976d2', '#dc004e', '#9c27b0', '#2e7d32', '#ed6c02'];

  try {
    switch (chartType) {
      case 'line':
        return renderLineChart(chartData, { height, showLegend, showGrid, colors });
      case 'bar':
        return renderBarChart(chartData, { height, showLegend, showGrid, colors });
      case 'area':
        return renderAreaChart(chartData, { height, showLegend, showGrid, colors });
      case 'pie':
        return renderPieChart(chartData, { height, showLegend, colors });
      case 'scatter':
        return renderScatterChart(chartData, { height, showLegend, showGrid, colors });
      default:
        return (
          <Alert severity="warning">
            Unsupported chart type: {chartType}
          </Alert>
        );
    }
  } catch (error) {
    return (
      <Alert severity="error">
        Error rendering chart: {error instanceof Error ? error.message : 'Unknown error'}
      </Alert>
    );
  }
};

function transformDataToChart(data: unknown): ChartData | null {
  if (!data) return null;

  // Handle ChartData format directly
  if (data.labels && data.datasets) {
    return data as ChartData;
  }

  // Handle array of objects (SQL results)
  if (Array.isArray(data) && data.length > 0) {
    const firstRow = data[0];
    const keys = Object.keys(firstRow);
    
    if (keys.length < 2) {
      return null;
    }

    // Assume first column is labels/x-axis, rest are data series
    const labelKey = keys[0];
    const dataKeys = keys.slice(1);

    const labels = data.map(row => String(row[labelKey]));
    const datasets = dataKeys.map((key, index) => ({
      label: key,
      data: data.map(row => Number(row[key]) || 0),
      background_color: `hsl(${(index * 137.5) % 360}, 70%, 50%)`,
      border_color: `hsl(${(index * 137.5) % 360}, 70%, 40%)`,
    }));

    return { labels, datasets };
  }

  return null;
}

function renderLineChart(chartData: ChartData, options: Record<string, unknown>) {
  const series = chartData.datasets.map((dataset, index) => ({
    id: `series-${index}`,
    label: dataset.label,
    data: dataset.data,
    color: options.colors[index % options.colors.length],
  }));

  return (
    <LineChart
      xAxis={[{ 
        scaleType: 'point', 
        data: chartData.labels,
      }]}
      series={series}
      height={options.height}
      margin={{ left: 70, right: 20, top: 20, bottom: 60 }}
      grid={options.showGrid}
      slotProps={{
        legend: {
          hidden: !options.showLegend,
        },
      }}
    />
  );
}

function renderBarChart(chartData: ChartData, options: Record<string, unknown>) {
  const series = chartData.datasets.map((dataset, index) => ({
    id: `series-${index}`,
    label: dataset.label,
    data: dataset.data,
    color: options.colors[index % options.colors.length],
  }));

  return (
    <BarChart
      xAxis={[{ 
        scaleType: 'band', 
        data: chartData.labels,
      }]}
      series={series}
      height={options.height}
      margin={{ left: 70, right: 20, top: 20, bottom: 60 }}
      grid={options.showGrid}
      slotProps={{
        legend: {
          hidden: !options.showLegend,
        },
      }}
    />
  );
}

function renderAreaChart(chartData: ChartData, options: Record<string, unknown>) {
  const series = chartData.datasets.map((dataset, index) => ({
    id: `series-${index}`,
    label: dataset.label,
    data: dataset.data,
    color: options.colors[index % options.colors.length],
    area: true,
  }));

  return (
    <LineChart
      xAxis={[{ 
        scaleType: 'point', 
        data: chartData.labels,
      }]}
      series={series}
      height={options.height}
      margin={{ left: 70, right: 20, top: 20, bottom: 60 }}
      grid={options.showGrid}
      slotProps={{
        legend: {
          hidden: !options.showLegend,
        },
      }}
    />
  );
}

function renderPieChart(chartData: ChartData, options: Record<string, unknown>) {
  // For pie charts, use the first dataset
  const dataset = chartData.datasets[0];
  if (!dataset) return null;

  const data = chartData.labels.map((label, index) => ({
    id: index,
    value: dataset.data[index],
    label: label,
  }));

  return (
    <PieChart
      series={[{
        data,
        highlightScope: { faded: 'global', highlighted: 'item' },
        faded: { innerRadius: 30, additionalRadius: -30, color: 'gray' },
      }]}
      height={options.height}
      margin={{ left: 20, right: 20, top: 20, bottom: 20 }}
      slotProps={{
        legend: {
          hidden: !options.showLegend,
          direction: 'row',
          position: { vertical: 'bottom', horizontal: 'middle' },
        },
      }}
    />
  );
}

function renderScatterChart(chartData: ChartData, options: Record<string, unknown>) {
  // For scatter plots, assume we have x,y pairs
  const series = chartData.datasets.map((dataset, index) => {
    const data = chartData.labels.map((label, i) => ({
      x: i, // Use index as x value, or parse label if it's numeric
      y: dataset.data[i],
      id: i,
    }));

    return {
      id: `series-${index}`,
      label: dataset.label,
      data,
      color: options.colors[index % options.colors.length],
    };
  });

  return (
    <ScatterChart
      series={series}
      height={options.height}
      margin={{ left: 70, right: 20, top: 20, bottom: 60 }}
      grid={options.showGrid}
      slotProps={{
        legend: {
          hidden: !options.showLegend,
        },
      }}
    />
  );
}