# ADR-20260726: Responsive safe areas and mobile card semantics

## Status

Accepted.

## Context

Tablet navigation visually hid a focusable collapse control, mobile fixed navigation and fullscreen
sheets ignored device safe areas, and read-only Run/Knowledge tables relied on wide scrolling.

## Decision

- Controls that are unavailable at a breakpoint use `display: none`; text-only labels may remain
  visually hidden.
- Mobile header, fixed primary navigation, content clearance and fullscreen dialogs include
  `env(safe-area-inset-*)`.
- Primary/utility touch targets are at least 44 px on phone layouts.
- Non-comparison Run history and Knowledge tables use stable row keys plus `data-label` card
  semantics below 680 px. Comparison/dense diagnostic tables keep explicit scroll behavior.
- Open Context drawers react to breakpoint/orientation changes and use the shared modal focus trap
  below 1280 px.

## Consequences

Keyboard focus cannot land on invisible navigation controls, mobile chrome remains operable around
notches/home indicators and long paths reflow inside deterministic cards.
