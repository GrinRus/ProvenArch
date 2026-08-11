# Task, change and publication linkage

- **ADR ID:** ADR-20260811-task-change-publication-linkage
- **Status:** accepted
- **Date:** 2026-08-11
- **Owners:** ACP maintainers

## Context

Validator-approved pipeline output is promoted automatically into the current architecture
workspace. Git Commit and Create proposal branch are later explicit full-workspace mutations. A
single boolean `published` on Task would conflate those events and could falsely claim that one Task
exclusively owns a commit containing other Task results, manual edits or unrelated untracked files.

## Decision

- Automatic promotion, semantic change review and Git publication remain separate authorities.
- Task Outcome may link an Attempt to its exact run snapshot and promoted semantic comparison, but
  it must use `Workspace has unpublished changes` until a successful Git mutation creates an
  authoritative publication association.
- A publication association is server-authored only after a successful Git mutation. It records the
  initiating Task/Attempt/run identities when present, action (`commit` or `proposal_branch`), branch,
  base/head identity, exact full-workspace inventory fingerprint and resulting commit/branch
  identity.
- The association describes the user action and exact inventory; it does not claim that every path
  in a full-workspace commit belongs exclusively to the initiating Task. UI counters continue to
  distinguish semantic changes from Git file inventory.
- Dirty state, branch names, latest commit recency or a clean working tree are never sufficient to
  infer Task publication. Existing server-authored Git truth and stale-confirmation protections
  remain authoritative.

## Alternatives considered

- Persist `Task.published=true`: rejected because it loses commit and inventory identity and is
  incorrect for mixed full-workspace commits.
- Infer publication from a clean worktree or latest commit: rejected because neither proves which
  Task/run/inventory was confirmed.
- Change publication to selected-file commit: rejected because the accepted MVP contract publishes
  the authoritative full workspace.

## Consequences

- Task and Changes APIs need explicit publication linkage fields and unknown/unavailable states.
- Git mutation results must be correlated with the confirmation fingerprint and selected
  Task/Attempt context without weakening the existing full-inventory precondition.
- One publication can reference a selected Task while honestly reporting mixed workspace content.

## Links

- Related epics: 23E, 23L
- Related ADR: `ADR-20260726-changes-git-truth.md`
- Related docs: `docs/spec/TASK_SPEC.md`, `docs/spec/API_SPEC.md`,
  `docs/UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`
