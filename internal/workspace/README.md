# internal/workspace/

Workspace parsing + safe file IO.

Current baseline responsibilities:
- validate absolute workspace root path
- require `workspace.yaml` in workspace root
- parse `workspace.yaml`
- run semantic checks: duplicate names, version, `path|git_url` invariants
- apply defaults (`docs.imports_path`)
- protect filesystem scope to workspace root (`Resolve`, `ReadFile`, `WriteFile`)
- create fixed MVP workspace layout (`EnsureLayout`)
