import type { MatrixRow } from './ComparisonMatrix';

/**
 * The best-value marker. A row that is not comparable is never ranked, and a cell carrying a
 * status instead of a number is excluded from the comparison rather than treated as zero — so a
 * missing value can never win or lose a row.
 */
export function bestCellIndex(row: MatrixRow, count: number): number | null {
  if (!row.comparable) return null;

  const values = Array.from({ length: count }, (_, index) => {
    const cell = row.cells[index];

    return cell && typeof cell.value === 'number' ? cell.value : null;
  });
  const numbers = values.filter((value): value is number => value !== null);

  if (numbers.length < 2) return null;

  const target = row.betterIs === 'lower' ? Math.min(...numbers) : Math.max(...numbers);
  const index = values.indexOf(target);

  return index === -1 ? null : index;
}
