import { describe, expect, it } from 'vitest';

import { readConfig } from './config';

describe('readConfig', () => {
  it('falls back to the port assigned to this product', () => {
    expect(readConfig({}).apiBaseUrl).toBe('http://localhost:8100');
  });

  it('prefers an explicitly configured base url', () => {
    expect(readConfig({ VITE_API_BASE_URL: 'https://api.example.test' }).apiBaseUrl).toBe(
      'https://api.example.test',
    );
  });

  it('ignores a blank base url', () => {
    expect(readConfig({ VITE_API_BASE_URL: '   ' }).apiBaseUrl).toBe('http://localhost:8100');
  });
});
