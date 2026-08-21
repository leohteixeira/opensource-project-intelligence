// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it } from 'vitest';

import { AdoptionSecurityScreen } from './project-evidence/AdoptionSecurityScreen';
import { AiRunsScreen } from './project-evidence/AiRunsScreen';
import { ContributorsScreen } from './project-evidence/ContributorsScreen';
import { ExportsScreen } from './project-evidence/ExportsScreen';
import { KnowledgeScreen } from './project-evidence/KnowledgeScreen';
import { LifecycleScreen } from './project-evidence/LifecycleScreen';
import { OverviewScreen } from './project-evidence/OverviewScreen';
import { ReleasesScreen } from './project-evidence/ReleasesScreen';
import { SourcesScreen } from './project-evidence/SourcesScreen';

/**
 * Each project-evidence screen carries the states the UI/UX contract specifies for its surface
 * behind one switcher. Most of the code lives in those branches, so every state is rendered here
 * rather than only the one the screen lands on.
 */
interface Mounted {
  readonly container: HTMLElement;
  readonly root: Root;
}

let mounted: Mounted | null = null;

const noop = () => {};

function render(element: React.ReactElement): HTMLElement {
  const container = document.createElement('div');

  document.body.append(container);

  const root = createRoot(container);

  act(() => root.render(element));
  mounted = { container, root };

  return container;
}

afterEach(() => {
  if (!mounted) return;

  const { container, root } = mounted;

  act(() => root.unmount());
  container.remove();
  mounted = null;
});

const SCREENS = [
  { name: 'overview', element: <OverviewScreen onGoTab={noop} onOpenLifecycle={noop} /> },
  { name: 'contributors', element: <ContributorsScreen /> },
  { name: 'adoption and security', element: <AdoptionSecurityScreen /> },
  { name: 'releases (en)', element: <ReleasesScreen locale="en" onOpenRun={noop} /> },
  { name: 'releases (pt-BR)', element: <ReleasesScreen locale="pt-BR" onOpenRun={noop} /> },
  { name: 'knowledge', element: <KnowledgeScreen onOpenRun={noop} /> },
  { name: 'sources', element: <SourcesScreen onOpenLifecycle={noop} /> },
  { name: 'exports', element: <ExportsScreen onBack={noop} /> },
  { name: 'AI runs', element: <AiRunsScreen onBack={noop} /> },
  { name: 'lifecycle', element: <LifecycleScreen onBack={noop} /> },
];

describe.each(SCREENS)('$name screen', ({ element }) => {
  it('renders every state its switcher offers', () => {
    const container = render(element);
    const labels = [...container.querySelectorAll('[role="tab"]')].map(
      (tab) => tab.textContent ?? '',
    );

    expect(labels.length).toBeGreaterThan(1);

    for (const label of labels) {
      const tab = [...container.querySelectorAll<HTMLButtonElement>('[role="tab"]')].find(
        (candidate) => candidate.textContent === label,
      );

      expect(tab, `state ${label} disappeared from the switcher`).toBeDefined();

      act(() => tab?.click());

      // A state that renders nothing is a broken branch, not an empty surface.
      expect(
        (container.textContent ?? '').length,
        `state ${label} rendered nothing`,
      ).toBeGreaterThan(200);
    }
  });
});
