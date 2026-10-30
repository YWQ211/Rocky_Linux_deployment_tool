# Rockdev 0.1 design

Go standard-library CLI for Rocky Linux: doctor, mirror list/check/use/restore, repo create, env list/install, interactive menu. User approved proceeding with Go on 2026-10-30. Local repo is an on-demand RPM repository, not a full remote mirror.

System detection reads /etc/os-release and maps amd64/arm64 to RPM architecture names. Mutations require Linux, Rocky 8/9/10, dnf and root; VM support remains unverified. Dry run prints actions without writes or subprocess execution.

Mirrors are named HTTPS endpoints: official, Aliyun, TUNA and USTC. Probe BaseOS/AppStream repomd.xml for the current major and architecture, validate XML, and show latency. Switching rewrites only existing baseurl/mirrorlist/metalink directives inside stock BaseOS/AppStream/extras sections, preserves GPG configuration, backs up original files and restores them if refresh fails. Explicit restore accepts a generated backup identifier.

Local repo creation downloads requested RPMs and all dependencies into an isolated staging directory, verifies RPM signatures, generates metadata with createrepo_c, then publishes to a new directory. It writes a disabled-by-default file:// .repo example for explicit use; no global repo enablement and no HTTP service.

The generated subset repo has module_hotfixes=1 to allow selected modular RPMs without full module metadata. DNF install/download forces both main and wildcard repository gpgcheck=1. Commands use LC_ALL=C for stable RPM signature parsing. Interactive input waits honor context cancellation; kernel process locks serialize mutating runs.

Environment presets install distribution packages for Python/pip, Java JDK/Maven, Go, Node/npm and build tools. DNF resolves versions from configured repos; no promise of latest releases or multi-version management. Python does not replace /usr/bin/python. Verify installed tool versions and print per-step failures.

Commands run directly with argument arrays, never through a shell. Context cancellation propagates. Actions stop at first failure and persist logs; interrupted package operations are not automatically reversed. Backup restore is only for managed repo configuration. Tests use temporary files, fake runners and local HTTP servers.
