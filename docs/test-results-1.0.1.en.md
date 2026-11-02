# 1.0.1 retest results

[中文](test-results-1.0.1.md) · English · [日本語](test-results-1.0.1.ja.md)

Date: 2026-11-02. Version: 1.0.1. Linux amd64 SHA256:
`24cc9db6ff00944aa0986e7564f5e6841315d204021f7513de64617d19a887cb`.

| Environment | Integration cases | Compile and runtime checks | Offline import, install and venv |
|---|---|---|---|
| Rocky 8.10 x86_64 | 59/59 passed | 10/10 passed | Passed |
| Rocky 9.8 x86_64 | 59/59 passed | 10/10 passed | Passed |

Cases cover keeping existing environments, explicit batch updates, real venv detection and protection, effective package sources and restoration, JSON profiles, RPM repositories, offline bundles and corrupt archive rejection. Runtime checks execute Python, Java, Go, Node and a compiled C program. Both systems also reject an explicit update through a custom Python wrapper.

Rocky 8 originally passed 52/59 with 1.0.0; the patched run passed 59/59. Original failures are retained. Seventeen path classification regressions cover the fix: four standard Rocky 8 entrypoints failed before the change and passed afterward. Effective pip 9 settings are read through its install command parser; pip was not upgraded to bypass compatibility differences. Cases require expected exit codes, and any failed case makes the script exit nonzero.

Unit tests and `go vet` passed on Windows with Go 1.27.1 and Linux with Go 1.26.10. Builds passed for Linux amd64/arm64, Chinese/English/Japanese defaults and Windows amd64. Independent review found no blocking issues.

Tests ran sequentially in containers on an existing Rocky 10.2 VM. Full containers were limited to one CPU and 1 GiB RAM; offline containers to 768 MiB. Containers were removed afterward and logs retained. Host RPM and repository inventories were unchanged. Retired test containers and transfer files were reclaimed; the existing scheduler remained healthy without a restart.

See the [TSV](test-results-1.0.1.tsv). Raw logs remain in `~/rockdev-fix-tests/` on the test VM. Runtime evidence is limited to Rocky 8.10/9.8 x86_64. ARM64 was cross-built; full Rocky 10.2 results remain the [historical 1.0.0 run](test-results-1.0.en.md). Containers do not validate separate kernels, complete systemd/SELinux behavior or whole-system upgrades.
