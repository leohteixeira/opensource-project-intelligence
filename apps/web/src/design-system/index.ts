/**
 * Open Source Project Intelligence design system.
 *
 * Ported from the versioned design-system project (`OPI Design System`, August 2026). The rules
 * that break the product if ignored:
 *
 * 1. Missing data is never zero. `Unknown`, `Not applicable` and `Insufficient data` are three
 *    distinct states, each with a glyph and a word, and none of them is a blank cell or a `0`.
 * 2. Health is seven independent dimensions. The overall score is a clearly secondary summary
 *    shown only when its evidence requirements are met, labelled with its version.
 * 3. Observed, forecast, AI-generated and human override are four separate, labelled things.
 * 4. Every number carries a unit, a window, a cutoff and a definition version.
 * 5. Colour is never the only cue. The primary action colour is `--blue-500`; every other hue is
 *    spoken for by the status system.
 * 6. Cards are white on a light page: 12px radius, hairline border, one quiet `--shadow-card`.
 *    Navigation is a single 64px top bar — there is no rail.
 */

export { Avatar, type AvatarProps } from './core/Avatar';
export { Banner, type BannerProps, type BannerTone } from './core/Banner';
export { Button, type ButtonProps, type ButtonSize, type ButtonVariant } from './core/Button';
export { EmptyState, type EmptyStateProps } from './core/EmptyState';
export { Icon, type IconProps } from './core/Icon';
export { ICON_NAMES, type IconName } from './core/icons';
export { IconButton, type IconButtonProps } from './core/IconButton';
export { Link, type LinkProps } from './core/Link';
export { Panel, type PanelProps } from './core/Panel';
export { Progress, type ProgressProps } from './core/Progress';
export { Skeleton, type SkeletonProps } from './core/Skeleton';
export { StatusBadge, type StatusBadgeProps } from './core/StatusBadge';
export { STATUS, type StatusKey } from './core/status';
export { Tooltip, type TooltipProps } from './core/Tooltip';
export { VisuallyHidden, type VisuallyHiddenProps } from './core/VisuallyHidden';
export { Wordmark, type WordmarkProps } from './core/Wordmark';

export { Checkbox, type CheckboxProps } from './forms/Checkbox';
export { DateRangeField, type DateRangeFieldProps } from './forms/DateRangeField';
export { FormField, type FieldAria, type FormFieldProps } from './forms/FormField';
export { RadioGroup, type RadioGroupProps, type RadioOption } from './forms/RadioGroup';
export { Select, type SelectOption, type SelectProps } from './forms/Select';
export { TextArea, type TextAreaProps } from './forms/TextArea';
export { TextField, type TextFieldProps } from './forms/TextField';

export {
  AppShell,
  type AppShellProps,
  type NavItem,
  type ShellMember,
} from './navigation/AppShell';
export { FilterBar, type AppliedFilter, type FilterBarProps } from './navigation/FilterBar';
export { Menu, type MenuItem, type MenuProps } from './navigation/Menu';
export { Pagination, type PaginationProps } from './navigation/Pagination';
export { Tabs, type TabItem, type TabsProps } from './navigation/Tabs';

export { Dialog, type DialogProps } from './overlays/Dialog';
export { Drawer, type DrawerProps } from './overlays/Drawer';

export {
  DefinitionList,
  type DefinitionItem,
  type DefinitionListProps,
} from './data/DefinitionList';
export { Table, type TableColumn, type TableProps } from './data/Table';

export {
  ComparisonMatrix,
  type ComparisonMatrixProps,
  type MatrixCell,
  type MatrixRow,
} from './intelligence/ComparisonMatrix';
export { bestCellIndex } from './intelligence/ranking';
export {
  CoverageDisclosure,
  type CoverageDisclosureProps,
  type CoverageSource,
} from './intelligence/CoverageDisclosure';
export {
  EvidenceLink,
  type EvidenceKind,
  type EvidenceLinkProps,
} from './intelligence/EvidenceLink';
export {
  HealthDimensions,
  type HealthDimension,
  type HealthDimensionsProps,
  type OverallScore,
} from './intelligence/HealthDimensions';
export { JobProgress, type JobProgressProps } from './intelligence/JobProgress';
export { MetricValue, type MetricValueProps } from './intelligence/MetricValue';
export {
  RadarList,
  type RadarEntry,
  type RadarListProps,
  type RadarRing,
} from './intelligence/RadarList';
export {
  Recommendation,
  type DecisiveFactor,
  type RecommendationProps,
  type RecommendationResult,
} from './intelligence/Recommendation';
export {
  RunMetadata,
  type RunMetadataLabels,
  type RunMetadataProps,
} from './intelligence/RunMetadata';
export { TrendChart, type SeriesPoint, type TrendChartProps } from './intelligence/TrendChart';
