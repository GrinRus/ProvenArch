import { useEffect, useRef, useState, type ReactNode } from "react";
import { ModalDialog } from "./ModalDialog";

export function ContextDrawer({ open, title, description, children, onClose }: { open: boolean; title: string; description: string; children: ReactNode; onClose: () => void }) {
  const [persistent, setPersistent] = useState(() => window.matchMedia?.("(min-width: 1280px)").matches ?? false);
  const closeRef = useRef<HTMLButtonElement | null>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);
  const wasOpenPersistentRef = useRef(false);
  useEffect(() => {
    const media = window.matchMedia?.("(min-width: 1280px)");
    if (!media) return;
    const update = () => setPersistent(media.matches);
    update();
    media.addEventListener?.("change", update);
    return () => media.removeEventListener?.("change", update);
  }, []);
  useEffect(() => {
    if (!persistent) {
      wasOpenPersistentRef.current = false;
      return;
    }
    if (!open) {
      if (wasOpenPersistentRef.current && returnFocusRef.current && document.contains(returnFocusRef.current)) returnFocusRef.current.focus();
      wasOpenPersistentRef.current = false;
      return;
    }
    if (!wasOpenPersistentRef.current) {
      returnFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      wasOpenPersistentRef.current = true;
      closeRef.current?.focus();
    }
  }, [open, persistent]);
  if (!open) return null;
  if (!persistent) return <ModalDialog id="workspace-details-drawer" open title={title} description={description} onCancel={onClose}><div className="context-drawer-content">{children}</div></ModalDialog>;
  return <aside id="workspace-details-drawer" className="context-drawer persistent" aria-label={title}><header><div><h2>{title}</h2><p>{description}</p></div><button ref={closeRef} type="button" onClick={onClose}>Close</button></header>{children}</aside>;
}
