import { useEffect, useId, useMemo, useRef, useState } from "react";

import type { OnboardingPathSuggestion } from "../lib/appContracts";
import { loadOnboardingPathSuggestions } from "../lib/onboardingApi";

type LocalPathComboboxProps = {
  id: string;
  label: string;
  kind: "workspace" | "repo";
  value: string;
  placeholder?: string;
  disabled?: boolean;
  testID?: string;
  onChange: (value: string) => void;
  onSelect?: (suggestion: OnboardingPathSuggestion) => void;
};

export function LocalPathCombobox({ id, label, kind, value, placeholder, disabled, testID, onChange, onSelect }: LocalPathComboboxProps) {
  const generatedID = useId();
  const listboxID = `${id || generatedID}-suggestions`;
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<OnboardingPathSuggestion[]>([]);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  const requestSeq = useRef(0);
  const blurTimer = useRef<number | null>(null);

  useEffect(() => {
    if (!open || disabled) {
      return;
    }
    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    let cancelled = false;
    setStatus("loading");
    const timer = window.setTimeout(() => {
      loadOnboardingPathSuggestions(kind, value)
        .then((payload) => {
          if (cancelled || requestSeq.current !== seq) {
            return;
          }
          setItems(payload.items ?? []);
          setStatus("idle");
        })
        .catch(() => {
          if (cancelled || requestSeq.current !== seq) {
            return;
          }
          setItems([]);
          setStatus("error");
        });
    }, 120);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [disabled, kind, open, value]);

  useEffect(() => {
    return () => {
      if (blurTimer.current !== null) {
        window.clearTimeout(blurTimer.current);
      }
    };
  }, []);

  const helperText = useMemo(() => {
    if (status === "loading") {
      return "Loading local path suggestions...";
    }
    if (status === "error") {
      return "Suggestions unavailable. Typed path still works.";
    }
    if (open && items.length === 0) {
      return "No local suggestions for this query.";
    }
    return "";
  }, [items.length, open, status]);

  return (
    <div className="field local-path-combobox" data-testid={testID}>
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        autoComplete="off"
        role="combobox"
        aria-autocomplete="list"
        aria-controls={listboxID}
        aria-expanded={open}
        aria-haspopup="listbox"
        onBlur={() => {
          if (blurTimer.current !== null) {
            window.clearTimeout(blurTimer.current);
          }
          blurTimer.current = window.setTimeout(() => setOpen(false), 100);
        }}
        onChange={(event) => {
          onChange(event.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
      />
      {open ? (
        <div className="path-combobox-popover" id={listboxID} role="listbox" aria-label={`${label} suggestions`}>
          {items.map((item) => (
            <button
              type="button"
              className="path-combobox-option"
              key={`${item.source}-${item.path}`}
              role="option"
              aria-selected={value === item.path}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => {
                onChange(item.path);
                onSelect?.(item);
                setOpen(false);
              }}
            >
              <span>
                <strong>{item.label}</strong>
                <code>{item.path}</code>
              </span>
              <span className={item.exists ? "path-combobox-meta" : "path-combobox-meta is-missing"}>
                {item.kind.replace("_", " ")} · {item.source}
                {item.exists ? "" : " · missing"}
              </span>
            </button>
          ))}
          {helperText ? <p className={status === "error" ? "path-combobox-helper is-error" : "path-combobox-helper"}>{helperText}</p> : null}
        </div>
      ) : null}
    </div>
  );
}
