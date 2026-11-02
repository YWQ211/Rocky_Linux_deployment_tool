#!/bin/bash
# Additional real-process regression inside the disposable Rocky test container.
set -eu
mkdir -p /tmp/compat-guards/bin
printf '#!/bin/sh\nexec /usr/bin/python3 "$@"\n' > /tmp/compat-guards/bin/python3
chmod +x /tmp/compat-guards/bin/python3
set +e
PATH="/tmp/compat-guards/bin:$PATH" rockdev --lang en env install python --update --yes > /results/custom-python.log 2>&1
rc=$?
set -e
test "$rc" = 1
grep -q 'Cannot update custom runtime /tmp/compat-guards/bin/python3' /results/custom-python.log
printf 'CUSTOM_PYTHON_GUARD_PASS\n'
