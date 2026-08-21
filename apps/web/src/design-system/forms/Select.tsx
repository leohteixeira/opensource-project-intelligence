import type { CSSProperties, ChangeEventHandler, ReactNode } from 'react';

import { Icon } from '../core/Icon';
import { FormField, type FieldAria } from './FormField';

export interface SelectOption {
  readonly value: string;
  readonly label: string;
  readonly disabled?: boolean;
}

export interface SelectProps {
  readonly id: string;
  readonly label?: ReactNode;
  readonly hint?: ReactNode;
  readonly error?: ReactNode;
  readonly required?: boolean;
  readonly options: readonly (string | SelectOption)[];
  readonly value?: string;
  readonly defaultValue?: string;
  readonly onChange?: ChangeEventHandler<HTMLSelectElement>;
  readonly disabled?: boolean;
  readonly size?: 'lg' | 'md';
  readonly placeholder?: string;
  readonly style?: CSSProperties;
}

function normalize(option: string | SelectOption): SelectOption {
  return typeof option === 'string' ? { value: option, label: option } : option;
}

/**
 * Native select. A listbox is only justified where an option needs rich content; a plain typed
 * choice stays native so mobile, keyboard and screen readers behave without work.
 */
export function Select({
  id,
  label,
  hint,
  error,
  required,
  options,
  value,
  defaultValue,
  onChange,
  disabled,
  size = 'lg',
  placeholder,
  style,
}: SelectProps) {
  const height = size === 'md' ? 'var(--control-h-md)' : 'var(--control-h-lg)';
  const control = (aria: FieldAria) => (
    <span style={{ position: 'relative', display: 'flex' }}>
      <select
        id={aria.id}
        className="opi-field"
        value={value}
        defaultValue={defaultValue}
        onChange={onChange}
        disabled={disabled}
        aria-describedby={aria.describedBy}
        aria-invalid={aria.invalid ? true : undefined}
        aria-label={label ? undefined : placeholder}
        style={{
          height,
          padding: '0 32px 0 var(--space-15)',
          font: 'var(--type-body)',
          fontSize: size === 'md' ? 'var(--text-sm)' : 'var(--text-base)',
          appearance: 'none',
          cursor: disabled ? 'not-allowed' : 'pointer',
        }}
      >
        {placeholder ? <option value="">{placeholder}</option> : null}
        {options.map(normalize).map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
      <Icon
        name="chevron-down"
        size={15}
        style={{
          position: 'absolute',
          right: 10,
          top: '50%',
          transform: 'translateY(-50%)',
          color: 'var(--text-secondary)',
          pointerEvents: 'none',
        }}
      />
    </span>
  );

  if (!label) {
    return (
      <span style={style}>{control({ id, describedBy: undefined, invalid: Boolean(error) })}</span>
    );
  }

  return (
    <FormField id={id} label={label} hint={hint} error={error} required={required} style={style}>
      {control}
    </FormField>
  );
}
