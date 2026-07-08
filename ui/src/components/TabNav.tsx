export type TabOption<T extends string> = {
  id: T;
  label: string;
  testId?: string;
};

type TabNavProps<T extends string> = {
  value: T;
  onChange: (value: T) => void;
  options: Array<TabOption<T>>;
  ariaLabel: string;
  testId?: string;
  className?: string;
};

export function TabNav<T extends string>(props: TabNavProps<T>) {
  const { value, onChange, options, ariaLabel, testId, className } = props;
  return (
    <section className={className ? `tabs-shell ${className}` : "tabs-shell"} role="tablist" aria-label={ariaLabel} data-testid={testId}>
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          role="tab"
          aria-selected={value === option.id}
          className={value === option.id ? "tab is-active" : "tab"}
          onClick={() => onChange(option.id)}
          data-testid={option.testId}
        >
          {option.label}
        </button>
      ))}
    </section>
  );
}
