// @vitest-environment jsdom
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, describe, expect, it } from 'vitest';

import { AdministrationKit } from './administration/AdministrationKit';
import { ProjectDetailScreen } from './workspace/ProjectDetailScreen';
import { PROJECTS } from './workspace/fixtures';
import { ProjectEvidenceKit } from './project-evidence/ProjectEvidenceKit';
import { PublicCatalogKit } from './public-catalog/PublicCatalogKit';
import { WorkspaceKit } from './workspace/WorkspaceKit';

/**
 * Every shell has to mount and reach its landing surface. The assertions are deliberately about
 * the contract rather than about wording: a shell that renders its first screen without throwing
 * proves the component tree, the fixtures and the token references line up.
 */
interface Mounted {
  readonly container: HTMLElement;
  readonly root: Root;
}

let mounted: Mounted | null = null;

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

const SHELLS = [
  {
    name: 'public catalog',
    element: <PublicCatalogKit />,
    expected: ['Public open source technologies tracked by this workspace', 'Sign in'],
  },
  {
    name: 'workspace',
    element: <WorkspaceKit />,
    expected: ['Portfolio', 'Requires attention', 'Observed trends and early warnings'],
  },
  {
    name: 'project evidence',
    element: <ProjectEvidenceKit />,
    expected: ['Specified state', 'Seven independent dimensions', 'Recommendation'],
  },
  {
    name: 'administration',
    element: <AdministrationKit />,
    expected: ['Applicant queue', 'Service accounts', 'The last active Admin is protected'],
  },
];

describe.each(SHELLS)('$name shell', ({ element, expected }) => {
  it('mounts and reaches its landing surface', () => {
    const container = render(element);
    const text = container.textContent ?? '';

    for (const fragment of expected) {
      expect(text).toContain(fragment);
    }
  });

  it('renders the skip link every shell owes a keyboard user', () => {
    const container = render(element);

    expect(container.querySelector('.opi-skip')).not.toBeNull();
  });
});

const PROJECT_TABS = [
  'overview',
  'health',
  'contributors',
  'adoption-security',
  'trends',
  'topics',
  'releases',
  'knowledge',
  'sources-jobs',
];

describe('project detail tabs', () => {
  const project = PROJECTS[0];

  it.each(PROJECT_TABS)('renders the %s tab', (tab) => {
    expect(project).toBeDefined();

    if (!project) return;

    const container = render(<ProjectDetailScreen project={project} tab={tab} onTab={() => {}} />);

    // Either the tab is built here or it says where it is built; neither is a blank surface.
    expect((container.textContent ?? '').length).toBeGreaterThan(400);
    expect(container.querySelector(`[role="tab"][aria-selected="true"]`)).not.toBeNull();
  });
});

/** Credential presence is shown; a credential value never is, on any administration surface. */
const SECRET_SHAPES = [
  /ghp_[A-Za-z0-9]{10,}/,
  /gh[pousr]_[A-Za-z0-9]{10,}/,
  /sk-[A-Za-z0-9]{10,}/,
  /Bearer +[A-Za-z0-9._-]{10,}/,
  /eyJ[A-Za-z0-9_-]{10,}/,
];

describe('administration secrecy', () => {
  it.each(SECRET_SHAPES)('renders nothing shaped like %s', (shape) => {
    const container = render(<AdministrationKit />);

    expect(container.textContent ?? '').not.toMatch(shape);
  });

  it('states the redaction rule instead of leaving it implied', () => {
    const container = render(<AdministrationKit />);

    expect(container.textContent ?? '').toContain(
      'No token or secret field is ever returned, even to an Admin.',
    );
  });
});
