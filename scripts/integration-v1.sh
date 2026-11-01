#!/bin/bash
set -u
mkdir -p /results /srv/rockdev /tmp/v1
printf 'case\texpected\tactual\tresult\n' > /results/results.tsv
run(){ local name=$1 expected=$2 actual result;shift 2;timeout 600 "$@" > "/results/$name.log" 2>&1;actual=$?;result=FAIL;if { [ "$expected" = 0 ] && [ "$actual" = 0 ]; } || { [ "$expected" = nonzero ] && [ "$actual" != 0 ] && [ "$actual" != 124 ]; };then result=PASS;fi;printf '%s\t%s\t%s\t%s\n' "$name" "$expected" "$actual" "$result" | tee -a /results/results.tsv; }
cat /etc/os-release > /results/os-release.txt
run version 0 rockdev version
for lang in zh en ja;do run "help-$lang" 0 rockdev --lang "$lang" --help;run "menu-$lang" 0 bash -c "printf '0\n' | rockdev --lang $lang";done
run health-before 0 rockdev --lang en env check
rpm -qa | sort > /tmp/v1/before.txt
run keep-existing 0 rockdev --lang en env install python --yes
rpm -qa | sort > /tmp/v1/kept.txt
run kept-rpm-inventory 0 python3 -c 'import pathlib,sys;sys.exit(0 if pathlib.Path(sys.argv[1]).read_bytes()==pathlib.Path(sys.argv[2]).read_bytes() else 1)' /tmp/v1/before.txt /tmp/v1/kept.txt
run update-explicit 0 rockdev --lang en env install python java go node build --update --yes
run health-after 0 rockdev --lang en env check
run venv-create 0 rockdev --lang en env install python --venv /tmp/v1/venv --yes
run venv-existing nonzero rockdev --lang en env install python --venv /tmp/v1/venv --yes
run health-isolated 0 env PATH="/tmp/v1/venv/bin:$PATH" VIRTUAL_ENV=/tmp/v1/venv rockdev --lang en env check python
run isolation-detected 0 grep -q 'isolated=true' /results/health-isolated.log
rpm -qa | sort > /tmp/v1/isolated-before.txt
run keep-isolated 0 env PATH="/tmp/v1/venv/bin:$PATH" VIRTUAL_ENV=/tmp/v1/venv rockdev --lang en env install python --update --yes
rpm -qa | sort > /tmp/v1/isolated-after.txt
run isolated-rpm-inventory 0 python3 -c 'import pathlib,sys;sys.exit(0 if pathlib.Path(sys.argv[1]).read_bytes()==pathlib.Path(sys.argv[2]).read_bytes() else 1)' /tmp/v1/isolated-before.txt /tmp/v1/isolated-after.txt
mkdir -p /tmp/v1/custom
printf '#!/bin/sh\necho custom-go\n' > /tmp/v1/custom/go;chmod +x /tmp/v1/custom/go
run custom-runtime-rejected nonzero env PATH="/tmp/v1/custom:$PATH" rockdev --lang en env install go --update --yes
run source-list 0 rockdev --lang en source list
printf 'cache=/tmp/npm-cache\n//private.invalid/:_authToken=fixture-only\n' > /root/.npmrc
cp /root/.npmrc /tmp/v1/npm-original
run source-npm 0 rockdev --lang en source use npm npmmirror --yes
run source-npm-effective 0 bash -c 'test "$(npm config get registry)" = https://registry.npmmirror.com/'
run source-npm-preserve 0 grep -q fixture-only /root/.npmrc
backup=$(python3 -c 'import pathlib,json;print(next(p.stem for p in pathlib.Path("/root/.local/state/rockdev/sources").glob("*.json") if json.loads(p.read_text())["Tool"]=="npm"))')
cp /root/.npmrc /tmp/v1/npm-managed;printf '\n# manual\n' >> /root/.npmrc
run source-edited-rejected nonzero rockdev --lang en source restore "$backup" --yes
cp /tmp/v1/npm-managed /root/.npmrc
run source-restore 0 rockdev --lang en source restore "$backup" --yes
run source-restored-bytes 0 python3 -c 'import pathlib,sys;sys.exit(0 if pathlib.Path(sys.argv[1]).read_bytes()==pathlib.Path(sys.argv[2]).read_bytes() else 1)' /root/.npmrc /tmp/v1/npm-original
run source-pip 0 rockdev --lang en source use pip tuna --yes
run source-pip-effective 0 bash -c 'test "$(python3 -m pip config get global.index-url)" = https://pypi.tuna.tsinghua.edu.cn/simple/'
run source-maven 0 rockdev --lang en source use maven aliyun --yes
mkdir -p /tmp/v1/maven
printf '<project><modelVersion>4.0.0</modelVersion><groupId>test</groupId><artifactId>test</artifactId><version>1</version><packaging>pom</packaging></project>' > /tmp/v1/maven/pom.xml
run source-maven-effective 0 mvn -o -f /tmp/v1/maven/pom.xml validate
go env GOSUMDB > /tmp/v1/sumdb-before
run source-go 0 rockdev --lang en source use go goproxy.cn --yes
run source-go-effective 0 bash -c 'test "$(go env GOPROXY)" = https://goproxy.cn'
go env GOSUMDB > /tmp/v1/sumdb-after
run source-sumdb-preserved 0 python3 -c 'import pathlib,sys;sys.exit(0 if pathlib.Path(sys.argv[1]).read_bytes()==pathlib.Path(sys.argv[2]).read_bytes() else 1)' /tmp/v1/sumdb-before /tmp/v1/sumdb-after
for pair in 'pip official' 'npm official' 'maven official' 'go goproxy.cn';do set -- $pair;run "source-check-$1" 0 rockdev --lang en source check "$1" "$2";done
run profile-export 0 rockdev --lang en profile export --file /tmp/v1/export.json python go --yes
run profile-validate 0 rockdev --lang en profile validate /tmp/v1/export.json
run profile-apply-keep 0 rockdev --lang en profile apply /tmp/v1/export.json --yes
run profile-export-existing nonzero rockdev --lang en profile export --file /tmp/v1/export.json python --yes
printf '{"schema":1,"environments":["python"],"update":false,"Update":true}' > /tmp/v1/unsafe.json
run profile-duplicate-case nonzero rockdev --lang en profile apply /tmp/v1/unsafe.json --yes
printf '{"schema":1,"environments":["python"],"venv":"/tmp/v1/profile-venv","sources":[{"tool":"pip","provider":"official"}]}' > /tmp/v1/venv.json
run profile-venv-preview 0 rockdev --lang ja profile apply /tmp/v1/venv.json --dry-run
run profile-preview-no-write 0 test ! -e /tmp/v1/profile-venv
run profile-venv-apply 0 rockdev --lang en profile apply /tmp/v1/venv.json --yes
run profile-pip-scoped 0 env PIP_CONFIG_FILE=/tmp/v1/profile-venv/pip.conf /tmp/v1/profile-venv/bin/python -m pip config get global.index-url
run source-global-pip-preserved 0 bash -c 'test "$(python3 -m pip config get global.index-url)" = https://pypi.tuna.tsinghua.edu.cn/simple/'
run repo-create 0 rockdev --lang en repo create --dir /srv/rockdev/python python3 python3-pip --yes
run bundle-export 0 rockdev --lang en bundle export --from /srv/rockdev/python --file /results/python.tar.gz --yes
run bundle-existing nonzero rockdev --lang en bundle export --from /srv/rockdev/python --file /results/python.tar.gz --yes
run bundle-preview 0 rockdev --lang en bundle import --file /results/python.tar.gz --dir /srv/rockdev/imported --dry-run
run bundle-preview-no-write 0 test ! -e /srv/rockdev/imported
run bundle-cancel 0 bash -c "printf 'n\n' | rockdev --lang en bundle import --file /results/python.tar.gz --dir /srv/rockdev/cancelled"
run bundle-cancel-no-write 0 test ! -e /srv/rockdev/cancelled
run bundle-cancel-no-success 0 bash -c '! grep -q "Local repository prepared" /results/bundle-cancel.log'
run bundle-import 0 rockdev --lang en bundle import --file /results/python.tar.gz --dir /srv/rockdev/imported --yes
run bundle-install-command 0 grep -q -- '--enablerepo=rockdev-local' /results/bundle-import.log
cp /results/python.tar.gz /tmp/v1/corrupt.tar.gz;printf broken | dd of=/tmp/v1/corrupt.tar.gz bs=1 seek=30 conv=notrunc status=none
run bundle-corrupt nonzero rockdev --lang en bundle import --file /tmp/v1/corrupt.tar.gz --dir /srv/rockdev/corrupt --yes
run bundle-corrupt-no-write 0 test ! -e /srv/rockdev/corrupt
printf 'V1_INTEGRATION_DONE\n'
