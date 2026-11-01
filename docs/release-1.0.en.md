# Rockdev 1.0.0

[中文](release-1.0.md) · English · [日本語](release-1.0.ja.md)

## Check environments before updating

```bash
rockdev env check
rockdev env check python java go
rockdev env install python go
sudo rockdev env install python go --update
rockdev env install python --venv /home/your-user/project/.venv
```

Checks show command paths, versions, duplicate PATH candidates, JAVA_HOME mismatches and Python isolation. They execute version queries and do not replace project tests.

Go checks use `GOTOOLCHAIN=local` in the subprocess to avoid automatic toolchain downloads, without changing the caller's environment.

Existing environments are kept by default. Interactive installation asks whether to update each one. `--yes` skips confirmation but does not authorize upgrades; `--update` permits updates of system runtimes. Custom runtimes must be updated through their original manager. Installing a system package is not reported as updating a custom runtime.

A detected Python venv/Conda environment is kept without changing global packages. `--venv` creates a new Python environment without DNF. Python and the parent directory must already exist; the target must be new. Existing venvs are not overwritten or upgraded.

## Package sources

```bash
rockdev source list
rockdev source show pip
rockdev source check pip tuna
rockdev source use pip tuna
rockdev source use npm npmmirror
rockdev source use maven aliyun
rockdev source use go goproxy.cn
rockdev source restore BACKUP_ID
```

Presets cover pip, npm, Maven and Go over HTTPS. Configuration belongs to the current user. Under sudo this is root, not the invoking developer. Plans display actual file paths and backup IDs.

Unrelated settings and npm credentials are preserved; Go's checksum database is not disabled. Existing Maven central or wildcard mirrors require manual review to protect private repositories. Environment variables that override configuration are identified by name, without printing their values.

pip respects explicit PIP_CONFIG_FILE or a configured venv's pip.conf, then the user configuration directory. npm userconfig and GOENV are supported. Restore checks backup metadata and current contents and refuses to overwrite manual edits. Backup hashes detect corruption, not a same-user attacker who can rewrite and rehash backups.

## Deployment profiles

```bash
rockdev profile export --file dev.json python java go
rockdev profile validate dev.json
rockdev profile apply dev.json --dry-run
sudo rockdev profile apply dev.json
rockdev profile apply examples/user-sources.json
```

Profiles are versioned JSON with no shell commands. Unknown fields, duplicate or differently cased keys, and unknown presets/providers are rejected before execution.

```json
{
  "schema": 1,
  "environments": ["python", "go"],
  "update": false,
  "sources": [{"tool": "go", "provider": "goproxy.cn"}]
}
```

`update` defaults to false; set it to true explicitly to allow updates. `profile apply --yes` does not change the policy. A mixed system/user profile run with sudo configures root's sources too; apply system and user settings separately when needed.

Python plus an absolute `venv` path creates an isolated environment; a pip source in the same profile targets that venv's pip.conf. The parent must exist and the target must be new. Edit example paths in `examples/` before use.

Execution stops at the first failed task. Completed package installations are not rolled back; retain logs and backups for recovery.

## Offline bundles

```bash
sudo rockdev repo create --dir /srv/rockdev/python python3 python3-pip
rockdev bundle export --from /srv/rockdev/python --file "$HOME/python.tar.gz"
rockdev bundle import --file "$HOME/python.tar.gz" --dir "$HOME/offline-python"
sudo dnf --setopt=reposdir="$HOME/offline-python" --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
```

Import prepares a repository without installing packages. It prints an explicit installation command that selects its directory and disables other repositories.

Targets must be new. Import verifies the exact Rocky release and architecture, manifest, sizes, SHA256 and trusted RPM signatures. Traversal, links, duplicates and hidden appended archives are rejected. Hashes check integrity; RPM signatures check package origin. Target machines must already trust the correct RPM keys; unknown keys are not imported.

Processing is streamed. Limits are 4 GiB expanded data, 10000 files and 8 MiB per manifest/metadata document. The repository is published only after all package signatures pass. Exact minor-release matching is required because dependencies can differ between Rocky point releases.

## Scope

Chinese, English and Japanese remain available, including default-language builds. Runtime multi-version management, a full mirror server and automatic uninstall are outside this release. See the [1.0 test report](test-results-1.0.en.md). Historical 0.1 results keep their original test date.
