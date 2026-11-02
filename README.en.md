# Rockdev 1.0.1

[中文](README.md) · English · [日本語](README.ja.md)

Documentation: [1.0 guide](docs/release-1.0.en.md) · [VM guide](docs/vm-quickstart.en.md) · [1.0 test report](docs/test-results-1.0.en.md)

1.0.1 fixes system Python detection on Rocky 8. [Patch notes and tests](docs/release-1.0.1.en.md).

A Go CLI for Rocky Linux. It checks the system, switches package mirrors, creates a local RPM repository and installs development tools.

## Run

Use the Linux binary for your architecture. Go is needed to build from source, but is not needed to run the binary.

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

## Commands

Version 1.0 adds environment inspection, package sources, JSON deployment profiles and offline bundle export/import. See the [1.0 guide](docs/release-1.0.en.md) for options and limits.

```bash
rockdev env check
rockdev env install python --venv /home/your-user/project/.venv
sudo rockdev env install python java go --update
rockdev source use pip tuna
rockdev profile apply examples/user-sources.json --dry-run
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

Run without arguments to open the menu. Changes require root and confirmation. `--yes` skips confirmation. `--dry-run` prints the plan without network requests, file writes or package commands.

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

## Files and limitations

- Logs: `/var/lib/rockdev/logs/`; configuration backups: `/var/lib/rockdev/backups/`.
- Mirror changes affect existing BaseOS, AppStream and extras sections. Other repositories are preserved. Failed cache refresh restores the original files.
- Restore checks file hashes and stops if the configuration was edited manually.
- The local repository is a selected RPM set with `module_hotfixes=1`, not a full distribution mirror. Clients need the corresponding trusted RPM keys.
- Installed packages are not rolled back on failure. No HTTP repository service or npm/pip/Maven/Go module mirror is included.
- Supported code paths: Rocky 8/9/10 on x86_64/aarch64. See [test results](docs/test-results-1.0.en.md) for what was actually tested.

## Build

The repository is private; GitHub credentials are required to clone it. Do not put tokens in the URL.

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
go test ./...
go vet ./...
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang en
```

Go 1.22 or later is required. Windows builds can use `scripts/build.ps1`. [Manual VM steps](docs/vm-quickstart.en.md).
