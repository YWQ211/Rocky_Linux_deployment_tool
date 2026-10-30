#!/bin/bash
set -u
mkdir -p /results /tmp/src
printf 'case\texpected\tactual\tresult\n' > /results/results.tsv
run(){ local name=$1 expected=$2 actual result;shift 2;timeout 300 "$@" > "/results/$name.log" 2>&1;actual=$?;result=FAIL;if { [ "$expected" = 0 ] && [ "$actual" = 0 ]; } || { [ "$expected" = nonzero ] && [ "$actual" != 0 ] && [ "$actual" != 124 ]; };then result=PASS;fi;printf '%s\t%s\t%s\t%s\n' "$name" "$expected" "$actual" "$result" | tee -a /results/results.tsv; }
python3 -m zipfile -e /app/test-source.zip /tmp/src
run linux-unit-tests 0 bash -c 'cd /tmp/src && CGO_ENABLED=0 GOCACHE=/tmp/go-cache /opt/go/bin/go test ./... -count=1'
run linux-vet 0 bash -c 'cd /tmp/src && CGO_ENABLED=0 GOCACHE=/tmp/go-cache /opt/go/bin/go vet ./...'
run english-build 0 /app/rockdev-linux-amd64-en --help
run japanese-build 0 /app/rockdev-linux-amd64-ja --help
run english-doctor 0 /app/rockdev-linux-amd64-en doctor
run japanese-doctor 0 /app/rockdev-linux-amd64-ja doctor
run japanese-log-error nonzero /app/rockdev-linux-amd64-ja mirror restore ../bad --yes
cat > /tmp/final.py <<'EOF'
import subprocess,signal,time,sys,pathlib
exe='/app/rockdev-linux-amd64'
mode=sys.argv[1]
if mode=='cancel':
 p=subprocess.Popen([exe,'--lang','en'],stdin=subprocess.PIPE,stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
 time.sleep(.3);p.send_signal(signal.SIGTERM);p.wait(timeout=3);print('exit',p.returncode);print(p.communicate()[1].decode());sys.exit(0 if p.returncode==1 else 1)
if mode=='nonroot':
 p=subprocess.run([exe,'--lang','en','env','install','python','--yes'],capture_output=True,text=True,user=65534);print(p.stdout,p.stderr);sys.exit(0 if p.returncode==1 and 'requires root' in p.stderr else 1)
if mode=='log':
 text=''.join(p.read_text() for p in pathlib.Path('/var/lib/rockdev/logs').glob('*.log'));print(text);sys.exit(0 if '無効なバックアップ ID' in text and '无效备份' not in text else 1)
EOF
run signal-cancel 0 python3 /tmp/final.py cancel
run nonroot-rejected 0 python3 /tmp/final.py nonroot
run japanese-failure-log 0 python3 /tmp/final.py log
/opt/go/bin/go version > /results/go-version.txt
printf 'FINAL_DONE\n'
