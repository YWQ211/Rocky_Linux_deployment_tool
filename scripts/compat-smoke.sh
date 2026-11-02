#!/bin/bash
# Run inside a disposable container with /results dedicated to this test.
set -u
mkdir -p /results /tmp/compat
printf 'case\texpected\tactual\tresult\n' > /results/results.tsv
run() {
  local label=$1 expected=$2 rc result=FAIL
  shift 2
  timeout 60 "$@" > "/results/$label.log" 2>&1
  rc=$?
  if { [ "$expected" = 0 ] && [ "$rc" = 0 ]; } || { [ "$expected" = reject ] && [ "$rc" = 1 ]; }; then result=PASS; fi
  printf '%s\t%s\t%s\t%s\n' "$label" "$expected" "$rc" "$result" | tee -a /results/results.tsv
}
cat /etc/os-release > /results/os-release.txt
run version 0 rockdev version
for lang in zh en ja; do run "help-$lang" 0 rockdev --lang "$lang" --help; done
run env-check 0 rockdev --lang en env check
run source-list 0 rockdev --lang en source list
run source-pip 0 rockdev --lang en source use pip official --yes
run source-npm 0 rockdev --lang en source use npm npmmirror --yes
run source-maven 0 rockdev --lang en source use maven aliyun --yes
run source-go 0 rockdev --lang en source use go goproxy.cn --yes
run profile-export 0 rockdev --lang en profile export --file /tmp/compat/dev.json python go --yes
run profile-validate 0 rockdev --lang en profile validate /tmp/compat/dev.json
run doctor-rejected reject rockdev --lang en doctor
run install-rejected reject rockdev --lang en env install python --update --yes
run repo-rejected reject rockdev --lang en repo create --dir /tmp/compat/repo python3 --yes
run mirror-rejected reject rockdev --lang en mirror use aliyun --yes
run profile-system-rejected reject rockdev --lang en profile apply /tmp/compat/dev.json --yes
run bundle-rejected reject rockdev --lang en bundle import --file /tmp/missing.tar.gz --dir /tmp/compat/import --yes
for label in doctor install repo mirror profile-system bundle; do
  run "$label-os-guard" 0 grep -q 'Only Rocky Linux' "/results/$label-rejected.log"
done
run no-repo-created 0 test ! -e /tmp/compat/repo
run no-import-created 0 test ! -e /tmp/compat/import
printf 'COMPAT_SMOKE_DONE\n'
