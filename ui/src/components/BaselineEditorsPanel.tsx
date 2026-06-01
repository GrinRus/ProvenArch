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
    <section className="panel" data-testid="charter-artifact-editor">
      <h2>Baseline: Editors</h2>
      <p className="hint">
        Editable baseline files from `charter/*` and `skills/*`. Live headless runtime customization consumes prompt packs for step0/step1/step3/step4;
        step2 uses enforced as-is policy without an editable prompt pack, and `skills/*/prompts/*.md` stay reference-only seeded assets.
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
        <option value="">Select an artifact to preview or edit</option>
        {baselineEditorArtifacts.map((artifact) => (
          <option key={artifact.path} value={artifact.path}>
            {artifact.label}
          </option>
        ))}
      </select>
      <label htmlFor="baselineArtifactEditor">{selectedEditorPath || "Artifact content"}</label>
      <textarea
        id="baselineArtifactEditor"
        value={selectedEditorContent}
        onChange={(event) => onEditorContentChange(event.target.value)}
        rows={10}
        placeholder="Select an artifact to preview or edit."
        disabled={!selectedEditorPath}
      />
      <button type="button" onClick={onSave} disabled={busy || !selectedEditorPath}>
        Save selected baseline artifact
      </button>
      {editorStatus ? <p className="status ok">{editorStatus}</p> : null}
    </section>
  );
}
