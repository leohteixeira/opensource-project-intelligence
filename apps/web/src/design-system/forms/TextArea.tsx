import type { CSSProperties, ChangeEventHandler, ReactNode } from 'react';

import { FormField } from './FormField';

export interface TextAreaProps {
  readonly id: string;
  readonly label: ReactNode;
  readonly hint?: ReactNode;
  readonly error?: ReactNode;
  readonly required?: boolean;
  readonly optional?: boolean;
  readonly value?: string;
  readonly defaultValue?: string;
  readonly placeholder?: string;
  readonly onChange?: ChangeEventHandler<HTMLTextAreaElement>;
  readonly rows?: number;
  readonly maxLength?: number;
  readonly counter?: boolean;
  readonly disabled?: boolean;
  readonly style?: CSSProperties;
}

export function TextArea({
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
  rows = 4,
  maxLength,
  counter,
  disabled,
  style,
}: TextAreaProps) {
  const used = typeof value === 'string' ? value.length : 0;

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
      {(aria) => (
        <span style={{ display: 'grid', gap: 'var(--space-05)' }}>
          <textarea
            id={aria.id}
            className="opi-field"
            rows={rows}
            value={value}
            defaultValue={defaultValue}
            placeholder={placeholder}
            onChange={onChange}
            disabled={disabled}
            maxLength={maxLength}
            aria-describedby={aria.describedBy}
            aria-invalid={aria.invalid ? true : undefined}
            style={{
              padding: 'var(--space-1) var(--space-15)',
              font: 'var(--type-body)',
              resize: 'vertical',
              minHeight: 88,
            }}
          />
          {counter && maxLength ? (
            <span
              style={{
                justifySelf: 'end',
                font: 'var(--type-caption)',
                color: 'var(--text-tertiary)',
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {used} / {maxLength}
            </span>
          ) : null}
        </span>
      )}
    </FormField>
  );
}
