import { useEffect, useId, useRef, type ReactNode } from "react";

type ModalDialogProps = {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  busy?: boolean;
  children?: ReactNode;
  onConfirm?: () => void;
  onCancel: () => void;
};

export function ModalDialog({ open, title, description, confirmLabel, busy, children, onConfirm, onCancel }: ModalDialogProps) {
  const titleID = useId();
  const descriptionID = useId();
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const cancelRef = useRef<HTMLButtonElement | null>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    returnFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    cancelRef.current?.focus();
    return () => returnFocusRef.current?.focus();
  }, [open]);

  if (!open) {
    return null;
  }

  return (
    <div className="modal-backdrop" role="presentation">
      <div
        ref={dialogRef}
        className="modal-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleID}
        aria-describedby={descriptionID}
        onKeyDown={(event) => {
          if (event.key === "Escape" && !busy) {
            event.preventDefault();
            onCancel();
            return;
          }
          if (event.key !== "Tab") {
            return;
          }
          const focusable = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>("button:not([disabled]), a[href], input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex='-1'])") ?? []);
          if (focusable.length === 0) {
            event.preventDefault();
            return;
          }
          const first = focusable[0];
          const last = focusable[focusable.length - 1];
          if (event.shiftKey && document.activeElement === first) {
            event.preventDefault();
            last.focus();
          } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
          }
        }}
      >
        <h2 id={titleID}>{title}</h2>
        <p id={descriptionID}>{description}</p>
        {children}
        <div className="modal-actions">
          <button ref={cancelRef} type="button" onClick={onCancel} disabled={busy}>Cancel</button>
          {confirmLabel ? <button type="button" onClick={onConfirm} disabled={busy}>{confirmLabel}</button> : null}
        </div>
      </div>
    </div>
  );
}
