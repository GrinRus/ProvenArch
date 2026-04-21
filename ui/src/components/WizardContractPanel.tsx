type WizardContractPanelProps = {
  busy: boolean;
  wizardProjectName: string;
  wizardScope: string;
  wizardNfr: string;
  wizardRules: string;
  wizardStatus: string;
  onProjectNameChange: (value: string) => void;
  onScopeChange: (value: string) => void;
  onNfrChange: (value: string) => void;
  onRulesChange: (value: string) => void;
  onSave: () => void;
};

export function WizardContractPanel({
  busy,
  wizardProjectName,
  wizardScope,
  wizardNfr,
  wizardRules,
  wizardStatus,
  onProjectNameChange,
  onScopeChange,
  onNfrChange,
  onRulesChange,
  onSave,
}: WizardContractPanelProps) {
  return (
    <section className="panel">
      <h2>Setup: Step 0 Wizard Contract</h2>
      <p className="hint">Structured contract persisted as `charter/wizard/step0-contract.json`.</p>

      <label htmlFor="wizardProjectName">Project name</label>
      <input id="wizardProjectName" value={wizardProjectName} onChange={(event) => onProjectNameChange(event.target.value)} />

      <label htmlFor="wizardScope">Scope</label>
      <textarea id="wizardScope" value={wizardScope} onChange={(event) => onScopeChange(event.target.value)} rows={3} />

      <label htmlFor="wizardNfr">NFR priorities (comma/newline)</label>
      <textarea id="wizardNfr" value={wizardNfr} onChange={(event) => onNfrChange(event.target.value)} rows={3} />

      <label htmlFor="wizardRules">Rules (comma/newline)</label>
      <textarea id="wizardRules" value={wizardRules} onChange={(event) => onRulesChange(event.target.value)} rows={3} />

      <button type="button" onClick={onSave} disabled={busy}>
        Save Step 0 wizard contract
      </button>

      {wizardStatus ? <p className="status ok">{wizardStatus}</p> : null}
    </section>
  );
}
