# Rockdev 1.0.1

## Documentation / 文档语言 / ドキュメント

| 简体中文 | English | 日本語 |
|---|---|---|
| [**完整中文说明（首页）**](README.md) | [**Full English guide — this page**](README.en.md) | [**日本語の説明はこちら →**](README.ja.md) |

Full guides are available in all three languages. The repository landing page is Chinese; use the links above to switch.

Documentation: [Full feature guide](docs/release-1.0.en.md) · [VM guide](docs/vm-quickstart.en.md) · [1.0.1 retest report](docs/test-results-1.0.1.en.md)

1.0.1 fixes system Python detection on Rocky 8. [Patch notes and tests](docs/release-1.0.1.en.md).

A Go CLI for Rocky Linux. It checks the system, switches package mirrors, creates a local RPM repository and installs development tools.

Regular users can run the launcher in the checkout without installing Go. Developers changing the program can use [Build](#build).

## Clone and run

On Rocky Linux 8/9/10, x86_64 or aarch64:

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
./rockdev
```

The first run needs network access, `curl` or `wget`, and `sha256sum`. The launcher downloads this checkout's fixed Release version, verifies it against `release/SHA256SUMS`, caches it and opens the menu. It does not follow latest automatically. Maintainers update the version and pins; users obtain those updates with `git pull`. Starting the launcher does not install packages with DNF.

```bash
./rockdev doctor
./rockdev --lang en
./rockdev --lang ja
sudo ./rockdev env install python go
```

Cached executables are verified on every run and can start without a network connection. Package installation still needs network access or an offline repository. User cache defaults to `~/.cache/rockdev/`, with an absolute `XDG_CACHE_HOME` supported. Root/sudo uses `/var/cache/rockdev/` instead of executing a user cache. Cache paths must be real, owned directories, not symlinks or shared directories writable by other users.

Automatic downloads require a publicly accessible repository and a published matching Release. Private Git clone credentials are not passed to the downloader; for a 404, check repository and Release visibility. Local source builds remain available.

## Run

These examples assume you obtained a compiled Linux binary for your architecture from [Releases](https://github.com/YWQ211/Rocky_Linux_deployment_tool/releases). Go is needed to build from source, but not to run the binary.

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --lang en
./rockdev-linux-amd64 --lang en doctor
```

Choose Chinese, English or Japanese with `--lang zh|en|ja`. `ROCKDEV_LANG` sets the default. Command names stay in English. DNF and other system tools keep their own output language.

The build script also produces `rockdev-linux-amd64-en` with English as its default language and a `-ja` build for Japanese. ARM64 equivalents are included.

```bash
export ROCKDEV_LANG=en
./rockdev-linux-amd64 env install python go --dry-run
```

## Official sources and regional use

Environment installation and RPM repository creation use the machine's enabled DNF repositories. Rockdev does not automatically switch them to Chinese mirrors. Existing official mirrorlist settings can remain in use. Only `mirror use` changes the system repository configuration; `--lang en` changes the interface language, not download sources.

| Package type | Official provider selector | Built-in official endpoint | Other presets |
|---|---|---|---|
| Rocky RPM | `mirror ... official` | `https://dl.rockylinux.org/pub/rocky` | `aliyun`, `tuna`, `ustc` |
| Python / pip | `source ... pip official` | `https://pypi.org/simple/` | `tuna`, `aliyun` |
| Node / npm | `source ... npm official` | `https://registry.npmjs.org/` | `npmmirror` |
| Maven | `source ... maven official` | `https://repo.maven.apache.org/maven2/` | `aliyun` |
| Go modules | `source ... go official` | `https://proxy.golang.org` | `goproxy.cn` |

To explicitly choose official sources:

```bash
rockdev mirror check official
sudo rockdev mirror use official
rockdev source use pip official
rockdev source use npm official
rockdev source use maven official
rockdev source use go official
```

`mirror` manages Rocky system RPM repositories; `source` manages the current user's development-package configuration. Changing pip's source does not change DNF.

`mirror use auto` checks metadata availability and elapsed check time among the four built-in system sources. It is not global mirror discovery, an RPM bandwidth benchmark or continuous failover. `official` selects a fixed official endpoint; it does not restore the original distribution mirrorlist. Custom URLs and country groups are not implemented. A reachable mirror is not necessarily the fastest for every region.

## Commands

Version 1.0 adds environment inspection, package sources, JSON deployment profiles and offline bundle export/import. See the [1.0 guide](docs/release-1.0.en.md) for options and limits.

```bash
rockdev env check
rockdev env install python --venv /home/your-user/project/.venv
sudo rockdev env install python java go --update
rockdev source use pip official
rockdev profile export --file dev.json python go
rockdev profile validate dev.json
rockdev profile apply dev.json --dry-run
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
```

Existing environments, including incomplete ones, are kept unless updating is selected. `--yes` does not authorize updates; use `--update` or choose explicitly. Custom runtimes use their original manager. Python venvs are separate. Source configuration belongs to the current user; sudo configures root.

| Command | Purpose |
|---|---|
| `doctor` | Show OS, architecture, root permission and DNF availability |
| `mirror list` | List Rocky, Aliyun, Tsinghua TUNA and USTC mirrors |
| `mirror check [name]` | Check repository metadata for the current system |
| `mirror use <name\|auto>` | Back up configuration, switch mirror and refresh cache |
| `mirror restore <backup-id>` | Restore a saved repository configuration |
| `repo create --dir <path> <packages...>` | Download RPMs and dependencies, verify signatures and create metadata |
| `env list` | Show development presets and commands found in PATH |
| `env install <presets...>` | Install development tools and check their commands |
| `version` | Print the version |
| `env check [presets...]` | Show paths, versions and potential conflicts |
| `source list/show/check/use/restore` | Configure and restore package sources |
| `profile validate/export/apply` | Validate, export and apply JSON profiles |
| `bundle export/import` | Export, verify and import offline RPM repositories |

Run without arguments to open the menu. System mirror changes, RPM repository creation and system package installation require root. Inspection, user source settings, new venvs and offline operations in accessible directories can run as a regular user. Profile permissions depend on its tasks. Changes show a plan and ask for confirmation; `--yes` skips confirmation but does not authorize updates. `--dry-run` makes no network requests, file writes or installations.

## Examples

```bash
sudo ./rockdev-linux-amd64 --lang en mirror use auto
sudo ./rockdev-linux-amd64 --lang en env install python java go node build
sudo mkdir -p /srv/rockdev
sudo ./rockdev-linux-amd64 --lang en repo create --dir /srv/rockdev/python python3 python3-pip
```

The repository parent must exist and the target directory must be new. The generated `rockdev-local.repo` is disabled by default. Copy it into `/etc/yum.repos.d/` and enable it explicitly for a transaction:

```bash
sudo cp /srv/rockdev/python/rockdev-local.repo /etc/yum.repos.d/
sudo dnf --disablerepo='*' --enablerepo=rockdev-local install python3 python3-pip
```

## Development presets

| Preset | Packages |
|---|---|
| `python` | Python 3, pip; venv support is checked |
| `java` | JDK 17, or JDK 21 on Rocky 10; Maven |
| `go` | Go |
| `node` | Node.js and npm |
| `build` | Git, GCC, G++, make, pkg-config |

Versions come from enabled DNF repositories. The tool does not select language versions or replace system Python. Create project environments with `python3 -m venv .venv`.

## Feature workflows

1. **System and environment inspection:** run `rockdev doctor` for OS and architecture, then `rockdev env check python java go node` for paths, versions, duplicate PATH candidates, JAVA_HOME mismatches and Python isolation. This is a deployment check; it does not repair project dependencies or change environments.
2. **System mirrors:** `mirror check` probes repositories for the current Rocky version and architecture. `mirror use` backs up and rewrites existing BaseOS/AppStream/extras sections, then refreshes the cache. A failed refresh restores the original configuration. Keep the printed backup ID for `mirror restore`.
3. **Local RPM repositories:** `repo create` downloads named packages and dependencies, including dependencies already installed on the download machine. It verifies signatures and produces repository metadata and a disabled `.repo` file. This prepares a reusable package set, not a full Rocky mirror or an HTTP server.
4. **Development environments:** install Python, Java, Go, Node or build-tool presets. Existing environments stay unchanged by default. `--update` allows DNF to update system runtimes; custom runtimes use their original manager. For an isolated project, use `rockdev env install python --venv "$HOME/project/.venv"`. The parent must exist and the target must be new; this command does not install system Python or overwrite a venv.
5. **Package sources:** inspect the target file and environment overrides with `source show pip`, then change it with commands such as `source use pip official`. Unrelated settings are preserved and a backup ID is printed. `source restore` checks hashes; manually edited files are not overwritten. Sudo configures root. Existing private Maven central or wildcard mirrors are not automatically replaced.
6. **Deployment profiles:** export with `profile export --file dev.json python go`, check with `profile validate dev.json`, and preview with `profile apply dev.json --dry-run` before applying. JSON can declare environments, a venv and sources. `update` defaults to false; arbitrary shell commands are not supported. Execution stops at the first failure without uninstalling packages already installed.
7. **Offline bundles:** on a connected machine, run `repo create`, then `bundle export`. Transfer the archive and run `bundle import` on the target. Import checks the exact Rocky version, architecture, sizes, SHA256 and trusted RPM signatures. It prepares a repository and does not install packages. The target must already trust the corresponding signing keys; archives cannot be mixed across minor releases or architectures.

See the [feature guide](docs/release-1.0.en.md) for profile and offline examples. This installation command uses only the prepared repository without changing global repository files:

```bash
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

## Files and limitations

- System logs: `/var/lib/rockdev/logs/`; system mirror backups: `/var/lib/rockdev/backups/`. User-operation logs are in `~/.local/state/rockdev/logs/`, and package-source backups in `~/.local/state/rockdev/sources/`.
- Mirror changes affect existing BaseOS, AppStream and extras sections. Other repositories are preserved. Failed cache refresh restores the original files.
- Restore checks file hashes and stops if the configuration was edited manually.
- The local repository is a selected RPM set with `module_hotfixes=1`, not a full distribution mirror. Clients need the corresponding trusted RPM keys.
- Installed packages are not rolled back on failure. No HTTP repository service or npm/pip/Maven/Go module mirror is included.
- Candidate platforms are Rocky 8/9/10 on x86_64/aarch64. Version 1.0.1 passed 59 integration cases each on Rocky 8.10 and 9.8 x86_64; ARM64 was only cross-built. Rocky 10.2 has historical 1.0.0 runtime evidence. See the [latest test report](docs/test-results-1.0.1.en.md). System deployment on other distributions is unsupported.

## Build

You need Git and **Go 1.22 or later**. Check `git --version` and `go version` first. DNF may provide an older Go version; prepare a suitable compiler using the [official Go installation guide](https://go.dev/doc/install) if needed.

This section is for developers changing the source; regular users can use `./rockdev` above. The `dist/` directory is not tracked, and local builds need Go. Public HTTPS clones need no repository access authorization; private repositories still require permission. Do not put tokens in URLs. Cloning does not change system configuration.

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang en
```

To use the `rockdev` command from any directory, optionally install it into PATH:

```bash
sudo install -m 0755 ./dist/rockdev /usr/local/bin/rockdev
rockdev --help
```

Windows users can cross-build Linux binaries with `scripts/build.ps1`. A Windows build does not add support for provisioning Rocky environments on Windows. See the [manual VM steps](docs/vm-quickstart.en.md).

## Publication and license status

This project uses the [MIT license](LICENSE), copyright `2026 YWQ211`. Use, modification, redistribution and commercial use are permitted with the copyright and license notices retained. The software is provided as is, without warranty. Notices for the Go runtime included in binaries are in [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).

Repository visibility is controlled separately by the maintainer; adding a license does not make a private repository public. See the [maintainer publishing guide](docs/publishing.md).
