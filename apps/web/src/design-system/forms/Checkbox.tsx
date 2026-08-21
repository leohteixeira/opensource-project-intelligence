import {
  useEffect,
  useRef,
  type CSSProperties,
  type ChangeEventHandler,
  type ReactNode,
} from 'react';

import { Icon } from '../core/Icon';

export interface CheckboxProps {
  readonly id: string;
  readonly label: ReactNode;
  readonly description?: ReactNode;
  readonly checked?: boolean;
  readonly indeterminate?: boolean;
  readonly onChange?: ChangeEventHandler<HTMLInputElement>;
  readonly disabled?: boolean;
  readonly name?: string;
  readonly value?: string;
  readonly style?: CSSProperties;
}

export function Checkbox({
  id,
  label,
  description,
  checked,
  indeterminate,
  onChange,
  disabled,
  name,
  value,
  style,
}: CheckboxProps) {
  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (ref.current) ref.current.indeterminate = Boolean(indeterminate);
  }, [indeterminate]);

  const on = Boolean(checked) || Boolean(indeterminate);

  return (
    <label
      htmlFor={id}
      style={{
        display: 'flex',
        gap: 'var(--space-1)',
        alignItems: 'flex-start',
        minHeight: 'var(--control-touch)',
        padding: 'var(--space-075) 0',
        cursor: disabled ? 'not-allowed' : 'pointer',
        ...style,
      }}
    >
      <span
        style={{
          position: 'relative',
          display: 'inline-flex',
          flex: 'none',
          width: 18,
          height: 18,
          marginTop: 2,
        }}
      >
        <input
          ref={ref}
          id={id}
          type="checkbox"
          name={name}
          value={value}
          checked={Boolean(checked)}
          onChange={onChange}
          disabled={disabled}
          style={{
            position: 'absolute',
            inset: 0,
            width: 18,
            height: 18,
            margin: 0,
            opacity: 0,
            cursor: 'inherit',
          }}
        />
        <span
          aria-hidden="true"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 18,
            height: 18,
            borderRadius: 'var(--radius-xs)',
            border: `1px solid ${on ? 'var(--ink)' : 'var(--border-field)'}`,
            background: disabled
              ? 'var(--surface-disabled)'
              : on
                ? 'var(--ink)'
                : 'var(--surface-card)',
            color: 'var(--white)',
            transition: 'var(--transition-control)',
          }}
        >
          {indeterminate ? (
            <Icon name="minus" size={13} strokeWidth={2.5} />
          ) : checked ? (
            <Icon name="check" size={13} strokeWidth={2.5} />
          ) : null}
        </span>
      </span>
      <span style={{ display: 'grid', gap: 1 }}>
        <span
          style={{
            font: 'var(--type-body)',
            fontSize: 'var(--text-sm)',
            color: disabled ? 'var(--text-disabled)' : 'var(--text-body)',
          }}
        >
          {label}
        </span>
        {description ? (
          <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
            {description}
          </span>
        ) : null}
      </span>
    </label>
  );
}
