# Runtime Dossier: users-service

- domain_id: `users-service`
- shard_id: `users-service`
- agent_role: `shard-analyst`
- repo_scopes: `users-service`
- summary: Extracted HTTP routes, detected a dependency on users-service, and flagged missing CI/CD evidence.
- citations: 3

## Coverage

- observed: `client code`, `http routes`, `service`
- missing: `gitlab-ci pipeline definition`, `k8s manifests`, `openapi spec`
- notes: No .gitlab-ci.yml found in analysed checkout., No helm/ or k8s/ directories found in repo.

## Entities

- `svc.payments` (service)

## Findings

- `finding.missing-owner.svc.payments`: Missing owner for service

## Questions

- `q.cicd.svc.payments`: Where is the CI/CD pipeline for Payments Service defined? (.gitlab-ci.yml, include, shared template, or external project)
- `q.owner.svc.payments`: Who owns Payments Service? (team name or CODEOWNERS mapping)

## Citation IDs

- `cite.edge-edge-svc-payments-calls-svc-users-1`
- `cite.entity-svc-payments-1`
- `cite.finding-finding-missing-owner-svc-payments`
