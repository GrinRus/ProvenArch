export type TabOption<T extends string> = {
  id: T;
  label: string;
  testId?: string;
};

type TabNavProps<T extends string> = {
  value: T;
  onChange: (value: T) => void;
  options: Array<TabOption<T>>;
  testId?: string;
};

export function TabNav<T extends string>(props: TabNavProps<T>) {
  const { value, onChange, options, testId } = props;
  return (
    <section className="tabs-shell" data-testid={testId}>
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
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
