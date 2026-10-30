#!/bin/bash
# Run as root in an isolated Rocky container; /results is the only writable host mount.
set -u
mkdir -p /results /srv/rockdev /tmp/smoke
printf 'case\texpected\tactual\tresult\n' > /results/results.tsv
run() {
 local name=$1 expected=$2 actual result
 shift 2
 timeout 360 "$@" > "/results/$name.log" 2>&1
 actual=$?;result=FAIL
 if { [ "$expected" = 0 ] && [ "$actual" = 0 ]; } || { [ "$expected" = nonzero ] && [ "$actual" != 0 ] && [ "$actual" != 124 ]; }; then result=PASS; fi
 printf '%s\t%s\t%s\t%s\n' "$name" "$expected" "$actual" "$result" | tee -a /results/results.tsv
}
cat /etc/os-release > /results/os-release.txt
rpm -qa | sort > /results/packages-before.txt
run version 0 rockdev version
for lang in zh en ja; do
 run "help-$lang" 0 rockdev --lang "$lang" --help
 run "menu-$lang" 0 bash -c "printf '0\n' | rockdev --lang $lang"
 run "doctor-$lang" 0 rockdev --lang "$lang" doctor
 run "preview-$lang" 0 rockdev --lang "$lang" env install python java go node build --dry-run
done
run invalid-lang nonzero rockdev --lang fr --help
run invalid-command nonzero rockdev unknown
run invalid-env nonzero rockdev env install unknown --yes
run invalid-package nonzero rockdev repo create --dir /srv/rockdev/invalid --nogpgcheck --yes
run mirror-list 0 rockdev --lang en mirror list
for id in official aliyun tuna ustc; do run "mirror-check-$id" 0 rockdev --lang en mirror check "$id"; done
run mirror-preview 0 rockdev --lang en mirror use auto --dry-run
run mirror-use 0 rockdev --lang en mirror use auto --yes
backup=''
for d in /var/lib/rockdev/backups/backup-*; do [ -d "$d" ] && backup=${d##*/}; done
if [ -n "$backup" ]; then
 cp -a /etc/yum.repos.d /tmp/changed-repos
 file=$(ls /etc/yum.repos.d/*.repo | head -n 1)
 printf '\n# manual edit\n' >> "$file"
 run restore-edited nonzero rockdev --lang en mirror restore "$backup" --yes
 cp -a /tmp/changed-repos/. /etc/yum.repos.d/
 run mirror-restore 0 rockdev --lang en mirror restore "$backup" --yes
fi
run restore-invalid nonzero rockdev --lang ja mirror restore ../escape --yes
for env in python java go node build; do run "install-$env" 0 rockdev --lang en env install "$env" --yes; done
run reinstall 0 rockdev --lang en env install python java go node build --yes
run env-list 0 rockdev --lang ja env list
run python-venv 0 bash -c 'python3 -m venv /tmp/smoke/venv && /tmp/smoke/venv/bin/python -c "print(6*7)" && /tmp/smoke/venv/bin/pip --version'
cat > /tmp/smoke/Hello.java <<'EOF'
class Hello { public static void main(String[] args) { System.out.println(42); } }
EOF
run java-compile 0 bash -c 'cd /tmp/smoke && javac Hello.java && java Hello'
cat > /tmp/smoke/main.go <<'EOF'
package main
import "fmt"
func main(){fmt.Println(42)}
EOF
run go-compile 0 bash -c 'cd /tmp/smoke && go build -o hello main.go && ./hello'
run node-script 0 node -e 'console.log(6*7)'
run npm-package 0 bash -c 'cd /tmp/smoke && npm install --no-audit --no-fund --registry=https://registry.npmjs.org is-number && node -e "console.log(require(\"is-number\")(42))"'
printf '#include <stdio.h>\nint main(void){puts("42");return 0;}\n' > /tmp/smoke/hello.c
run gcc-compile 0 bash -c 'cd /tmp/smoke && gcc hello.c -o hello-c && ./hello-c'
run cpp-compile 0 bash -c 'cd /tmp/smoke && g++ hello.c -o hello-cpp && ./hello-cpp'
run make 0 make --version
run git 0 git --version
run repo-preview 0 rockdev --lang en repo create --dir /srv/rockdev/preview python3 --dry-run
run repo-create 0 rockdev --lang en repo create --dir /srv/rockdev/python python3 python3-pip --yes
run repo-existing nonzero rockdev --lang en repo create --dir /srv/rockdev/python python3 --yes
run repo-node 0 rockdev --lang en repo create --dir /srv/rockdev/node nodejs npm --yes
run repo-missing-parent nonzero rockdev --lang ja repo create --dir /missing/path/new python3 --yes
run repo-unavailable-package nonzero rockdev --lang en repo create --dir /srv/rockdev/missing rockdev-no-such-package-0x123 --yes
run failed-repo-not-published 0 test ! -e /srv/rockdev/missing
run preview-no-repo 0 test ! -e /srv/rockdev/preview
if command -v python3 >/dev/null; then
 cat > /tmp/smoke/lock.py <<'EOF'
import fcntl,subprocess,sys
with open('/var/lib/rockdev/operation.lock','a') as f:
 fcntl.flock(f,fcntl.LOCK_EX|fcntl.LOCK_NB)
 p=subprocess.run(['rockdev','--lang','en','env','install','python','--yes'],capture_output=True,text=True)
 print(p.stdout,p.stderr)
 sys.exit(0 if p.returncode and 'Another rockdev' in p.stderr else 1)
EOF
 run operation-lock 0 python3 /tmp/smoke/lock.py
fi
rpm -qa | sort > /results/packages-after.txt
printf 'INTEGRATION_DONE\n'
