import type { CSSProperties, ChangeEventHandler, ReactNode } from 'react';

import { Icon } from '../core/Icon';
import type { IconName } from '../core/icons';
import { FormField, type FieldAria } from './FormField';

export interface TextFieldProps {
  readonly id: string;
  readonly label?: ReactNode;
  readonly hint?: ReactNode;
  readonly error?: ReactNode;
  readonly required?: boolean;
  readonly optional?: boolean;
  readonly value?: string;
  readonly defaultValue?: string;
  readonly placeholder?: string;
  readonly onChange?: ChangeEventHandler<HTMLInputElement>;
  readonly type?: 'text' | 'search' | 'url' | 'email' | 'number' | 'date';
  readonly mono?: boolean;
  readonly iconStart?: IconName;
  readonly suffix?: ReactNode;
  readonly disabled?: boolean;
  readonly readOnly?: boolean;
  readonly size?: 'lg' | 'md';
  readonly inputMode?: 'text' | 'numeric' | 'search' | 'url' | 'email';
  readonly style?: CSSProperties;
}

export function TextField({
  id,
  label,
  hint,
  error,
  required,
  optional,
  value,
  defaultValue,
  placeholder,
  onChange,
  type = 'text',
  mono,
  iconStart,
  suffix,
  disabled,
  readOnly,
  size = 'lg',
  inputMode,
  style,
}: TextFieldProps) {
  const height = size === 'md' ? 'var(--control-h-md)' : 'var(--control-h-lg)';
  const input = (aria: FieldAria) => (
    <span style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
      {iconStart ? (
        <Icon
          name={iconStart}
          size={15}
          style={{ position: 'absolute', left: 10, color: 'var(--text-tertiary)' }}
        />
      ) : null}
      <input
        id={aria.id}
        className="opi-field"
        type={type}
        value={value}
        defaultValue={defaultValue}
        placeholder={placeholder}
        onChange={onChange}
        disabled={disabled}
        readOnly={readOnly}
        inputMode={inputMode}
        required={required}
        aria-describedby={aria.describedBy}
        aria-invalid={aria.invalid ? true : undefined}
        aria-label={label ? undefined : placeholder}
        style={{
          height,
          padding: `0 ${suffix ? '56px' : 'var(--space-15)'} 0 ${iconStart ? '32px' : 'var(--space-15)'}`,
          font: mono ? 'var(--type-mono)' : 'var(--type-body)',
          fontSize: 'var(--text-base)',
        }}
      />
      {suffix ? (
        <span
          style={{
            position: 'absolute',
            right: 10,
            font: 'var(--type-caption)',
            color: 'var(--text-tertiary)',
          }}
        >
          {suffix}
        </span>
      ) : null}
    </span>
  );

  if (!label) {
    return (
      <span style={style}>{input({ id, describedBy: undefined, invalid: Boolean(error) })}</span>
    );
  }

  return (
    <FormField
      id={id}
      label={label}
      hint={hint}
      error={error}
      required={required}
      optional={optional}
      style={style}
    >
      {input}
    </FormField>
  );
}
