#!/bin/bash
# Execute after the integration suite in its disposable Rocky container.
set -u
mkdir -p /tmp/compat-runtime
cd /tmp/compat-runtime || exit 1
printf 'case\texpected\tactual\tresult\n' > /results/runtime.tsv
run() {
  local label=$1 rc result=FAIL
  shift
  timeout 90 "$@" > "/results/runtime-$label.log" 2>&1
  rc=$?
  [ "$rc" = 0 ] && result=PASS
  printf '%s\t0\t%s\t%s\n' "$label" "$rc" "$result" | tee -a /results/runtime.tsv
}
printf 'print("python-ok")\n' > hello.py
printf '#include <stdio.h>\nint main(void){puts("c-ok");return 0;}\n' > hello.c
printf 'public class Hello { public static void main(String[] args){System.out.println("java-ok");} }\n' > Hello.java
printf 'package main\nimport "fmt"\nfunc main(){fmt.Println("go-ok")}\n' > hello.go
run python python3 hello.py
run c-build gcc hello.c -o hello-c
run c-run ./hello-c
run java-build javac Hello.java
run java-run java Hello
run go-build env GOTOOLCHAIN=local go build -o hello-go hello.go
run go-run ./hello-go
run node node -e 'console.log("node-ok")'
run doctor rockdev --lang en doctor
run mirror-check rockdev --lang en mirror check official
awk -F '\t' 'NR > 1 && $4 != "PASS" {failed=1} END {exit failed}' /results/runtime.tsv
