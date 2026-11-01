# 1.0.0 test report

[中文](test-results-1.0.md) · English · [日本語](test-results-1.0.ja.md)

Date: 2026-11-01. Platform: isolated containers inside a Rocky Linux 10.2 x86_64 VM.

## Results

**All 59 new-feature integration checks passed.** Exit codes are in [test-results-1.0.tsv](test-results-1.0.tsv). Additional checks passed: Linux unit tests/vet, import and installation in a new container with networking disabled, and final-build language, mixed-profile and venv checks.

| Area | Result | Evidence |
|---|---|---|
| Existing environments | Passed | `--yes` alone leaves the RPM inventory unchanged |
| Explicit updates | Passed | `--update` installs and checks Python, Java, Go, Node and build tools |
| Health checks | Passed | Before/after paths and versions; actual venv detected |
| Isolated environment | Passed | An activated venv keeps system packages unchanged even with update requested |
| Custom runtime | Passed | DNF update rejected for a custom Go path |
| npm configuration | Passed | Effective registry, unrelated settings/credentials preserved, edited restore refused, byte-exact restore |
| pip configuration | Passed | pip reads the selected index-url |
| Maven configuration | Passed | Offline Maven validate accepts settings.xml |
| Go configuration | Passed | Effective GOPROXY; GOSUMDB unchanged |
| Source connectivity | Passed | Metadata checks for pip/npm/Maven official endpoints and goproxy.cn |
| JSON profiles | Passed | Export, validation, application, no overwrite, differently cased policy keys rejected |
| Profile venv | Passed | Environment created; pip source targets its config while global user pip settings stay unchanged |
| Bundles | Passed | Export, preview, cancellation, import, existing-target protection and explicit install command |
| Corrupt archive | Passed | Modified archive rejected without publishing a target |
| Offline installation | Passed | Fresh network-disabled container imports bundle, installs pip locally and creates venv |
| Final build | Passed | Linux unit/vet, language builds, mixed system-install/user-source profile |

The first inventory-comparison command used `cmp`, which was absent from the minimal image and returned 127. Original logs were retained. Python then compared the same two saved inventories successfully; the summary records that recheck as passed. The script now uses Python.

## Safety and resource regressions

Unit coverage includes duplicate/case-variant JSON keys, unknown fields and nesting; self-closing Maven XML; pip colon options and commented sections; source-backup metadata corruption/manual edits; archive traversal, links, duplicates, hash failures, platform mismatches, appended archives and untrusted-signature branches.

A 32 MiB RPM streaming regression measures export, inspection and import separately; each allocates under 16 MiB in that sample. This is not a universal fixed-memory guarantee.

A real subprocess regression excludes command-log prefixes from captured output so genuine venvs are not misclassified. Independent review findings were fixed and covered by regressions.

## Limits

- Windows Go tests/vet and Linux amd64/arm64 plus Windows amd64 builds passed. Linux ran the same suite using Go 1.26.10.
- Full installation workflows ran first on the candidate build; final-build checks repeated unit tests, languages, mixed profiles and isolation policy.
- Host package inventory and repository hashes were unchanged, and the existing service container stayed healthy. Host `/etc`, RPM database and Docker socket were not mounted.
- Runtime coverage is Rocky 10.2 x86_64 only. Rocky 8/9 and ARM64, power loss and full-disk conditions were not tested.
- Real trusted RPM signatures passed. Untrusted-signature rejection uses test runners; no untrusted keys were imported.
- Raw results remain under `~/rockdev-v1-tests/` on the VM. The [0.1 report](test-results.en.md) retains its original date.
