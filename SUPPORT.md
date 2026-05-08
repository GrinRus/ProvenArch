# Support

## Where to ask

Use GitHub Issues for:
- reproducible bugs;
- install or first-run failures;
- documentation gaps;
- feature requests with clear use cases.

Use pull requests for focused fixes that follow [CONTRIBUTING.md](CONTRIBUTING.md).

## Before opening an issue

Please include:
- `acp version`;
- OS and architecture;
- install method;
- command run;
- `acp doctor` output with secrets removed;
- whether the run used `--runtime fake` or `--runtime headless`;
- relevant logs from `reports/taskruns/<run_id>/`.

## Security reports

Do not report vulnerabilities in public issues. Follow [SECURITY.md](SECURITY.md).

## MVP support boundary

The primary distribution path is GitHub Releases single-binary archives for macOS/Linux `amd64` and `arm64`.

Docker, npm package, PyPI, Maven, crates.io, hosted/SaaS mode, and security/compliance enforcement are not primary MVP support surfaces.
