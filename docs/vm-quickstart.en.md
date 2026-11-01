# Running Rockdev on a VM

[中文](vm-quickstart.md) · English · [日本語](vm-quickstart.ja.md)

## Get the source

This is a private repository. Use an authorized GitHub SSH key or an HTTPS token. Do not put tokens in repository URLs, scripts or commits. Your Linux login password is not a GitHub credential.

```bash
git clone https://github.com/YWQ211/Rocky_Linux_deployment_tool.git
cd Rocky_Linux_deployment_tool
```

If the VM has no GitHub credentials, use a separately supplied Linux binary until access is configured.

## Build from source

Requires Go 1.22 or later. The compiled program does not require Go.

```bash
go version
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/rockdev ./cmd/rockdev
./dist/rockdev --lang en --help
./dist/rockdev --lang en doctor
./dist/rockdev --lang en env install python go --dry-run
./dist/rockdev --lang en
```

## Use a prebuilt binary

Choose the binary matching your CPU. Replace `amd64` with `arm64` for ARM64.

```bash
chmod +x rockdev-linux-amd64
./rockdev-linux-amd64 --lang en --help
./rockdev-linux-amd64 --lang en doctor
./rockdev-linux-amd64 --lang en env install python go --dry-run
./rockdev-linux-amd64 --lang en
```

The `rockdev-linux-amd64-en` build defaults to English. All builds accept `--lang zh|en|ja`; `ROCKDEV_LANG` sets the default. System tools such as DNF retain their own output language.

## Try the workflows

Replace `rockdev` below with the actual program path, such as `./rockdev-linux-amd64` or `./dist/rockdev`. Add `--lang en` to select English, including when using sudo.

```bash
# Read-only system check
rockdev --lang en doctor

# Check mirrors without changing configuration
rockdev --lang en mirror list
rockdev --lang en mirror check

# Switch to a currently available mirror
sudo rockdev --lang en mirror use auto

# Install development tools
sudo rockdev --lang en env install python java go node build

# Create a local RPM repository; the target must not already exist
sudo mkdir -p /srv/rockdev
sudo rockdev --lang en repo create --dir /srv/rockdev/python python3 python3-pip
```

To restore repository settings, use the backup ID printed by the switch command:

```bash
sudo rockdev --lang en mirror restore <backup-id>
```

Logs are saved under `/var/lib/rockdev/logs/`. Changes require confirmation unless `--yes` is given. `--dry-run` shows the plan without network access, file changes or installation.

Take a VM snapshot before a first deployment. See the [test report](test-results.en.md) for the tested platform, checks and known limits.
