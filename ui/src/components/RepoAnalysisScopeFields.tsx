import { analysisScopeSummary } from "../lib/analysisScope";

export type RepoAnalysisScopeFieldsProps = {
  repoId: string;
  include: string;
  exclude: string;
  onIncludeChange: (value: string) => void;
  onExcludeChange: (value: string) => void;
};

export function RepoAnalysisScopeFields({ repoId, include, exclude, onIncludeChange, onExcludeChange }: RepoAnalysisScopeFieldsProps) {
  return (
    <details className="analysis-scope-fields">
      <summary>
        <span>Analysis scope</span>
        <span className="analysis-scope-summary">{analysisScopeSummary(include, exclude)}</span>
      </summary>
      <div className="analysis-scope-grid">
        <div className="field">
          <label htmlFor={`analysisInclude-${repoId}`}>Include globs</label>
          <textarea
            id={`analysisInclude-${repoId}`}
            value={include}
            onChange={(event) => onIncludeChange(event.target.value)}
            rows={3}
            placeholder={"services/**\ncmd/**"}
          />
        </div>
        <div className="field">
          <label htmlFor={`analysisExclude-${repoId}`}>Exclude globs</label>
          <textarea
            id={`analysisExclude-${repoId}`}
            value={exclude}
            onChange={(event) => onExcludeChange(event.target.value)}
            rows={3}
            placeholder={"**/vendor/**\n**/node_modules/**"}
          />
        </div>
      </div>
    </details>
  );
}
