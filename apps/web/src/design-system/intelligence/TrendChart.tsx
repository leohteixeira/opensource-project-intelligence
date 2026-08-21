import type { CSSProperties } from 'react';

import { Icon } from '../core/Icon';

export interface SeriesPoint {
  readonly label: string;
  readonly value: number;
}

type Point = readonly [number, number];

/**
 * Supplementary time series. The observed segment is a solid line; a forecast continues as a
 * dashed line in the forecast hue and is separated by a rule, so an inference can never read as a
 * measurement. The visually hidden table below it is the accessible primary representation.
 */
export interface TrendChartProps {
  readonly series: readonly SeriesPoint[];
  readonly forecast?: readonly SeriesPoint[];
  readonly label: string;
  readonly unit?: string;
  readonly width?: number;
  readonly height?: number;
  readonly baseline?: number;
  readonly showForecast?: boolean;
  readonly style?: CSSProperties;
}

export function TrendChart({
  series,
  forecast = [],
  label,
  unit,
  width = 320,
  height = 72,
  baseline,
  showForecast = true,
  style,
}: TrendChartProps) {
  const all = [...series, ...(showForecast ? forecast : [])];

  if (!all.length || !series.length) return null;

  const values = all.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const stepX = all.length > 1 ? (width - 8) / (all.length - 1) : width;
  const round = (value: number) => Math.round(value * 100) / 100;
  const at = (value: number, index: number): Point => [
    round(4 + index * stepX),
    round(height - 6 - ((value - min) / span) * (height - 12)),
  ];
  const path = (points: readonly Point[]) =>
    points.map((point, index) => `${index ? 'L' : 'M'}${point[0]} ${point[1]}`).join(' ');

  const observed = series.map((point, index) => at(point.value, index));
  const last = observed[observed.length - 1] as Point;
  const projected =
    showForecast && forecast.length
      ? [last, ...forecast.map((point, index) => at(point.value, series.length + index))]
      : [];
  const baselineY = baseline === undefined ? null : at(baseline, 0)[1];

  return (
    <figure style={{ margin: 0, display: 'grid', gap: 'var(--space-05)', minWidth: 0, ...style }}>
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        aria-hidden="true"
        style={{ maxWidth: '100%' }}
      >
        {baselineY !== null ? (
          <line
            x1="0"
            x2={width}
            y1={baselineY}
            y2={baselineY}
            stroke="var(--series-grid)"
            strokeWidth="1"
            strokeDasharray="2 3"
          />
        ) : null}
        <path
          d={path(observed)}
          fill="none"
          stroke="var(--series-2)"
          strokeWidth="1.75"
          strokeLinejoin="round"
          strokeLinecap="round"
        />
        {projected.length > 1 ? (
          <path
            d={path(projected)}
            fill="none"
            stroke="var(--series-forecast)"
            strokeWidth="1.75"
            strokeDasharray="4 3"
            strokeLinecap="round"
          />
        ) : null}
        {projected.length > 1 ? (
          <line
            x1={last[0]}
            x2={last[0]}
            y1="0"
            y2={height}
            stroke="var(--forecast-border)"
            strokeWidth="1"
          />
        ) : null}
        <circle cx={last[0]} cy={last[1]} r="2.5" fill="var(--series-2)" />
      </svg>
      <figcaption
        style={{
          display: 'flex',
          gap: 'var(--space-15)',
          flexWrap: 'wrap',
          font: 'var(--type-mono-xs)',
          color: 'var(--text-secondary)',
        }}
      >
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <Icon name="chart-line" size={11} style={{ color: 'var(--info-fg)' }} />
          observed {label}
          {unit ? ` (${unit})` : ''}
        </span>
        {projected.length > 1 ? (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              color: 'var(--forecast-fg)',
            }}
          >
            <Icon name="trending-up" size={11} />
            forecast
          </span>
        ) : null}
      </figcaption>
      <table className="opi-vh">
        <caption>{label} by period</caption>
        <thead>
          <tr>
            <th scope="col">Period</th>
            <th scope="col">Value</th>
            <th scope="col">Kind</th>
          </tr>
        </thead>
        <tbody>
          {series.map((point) => (
            <tr key={`observed-${point.label}`}>
              <th scope="row">{point.label}</th>
              <td>{point.value}</td>
              <td>observed</td>
            </tr>
          ))}
          {showForecast
            ? forecast.map((point) => (
                <tr key={`forecast-${point.label}`}>
                  <th scope="row">{point.label}</th>
                  <td>{point.value}</td>
                  <td>forecast</td>
                </tr>
              ))
            : null}
        </tbody>
      </table>
    </figure>
  );
}
