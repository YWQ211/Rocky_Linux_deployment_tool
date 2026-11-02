#!/bin/sh
# Real Rocky + Go ELF smoke; fixture transport avoids external network in container.
set -eu
cd "$(dirname "$0")/.."
image=${ROCKDEV_SMOKE_IMAGE:-quay.io/rockylinux/rockylinux:9}
docker run --rm --network none --memory 128m --cpus 1 \
    -v "$PWD:/repo:ro" -v "$PWD/dist/release:/input:ro" "$image" sh -eu -c '
mkdir -p /fixture/bin
cat > /fixture/bin/curl <<"STUB"
#!/bin/sh
while [ "$#" -gt 0 ]; do
    case "$1" in --output) out=$2; shift 2;; *) url=$1; shift;; esac
done
echo "$url" >> /fixture/downloads
if [ -f /fixture/bad ]; then printf bad > "$out"; else cp "/input/${url##*/}" "$out"; fi
STUB
chmod +x /fixture/bin/curl
export PATH=/fixture/bin:/usr/bin:/bin
cd /repo
version=$(cat release/VERSION)
XDG_CACHE_HOME=/tmp/untrusted ./rockdev version | grep -Fx "$version"
test ! -e /tmp/untrusted/rockdev
test -x "/var/cache/rockdev/$version/amd64/rockdev-linux-amd64"
./rockdev doctor
./rockdev --lang ja --help > /fixture/help
grep -q "開発環境" /fixture/help
test "$(wc -l < /fixture/downloads)" = 1
printf bad > "/var/cache/rockdev/$version/amd64/rockdev-linux-amd64"
./rockdev version | grep -Fx "$version"
test "$(wc -l < /fixture/downloads)" = 2
touch /fixture/bad
printf bad > "/var/cache/rockdev/$version/amd64/rockdev-linux-amd64"
if ./rockdev version; then echo "corrupt download executed" >&2; exit 1; fi
echo ROCKY_ELF_LAUNCHER_SMOKE_PASS
'
