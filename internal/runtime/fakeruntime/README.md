# internal/runtime/fakeruntime

Provider-neutral deterministic runtime for required CI and local smoke tests.

It writes the same artifact contracts as live adapters and then runs the shared
strict artifact validators. It does not execute or emulate any live provider
CLI.
