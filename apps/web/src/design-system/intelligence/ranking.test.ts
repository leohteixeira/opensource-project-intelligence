import { describe, expect, it } from 'vitest';

import type { MatrixRow } from './ComparisonMatrix';
import { bestCellIndex } from './ranking';

interface Case {
  readonly name: string;
  readonly row: MatrixRow;
  readonly projects: number;
  readonly expected: number | null;
}

const row = (overrides: Partial<MatrixRow>): MatrixRow => ({
  metric: 'release_frequency',
  unit: 'releases / 90d',
  comparable: true,
  betterIs: 'higher',
  cells: [],
  ...overrides,
});

const CASES: readonly Case[] = [
  {
    name: 'marks the highest value when higher is better',
    row: row({ cells: [{ value: 8 }, { value: 14 }, { value: 2 }] }),
    projects: 3,
    expected: 1,
  },
  {
    name: 'marks the lowest value when lower is better',
    row: row({ betterIs: 'lower', cells: [{ value: 9.4 }, { value: 212 }, { value: 4.1 }] }),
    projects: 3,
    expected: 2,
  },
  {
    name: 'never ranks a row the catalog declares incomparable',
    row: row({ comparable: false, cells: [{ value: 0.12 }, { value: 0.31 }] }),
    projects: 2,
    expected: null,
  },
  {
    name: 'ignores a cell carrying a status instead of a number',
    row: row({ cells: [{ value: 8 }, { status: 'insufficient_data' }, { value: 14 }] }),
    projects: 3,
    expected: 2,
  },
  {
    name: 'does not treat a missing value as zero when lower is better',
    row: row({ betterIs: 'lower', cells: [{ value: 3 }, { status: 'not_applicable' }] }),
    projects: 2,
    expected: null,
  },
  {
    name: 'ranks a measured zero, which is a value and not a gap',
    row: row({ betterIs: 'lower', cells: [{ value: 0 }, { value: 3 }] }),
    projects: 2,
    expected: 0,
  },
  {
    name: 'needs at least two numbers to declare a best value',
    row: row({ cells: [{ value: 8 }, { status: 'unknown' }, { status: 'unknown' }] }),
    projects: 3,
    expected: null,
  },
  {
    name: 'reads no further than the projects actually selected',
    row: row({ cells: [{ value: 8 }, { value: 14 }, { value: 99 }] }),
    projects: 2,
    expected: 1,
  },
  {
    name: 'reports nothing when the row carries no cell at all',
    row: row({ cells: [] }),
    projects: 3,
    expected: null,
  },
];

/**
 * A missing, not-applicable or incomparable cell is never ranked: the marker must not turn an
 * absent measurement into a winner or a loser.
 */
describe('bestCellIndex', () => {
  it.each(CASES)('$name', ({ row: matrixRow, projects, expected }) => {
    expect(bestCellIndex(matrixRow, projects)).toBe(expected);
  });
});
