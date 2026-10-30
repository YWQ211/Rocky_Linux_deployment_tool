#!/bin/bash
set -u
mkdir -p /results /var/lib/rockdev
printf 'case\texpected\tactual\tresult\n' > /results/results.tsv
run(){ local name=$1 expected=$2 actual result;shift 2;timeout 180 "$@" > "/results/$name.log" 2>&1;actual=$?;result=FAIL;if { [ "$expected" = 0 ] && [ "$actual" = 0 ]; } || { [ "$expected" = nonzero ] && [ "$actual" != 0 ] && [ "$actual" != 124 ]; };then result=PASS;fi;printf '%s\t%s\t%s\t%s\n' "$name" "$expected" "$actual" "$result" | tee -a /results/results.tsv; }
# Give this disposable container an explicit starting source different from Aliyun.
for f in /etc/yum.repos.d/*.repo; do sed -i 's|https://mirrors.aliyun.com/rockylinux|https://dl.rockylinux.org/pub/rocky|g' "$f"; done
run switch-with-backup 0 rockdev --lang en mirror use aliyun --yes
backup='';for d in /var/lib/rockdev/backups/backup-*;do [ -d "$d" ] && backup=${d##*/};done
if [ -n "$backup" ];then
 before=(/var/lib/rockdev/backups/backup-*)
 run same-mirror 0 rockdev --lang ja mirror use aliyun --yes
 after=(/var/lib/rockdev/backups/backup-*)
 run same-mirror-no-new-backup 0 test "${#before[@]}" = "${#after[@]}"
 file=$(python3 -c 'import json,sys;print("/etc/yum.repos.d/"+json.load(open(sys.argv[1]))["Entries"][0]["Name"])' "/var/lib/rockdev/backups/$backup/manifest.json")
 cp "$file" /tmp/managed.repo
 printf '\n# manual edit\n' >> "$file"
 run edited-restore-rejected nonzero rockdev --lang en mirror restore "$backup" --yes
 cp /tmp/managed.repo "$file"
 run restore 0 rockdev --lang ja mirror restore "$backup" --yes
fi
sha256sum /etc/yum.repos.d/*.repo > /tmp/before.sha256
mkdir -p /tmp/fail-dnf
printf '#!/bin/sh\nexit 42\n' > /tmp/fail-dnf/dnf;chmod 0755 /tmp/fail-dnf/dnf
run refresh-failure nonzero env PATH="/tmp/fail-dnf:$PATH" rockdev --lang en mirror use aliyun --yes
run rollback-hashes 0 sha256sum -c /tmp/before.sha256
cat > /tmp/edge.py <<'EOF'
import fcntl,subprocess,signal,sys,time
mode=sys.argv[1]
if mode=='cancel':
 p=subprocess.Popen(['rockdev','--lang','en'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
 time.sleep(.2);p.send_signal(signal.SIGTERM);p.wait(timeout=3);p.communicate();print('exit',p.returncode);sys.exit(0 if p.returncode==1 else 1)
if mode=='lock':
 with open('/var/lib/rockdev/operation.lock','a') as f:
  fcntl.flock(f,fcntl.LOCK_EX|fcntl.LOCK_NB)
  p=subprocess.run(['rockdev','--lang','en','env','install','python','--yes'],capture_output=True,text=True)
  print(p.stdout,p.stderr);sys.exit(0 if p.returncode and 'Another rockdev' in p.stderr else 1)
EOF
run signal-cancel 0 python3 /tmp/edge.py cancel
run lock 0 python3 /tmp/edge.py lock
cp /etc/os-release /tmp/os-release.original
printf 'ID=ubuntu\nVERSION_ID=24.04\n' > /etc/os-release
run unsupported-os nonzero rockdev --lang en env install python --dry-run
cp /tmp/os-release.original /etc/os-release
printf 'EDGE_DONE\n'
