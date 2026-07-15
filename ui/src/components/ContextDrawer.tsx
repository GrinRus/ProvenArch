import { useEffect, useState, type ReactNode } from "react";
import { ModalDialog } from "./ModalDialog";

export function ContextDrawer({ open, title, description, children, onClose }: { open: boolean; title: string; description: string; children: ReactNode; onClose: () => void }) {
  const [persistent, setPersistent] = useState(() => window.matchMedia?.("(min-width: 1280px)").matches ?? false);
  useEffect(() => {
    const media = window.matchMedia?.("(min-width: 1280px)");
    if (!media) return;
    const update = () => setPersistent(media.matches);
    update();
    media.addEventListener?.("change", update);
    return () => media.removeEventListener?.("change", update);
  }, []);
  if (!open) return null;
  if (!persistent) return <ModalDialog open title={title} description={description} onCancel={onClose}><div className="context-drawer-content">{children}</div></ModalDialog>;
  return <aside className="context-drawer persistent" aria-label={title}><header><div><h2>{title}</h2><p>{description}</p></div><button type="button" onClick={onClose}>Close</button></header>{children}</aside>;
}
