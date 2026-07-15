import type { ButtonHTMLAttributes, HTMLAttributes, ReactNode } from "react";

type Tone = "neutral" | "primary" | "success" | "warning" | "danger";
type Density = "comfortable" | "compact";

export function Button({ tone = "neutral", density = "comfortable", className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { tone?: Tone; density?: Density }) {
  return <button {...props} className={`ui-button tone-${tone} density-${density} ${className}`.trim()} />;
}

export function PageHeader({ title, purpose, state, source, action }: { title: string; purpose: string; state?: ReactNode; source?: ReactNode; action?: ReactNode }) {
  return <header className="ui-page-header"><div><h1>{title}</h1><p>{purpose}</p>{source ? <div className="ui-page-source">{source}</div> : null}</div>{state ? <div className="ui-page-state">{state}</div> : null}{action ? <div className="ui-page-action">{action}</div> : null}</header>;
}

export function ContextBar({ children, tone = "neutral", ...props }: HTMLAttributes<HTMLElement> & { tone?: Tone }) {
  return <aside {...props} className={`ui-context-bar tone-${tone}`}>{children}</aside>;
}

export function RecoveryPanel({ title, children, tone = "warning", action }: { title: string; children: ReactNode; tone?: Tone; action?: ReactNode }) {
  return <section className={`ui-recovery-panel tone-${tone}`}><div><strong>{title}</strong>{children}</div>{action}</section>;
}

export function MetricGrid({ items, density = "comfortable" }: { items: Array<{ label: string; value: ReactNode }>; density?: Density }) {
  return <dl className={`ui-metric-grid density-${density}`}>{items.map((item) => <div key={item.label}><dt>{item.label}</dt><dd>{item.value}</dd></div>)}</dl>;
}

export function DefinitionList({ items, density = "comfortable" }: { items: Array<{ label: string; value: ReactNode }>; density?: Density }) {
  return <dl className={`ui-definition-list density-${density}`}>{items.map((item) => <div key={item.label}><dt>{item.label}</dt><dd>{item.value}</dd></div>)}</dl>;
}

export function AsyncState({ state, children }: { state: "loading" | "empty" | "partial" | "stale" | "offline" | "error" | "recovered"; children: ReactNode }) {
  const urgent = state === "error" || state === "offline";
  return <div className={`ui-async-state is-${state}`} role={urgent ? "alert" : "status"}>{children}</div>;
}

export function RouteTabs<T extends string>({ label, value, items, onChange }: { label: string; value: T; items: Array<{ id: T; label: string; testId?: string }>; onChange: (value: T) => void }) {
  return <nav className="ui-route-tabs" aria-label={label}>{items.map((item) => <Button key={item.id} density="compact" data-testid={item.testId} aria-current={value === item.id ? "page" : undefined} onClick={() => onChange(item.id)}>{item.label}</Button>)}</nav>;
}
