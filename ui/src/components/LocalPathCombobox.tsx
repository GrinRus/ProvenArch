import { useEffect, useId, useMemo, useRef, useState } from "react";

import type { OnboardingPathSuggestion } from "../lib/appContracts";
import { loadOnboardingPathSuggestions } from "../lib/onboardingApi";
import { AsyncStatusMessage } from "./AccessibleStatus";

type LocalPathComboboxProps = {
  id: string;
  label: string;
  kind: "workspace" | "repo";
  value: string;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
  testID?: string;
  onChange: (value: string) => void;
  onSelect?: (suggestion: OnboardingPathSuggestion) => void;
};

export function LocalPathCombobox({ id, label, kind, value, placeholder, disabled, invalid, describedBy, testID, onChange, onSelect }: LocalPathComboboxProps) {
  const generatedID = useId();
  const listboxID = `${id || generatedID}-suggestions`;
  const helperID = `${id || generatedID}-helper`;
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<OnboardingPathSuggestion[]>([]);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  const requestSeq = useRef(0);
  const blurTimer = useRef<number | null>(null);
  const activeOptionID = open && activeIndex !== null ? pathOptionID(listboxID, activeIndex) : undefined;

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
    if (!open || items.length === 0) {
      setActiveIndex(null);
      return;
    }
    setActiveIndex((current) => {
      if (current === null) {
        return current;
      }
      return Math.min(current, items.length - 1);
    });
  }, [items.length, open]);

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
  const describedByIDs = [describedBy, helperText ? helperID : ""].filter(Boolean).join(" ") || undefined;

  const selectSuggestion = (item: OnboardingPathSuggestion) => {
    if (blurTimer.current !== null) {
      window.clearTimeout(blurTimer.current);
      blurTimer.current = null;
    }
    onChange(item.path);
    onSelect?.(item);
    setOpen(false);
    setActiveIndex(null);
  };

  const moveActiveOption = (direction: 1 | -1) => {
    setOpen(true);
    if (items.length === 0) {
      setActiveIndex(null);
      return;
    }
    setActiveIndex((current) => {
      if (current === null) {
        return direction === 1 ? 0 : items.length - 1;
      }
      return (current + direction + items.length) % items.length;
    });
  };

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
        aria-activedescendant={activeOptionID}
        aria-invalid={invalid || undefined}
        aria-describedby={describedByIDs}
        onBlur={() => {
          if (blurTimer.current !== null) {
            window.clearTimeout(blurTimer.current);
          }
          blurTimer.current = window.setTimeout(() => {
            setOpen(false);
            setActiveIndex(null);
          }, 100);
        }}
        onChange={(event) => {
          onChange(event.target.value);
          setOpen(true);
          setActiveIndex(null);
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown") {
            event.preventDefault();
            moveActiveOption(1);
            return;
          }
          if (event.key === "ArrowUp") {
            event.preventDefault();
            moveActiveOption(-1);
            return;
          }
          if (event.key === "Enter" && open && activeIndex !== null && items[activeIndex]) {
            event.preventDefault();
            selectSuggestion(items[activeIndex]);
            return;
          }
          if (event.key === "Escape" && open) {
            event.preventDefault();
            setOpen(false);
            setActiveIndex(null);
          }
        }}
      />
      {open ? (
        <div className="path-combobox-popover" id={listboxID} role="listbox" aria-label={`${label} suggestions`}>
          {items.map((item, index) => {
            const isActive = index === activeIndex;
            const isSelectedValue = activeIndex === null && value === item.path;
            return (
              <button
                type="button"
                className={isActive ? "path-combobox-option is-active" : "path-combobox-option"}
                id={pathOptionID(listboxID, index)}
                key={`${item.source}-${item.path}`}
                role="option"
                aria-selected={isActive || isSelectedValue}
                onFocus={() => setActiveIndex(index)}
                onMouseEnter={() => setActiveIndex(index)}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => selectSuggestion(item)}
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
            );
          })}
          {helperText ? (
            <AsyncStatusMessage id={helperID} tone={status === "error" ? "error" : status === "loading" ? "progress" : "info"} className={status === "error" ? "path-combobox-helper is-error" : "path-combobox-helper"}>
              {helperText}
            </AsyncStatusMessage>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function pathOptionID(listboxID: string, index: number): string {
  return `${listboxID}-option-${index}`;
}
