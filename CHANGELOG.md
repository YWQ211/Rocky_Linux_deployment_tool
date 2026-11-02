# Changes

## 1.0.1 — 2026-11-02

- Recognize Rocky 8 system Python when standard entrypoints resolve to `/usr/libexec/platform-python3.6`; continue rejecting custom runtimes.
- Verify effective pip sources with both pip 9 and newer versions. Integration runs now fail on unexpected exit codes or failed cases.
- Build versioned artifacts under `dist/1.0.1`, preserving earlier releases.

## 1.0.0 — 2026-11-01

- Inspect development environments before installation. Existing environments stay unchanged unless updating is explicitly selected. `--yes` alone does not authorize upgrades.
- Create isolated Python environments with `--venv`.
- Configure and restore user-scoped pip, npm, Maven and Go sources, preserving unrelated settings and credentials.
- Validate, export and apply strict JSON deployment profiles.
- Export and import offline RPM repositories with streaming validation, platform checks, SHA256 and trusted signatures.
- Keep Chinese, English and Japanese interfaces and documentation.

## 0.1.0 — 2026-10-30

- System checks, Rocky repository mirrors, local RPM repositories and development presets.
- Interactive menu, command mode, previews, backups and logs.
