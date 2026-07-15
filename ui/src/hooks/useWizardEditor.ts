import { useState } from "react";

import type { WizardContract } from "../lib/appContracts";
import { splitListInput } from "../lib/runState";
import { loadArtifactText, saveEditableArtifact } from "../lib/workspaceApi";

type UseWizardEditorOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useWizardEditor({ setBusy, setError }: UseWizardEditorOptions) {
  const [wizardProjectName, setWizardProjectName] = useState("ProvenArch MVP");
  const [wizardScope, setWizardScope] = useState("payments, users, ci-cd");
  const [wizardNfr, setWizardNfr] = useState("availability, traceability");
  const [wizardRules, setWizardRules] = useState("no silent re-key, evidence-first findings");
  const [wizardStatus, setWizardStatus] = useState("");
  const [wizardContractLoaded, setWizardContractLoaded] = useState(false);
  const [wizardContractReady, setWizardContractReady] = useState(false);

  async function loadWizardContract() {
    try {
      const content = (await loadArtifactText("charter/wizard/step0-contract.json"))?.trim() ?? "";
      if (!content) {
        return;
      }
      const parsed = JSON.parse(content) as Partial<WizardContract>;
      if (typeof parsed.project_name === "string") {
        setWizardProjectName(parsed.project_name);
      }
      if (typeof parsed.scope === "string") {
        setWizardScope(parsed.scope);
      }
      setWizardContractReady(Boolean(parsed.project_name?.trim() && parsed.scope?.trim()));
      if (Array.isArray(parsed.nfr_priorities)) {
        setWizardNfr(parsed.nfr_priorities.join(", "));
      }
      if (Array.isArray(parsed.rules)) {
        setWizardRules(parsed.rules.join(", "));
      }
    } catch {
      // Wizard contract remains optional during bootstrap.
    } finally {
      setWizardContractLoaded(true);
    }
  }

  async function handleSaveStep0WizardContract() {
    setBusy(true);
    setError(null);
    setWizardStatus("");

    const projectName = wizardProjectName.trim();
    const scope = wizardScope.trim();
    if (!projectName || !scope) {
      setBusy(false);
      setError("step0 wizard contract requires project name and scope");
      return;
    }

    const payload: WizardContract = {
      version: 1,
      project_name: projectName,
      scope,
      nfr_priorities: splitListInput(wizardNfr),
      rules: splitListInput(wizardRules),
    };

    try {
      await saveEditableArtifact("charter/wizard/step0-contract.json", `${JSON.stringify(payload, null, 2)}\n`);
      setWizardStatus("Saved charter/wizard/step0-contract.json");
      setWizardContractReady(true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save step0 wizard contract");
    } finally {
      setBusy(false);
    }
  }

  return {
    wizardProjectName,
    wizardScope,
    wizardNfr,
    wizardRules,
    wizardStatus,
    wizardContractLoaded,
    wizardContractReady,
    loadWizardContract,
    setWizardProjectName,
    setWizardScope,
    setWizardNfr,
    setWizardRules,
    handleSaveStep0WizardContract,
  };
}
