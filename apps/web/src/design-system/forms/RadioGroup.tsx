import type { CSSProperties, ChangeEventHandler, ReactNode } from 'react';

export interface RadioOption {
  readonly value: string;
  readonly label: string;
  readonly description?: string;
  readonly disabled?: boolean;
}

/**
 * One mutually exclusive choice. Options may carry a description, which is how role choices and
 * export formats explain their consequence at the point of decision.
 */
export interface RadioGroupProps {
  readonly name: string;
  readonly legend: ReactNode;
  readonly hint?: ReactNode;
  readonly error?: ReactNode;
  readonly options: readonly (string | RadioOption)[];
  readonly value?: string;
  readonly onChange?: ChangeEventHandler<HTMLInputElement>;
  readonly disabled?: boolean;
  readonly orientation?: 'vertical' | 'horizontal';
  readonly style?: CSSProperties;
}

function normalize(option: string | RadioOption): RadioOption {
  return typeof option === 'string' ? { value: option, label: option } : option;
}

export function RadioGroup({
  name,
  legend,
  hint,
  error,
  options,
  value,
  onChange,
  disabled,
  orientation = 'vertical',
  style,
}: RadioGroupProps) {
  const horizontal = orientation === 'horizontal';

  return (
    <fieldset
      style={{
        border: 0,
        margin: 0,
        padding: 0,
        minWidth: 0,
        display: 'grid',
        gap: 'var(--space-05)',
        ...style,
      }}
    >
      <legend style={{ font: 'var(--type-ui)', color: 'var(--text-primary)', padding: 0 }}>
        {legend}
      </legend>
      {hint ? (
        <p style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>{hint}</p>
      ) : null}
      <div
        style={{
          display: horizontal ? 'flex' : 'grid',
          gap: horizontal ? 'var(--space-2)' : 0,
          flexWrap: 'wrap',
        }}
      >
        {options.map(normalize).map((option) => {
          const on = value === option.value;

          return (
            <label
              key={option.value}
              style={{
                display: 'flex',
                gap: 'var(--space-1)',
                alignItems: 'flex-start',
                minHeight: 'var(--control-touch)',
                padding: 'var(--space-075) 0',
                cursor: disabled ? 'not-allowed' : 'pointer',
              }}
            >
              <span
                style={{
                  position: 'relative',
                  display: 'inline-flex',
                  width: 18,
                  height: 18,
                  flex: 'none',
                  marginTop: 2,
                }}
              >
                <input
                  type="radio"
                  name={name}
                  value={option.value}
                  checked={on}
                  onChange={onChange}
                  disabled={disabled || option.disabled}
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
                    borderRadius: 'var(--radius-pill)',
                    border: `1px solid ${on ? 'var(--ink)' : 'var(--border-field)'}`,
                    background: 'var(--surface-card)',
                    transition: 'var(--transition-control)',
                  }}
                >
                  {on ? (
                    <span
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: 'var(--radius-pill)',
                        background: 'var(--ink)',
                      }}
                    />
                  ) : null}
                </span>
              </span>
              <span style={{ display: 'grid', gap: 1 }}>
                <span
                  style={{
                    font: 'var(--type-body)',
                    fontSize: 'var(--text-sm)',
                    color: 'var(--text-body)',
                  }}
                >
                  {option.label}
                </span>
                {option.description ? (
                  <span style={{ font: 'var(--type-caption)', color: 'var(--text-secondary)' }}>
                    {option.description}
                  </span>
                ) : null}
              </span>
            </label>
          );
        })}
      </div>
      {error ? (
        <p style={{ font: 'var(--type-caption)', color: 'var(--critical-fg)' }}>{error}</p>
      ) : null}
    </fieldset>
  );
}
