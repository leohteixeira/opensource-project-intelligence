import { describe, expect, it } from 'vitest';

import { ICON_NAMES } from './icons';
import { STATUS, type StatusKey } from './status';

/**
 * Colour is never the only cue: every status in the frozen vocabulary carries a glyph and a word
 * as well as a tone, and the glyph it names has to exist in the icon registry.
 */
describe('STATUS vocabulary', () => {
  const entries = Object.entries(STATUS) as [StatusKey, (typeof STATUS)[StatusKey]][];

  it('covers exactly the frozen state list', () => {
    const expected = [
      'ready',
      'available',
      'succeeded',
      'recommended',
      'conditional',
      'not_recommended',
      'insufficient_data',
      'unknown',
      'not_applicable',
      'queued',
      'running',
      'stale',
      'failed',
      'cancelled',
      'paused',
      'archived',
      'active',
      'forecast',
      'observed',
      'ai',
      'translated',
    ];

    expect(Object.keys(STATUS).sort()).toEqual(expected.sort());
  });

  it.each(entries)('%s carries a word and a glyph the registry knows', (key, spec) => {
    expect(spec.label, `${key} has no word`).not.toBe('');
    expect(ICON_NAMES, `${key} names an unknown glyph`).toContain(spec.glyph);
  });

  it('keeps the three missing-data states distinct', () => {
    const unknown = STATUS.unknown;
    const notApplicable = STATUS.not_applicable;
    const insufficient = STATUS.insufficient_data;
    const words = [unknown.label, notApplicable.label, insufficient.label];
    const glyphs = [unknown.glyph, notApplicable.glyph, insufficient.glyph];

    expect(new Set(words).size).toBe(3);
    expect(new Set(glyphs).size).toBe(3);
  });
});
