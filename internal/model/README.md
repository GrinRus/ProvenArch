# internal/model

Пакет реализует entity-per-file model store в workspace:
- upsert/remove entity и edge;
- alias/canonical ID resolution;
- deterministic collision policy (`.repo-<repo-slug>`);
- semantic guardrails (например, owner linkage).

Model layer является deterministic baseline surface и защищён golden/scenario тестами.
