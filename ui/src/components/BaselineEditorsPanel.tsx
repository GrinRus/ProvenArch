import type { Diagnostic, EditableArtifactOption } from "../lib/appContracts";

type BaselineEditorsPanelProps = {
  busy: boolean;
  baselineBundleWarnings: Diagnostic[];
  baselineEditorArtifacts: EditableArtifactOption[];
  selectedEditorPath: string;
  selectedEditorContent: string;
  editorStatus: string;
  onEditorSelectionChange: (path: string) => void;
  onEditorContentChange: (value: string) => void;
  onSave: () => void;
};

export function BaselineEditorsPanel({
  busy,
  baselineBundleWarnings,
  baselineEditorArtifacts,
  selectedEditorPath,
  selectedEditorContent,
  editorStatus,
  onEditorSelectionChange,
  onEditorContentChange,
  onSave,
}: BaselineEditorsPanelProps) {
  return (
    <section className="panel">
      <h2>Baseline: Editors</h2>
      <p className="hint">
        Editable baseline files from `charter/*` and `skills/*`. Live headless runtime customization consumes prompt packs for `collect`/`findings`;
        `skills/*/prompts/*.md` stay editable here as reference-only seeded assets.
      </p>
      {baselineBundleWarnings.map((diagnostic, index) => (
        <p key={`${diagnostic.code}-${diagnostic.message}-${index}`} className={diagnostic.level === "error" ? "status err" : "status warn"}>
          {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
        </p>
      ))}
      <label htmlFor="baselineArtifactSelect">Select artifact</label>
      <select
        id="baselineArtifactSelect"
        value={selectedEditorPath}
        onChange={(event) => onEditorSelectionChange(event.target.value)}
        disabled={baselineEditorArtifacts.length === 0}
      >
        {baselineEditorArtifacts.map((artifact) => (
          <option key={artifact.path} value={artifact.path}>
            {artifact.label}
          </option>
        ))}
      </select>
      <label htmlFor="baselineArtifactEditor">{selectedEditorPath}</label>
      <textarea
        id="baselineArtifactEditor"
        value={selectedEditorContent}
        onChange={(event) => onEditorContentChange(event.target.value)}
        rows={10}
        disabled={!selectedEditorPath}
      />
      <button type="button" onClick={onSave} disabled={busy || !selectedEditorPath}>
        Save selected baseline artifact
      </button>
      {editorStatus ? <p className="status ok">{editorStatus}</p> : null}
    </section>
  );
}
