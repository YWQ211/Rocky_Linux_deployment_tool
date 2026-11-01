# 0.1.0 test report

[中文](test-results.md) · English · [日本語](test-results.ja.md)

Test date: **2026-10-30**. This translation does not represent a new test run.

## Scope

A separate Rocky installation directory was created inside a Rocky Linux 10.2 x86_64 VM and imported as a Docker image. Separate containers covered installation, error handling and offline use. The host's `/etc`, RPM database and Docker socket were not mounted into test containers. Host package inventory and repository hashes were unchanged after testing; the existing service container remained running.

**71 checks: 69 passed, two mirrors unavailable.** Exact exit codes are in [test-results.tsv](test-results.tsv). The count includes one unit-test invocation and one vet invocation; the unit-test suite contains 21 test functions.

## Results

| Area | Result | Coverage |
|---|---|---|
| System checks | Passed | OS, architecture, permissions, DNF; changes rejected on a non-Rocky system |
| Chinese, English, Japanese | Passed | Help, menu, system checks, previews, standalone English/Japanese builds |
| Input and permissions | Passed | Invalid commands, environments, languages, RPM options; non-root install rejected |
| Mirror selection | Passed | Official and Aliyun available; auto selected Aliyun |
| Mirror changes | Passed | Configuration backup, cache refresh, repeated selection creates no new backup |
| Restore | Passed | Restore succeeds; edited files and invalid backup IDs are rejected |
| Refresh failure | Passed | Simulated DNF exit 42; original configuration restored with matching hashes |
| Python | Passed | pip installation, venv creation and script execution |
| Java / Maven | Passed | Version checks; compile and run a Java program |
| Go | Passed | Install, compile and run a program |
| Node.js / npm | Passed | Run a script; install and call is-number from npm registry |
| Build tools | Passed | Git, make, C and C++ compilation |
| Repeated installation | Passed | Reinstall all five presets and complete command checks |
| Local RPM repositories | Passed | Python and Node package sets, signature checks and metadata generation |
| Repository errors | Passed | Existing target, missing parent, unavailable package; failed targets are not published |
| Offline Python | Passed | Fresh container with networking disabled; install Python/pip and create venv using only the local repo |
| Offline Node | Passed | Another fresh container with networking disabled; install and run Node/npm using only the local repo |
| Process lock | Passed | A second modification is rejected while the lock is held |
| SIGTERM | Passed | Exit while waiting for menu input without sending another newline |
| Japanese error log | Passed | Final build writes the translated failure summary and preserves the error chain |

## Mirror availability

| Mirror | Observed result |
|---|---|
| Rocky official | Metadata check passed |
| Aliyun | Metadata, package installation and repository downloads passed |
| Tsinghua TUNA | Rocky 10 BaseOS metadata URL returned HTTP 404 |
| USTC | Rocky 10 BaseOS metadata URL returned HTTP 403 |

Unavailable mirrors remain selectable, but failed checks explain the cause and exclude them from automatic selection. Availability depends on the mirror and network at the time of use.

Docker initially injected an unusable proxy. Network checks were rerun after clearing proxy variables in test containers. The first SIGTERM test closed stdin at the same time as signalling the process, creating a race; the summary uses the corrected test that keeps stdin open. Original logs were retained.

## Installed versions

| Tool | Version |
|---|---|
| Python | 3.12.15 |
| pip | 23.3.2 |
| OpenJDK | 21.0.12.1 |
| Maven | 3.9.9 |
| Go | 1.26.10 |
| Node.js | 22.23.2 |
| npm | 10.9.8 |
| GCC / G++ | 14.3.1 |

These came from the enabled Rocky repositories during testing; Rockdev does not pin them.

## Builds and limits

- Windows: Go 1.27.1; 21 test functions and `go vet` passed.
- Linux: Rocky's Go 1.26.10; the same unit tests and `go vet` passed.
- Linux amd64/arm64, Windows amd64 and English/Japanese default builds compiled. The English and Japanese amd64 builds were executed.
- Installation and repository workflows ran on the language-feature build. The final build additionally checked language defaults, failure logs, permissions, signals and Linux unit tests.
- Rocky 8, Rocky 9 and ARM64 were not runtime-tested. Power loss, full disks and full VM boot/recovery were not simulated.
- Real malicious RPM rejection was not tested. Unsigned/untrusted signature branches have unit coverage; downloaded packages underwent real signature checks.

Scripts: `scripts/integration.sh`, `scripts/integration-edge.sh`, `scripts/integration-final.sh`. Original logs and stopped test containers were retained on the test VM under `~/rockdev-isolated-tests/`.
