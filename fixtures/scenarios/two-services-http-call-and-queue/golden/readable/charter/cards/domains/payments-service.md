# Domain: payments-service

- id: `payments-service`
- repo_scope: `payments-service`

## Derived (ACP Step1)

- domain_id: `payments-service`
- repo_scope: `payments-service`
- related_entities: `svc.payments`
- related_findings: `finding.missing-owner.svc.payments`
- open_questions: `q.cicd.svc.payments`, `q.owner.svc.payments`
- coverage_missing: gitlab-ci pipeline definition, k8s manifests, openapi spec
- evidence_refs:
  - `payments-service:cmd/payments/main.go`
