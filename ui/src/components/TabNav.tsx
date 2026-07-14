import { useRef, type KeyboardEvent } from "react";

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
  idBase: string;
  testId?: string;
  className?: string;
};

export function tabId(idBase: string, value: string): string {
  return `${idBase}-tab-${value}`;
}

export function tabPanelId(idBase: string, value: string): string {
  return `${idBase}-tabpanel-${value}`;
}

export function tabPanelProps(idBase: string, value: string) {
  return {
    id: tabPanelId(idBase, value),
    role: "tabpanel",
    "aria-labelledby": tabId(idBase, value),
  } as const;
}

export function TabNav<T extends string>(props: TabNavProps<T>) {
  const { value, onChange, options, ariaLabel, idBase, testId, className } = props;
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const selectedIndex = Math.max(
    0,
    options.findIndex((option) => option.id === value),
  );

  const focusAndSelect = (index: number) => {
    const option = options[index];
    if (!option) {
      return;
    }
    onChange(option.id);
    tabRefs.current[index]?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (options.length === 0) {
      return;
    }
    let nextIndex: number | null = null;
    switch (event.key) {
      case "ArrowRight":
      case "ArrowDown":
        nextIndex = (index + 1) % options.length;
        break;
      case "ArrowLeft":
      case "ArrowUp":
        nextIndex = (index - 1 + options.length) % options.length;
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = options.length - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    focusAndSelect(nextIndex);
  };

  return (
    <section className={className ? `tabs-shell ${className}` : "tabs-shell"} role="tablist" aria-label={ariaLabel} data-testid={testId}>
      {options.map((option, index) => (
        <button
          key={option.id}
          ref={(node) => {
            tabRefs.current[index] = node;
          }}
          id={tabId(idBase, option.id)}
          type="button"
          role="tab"
          aria-selected={value === option.id}
          aria-controls={tabPanelId(idBase, option.id)}
          tabIndex={index === selectedIndex ? 0 : -1}
          className={value === option.id ? "tab is-active" : "tab"}
          onClick={() => onChange(option.id)}
          onKeyDown={(event) => handleKeyDown(event, index)}
          data-testid={option.testId}
        >
          {option.label}
        </button>
      ))}
    </section>
  );
}
