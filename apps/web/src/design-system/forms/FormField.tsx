import type { CSSProperties, ReactNode } from 'react';

import { Icon } from '../core/Icon';

export interface FieldAria {
  readonly id: string;
  readonly describedBy: string | undefined;
  readonly invalid: boolean;
}

/**
 * Label, optional hint, error and the control itself. Every field in the product is wrapped by
 * this, so the label/description/error wiring exists exactly once.
 */
export interface FormFieldProps {
  readonly id: string;
  readonly label: ReactNode;
  readonly hint?: ReactNode;
  readonly error?: ReactNode;
  readonly required?: boolean;
  readonly optional?: boolean;
  readonly children: ReactNode | ((aria: FieldAria) => ReactNode);
  readonly htmlFor?: string;
  readonly style?: CSSProperties;
}

export function FormField({
  id,
  label,
  hint,
  error,
  required,
  optional,
  children,
  htmlFor,
  style,
}: FormFieldProps) {
  const fieldId = htmlFor ?? id;
  const hintId = hint ? `${fieldId}-hint` : undefined;
  const errorId = error ? `${fieldId}-error` : undefined;
  const describedBy = [hintId, errorId].filter(Boolean).join(' ') || undefined;

  return (
    <div style={{ display: 'grid', gap: 'var(--space-05)', minWidth: 0, ...style }}>
      <label
        htmlFor={fieldId}
        style={{
          font: 'var(--type-ui)',
          color: 'var(--text-primary)',
          display: 'flex',
          gap: 'var(--space-05)',
          alignItems: 'baseline',
        }}
      >
        <span>{label}</span>
        {required ? (
          <span style={{ color: 'var(--critical-fg)' }} aria-hidden="true">
            *
          </span>
        ) : null}
        {required ? <span className="opi-vh">required</span> : null}
        {optional ? (
          <span style={{ font: 'var(--type-caption)', color: 'var(--text-tertiary)' }}>
            Optional
          </span>
        ) : null}
      </label>
      {hint ? (
        <p id={hintId} style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
          {hint}
        </p>
      ) : null}
      {typeof children === 'function'
        ? children({ id: fieldId, describedBy, invalid: Boolean(error) })
        : children}
      {error ? (
        <p
          id={errorId}
          style={{
            display: 'flex',
            gap: 'var(--space-05)',
            alignItems: 'center',
            font: 'var(--type-caption)',
            color: 'var(--critical-fg)',
          }}
        >
          <Icon name="circle-alert" size={13} />
          {error}
        </p>
      ) : null}
    </div>
  );
}
