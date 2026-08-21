import type { NavItem } from '../design-system';

/**
 * Exactly five primary destinations — the mobile bottom bar carries five items and never six.
 * Administration collapses into one labelled menu in the same bar; it is never a second row and
 * never a rail.
 */
export const PRIMARY_NAV: readonly NavItem[] = [
  { key: 'portfolio', label: 'Portfolio', icon: 'layout-dashboard' },
  { key: 'projects', label: 'Projects', icon: 'package' },
  { key: 'compare', label: 'Compare', icon: 'table-2' },
  { key: 'radar', label: 'Radar', icon: 'radar' },
  { key: 'alerts', label: 'Alerts', icon: 'bell', badge: 2 },
];

export const ADMIN_NAV: readonly NavItem[] = [
  { key: 'members', label: 'Members', icon: 'users' },
  { key: 'policies', label: 'Policies', icon: 'scale' },
  { key: 'audit', label: 'Audit', icon: 'file-text' },
  { key: 'operations', label: 'Operations', icon: 'activity' },
];
