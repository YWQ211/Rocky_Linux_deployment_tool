#!/bin/bash
# Dedicated fresh Rocky container, started with --network none.
set -eu
major=$1
rpm --import "/etc/pki/rpm-gpg/RPM-GPG-KEY-Rocky-$major"
rockdev --lang en bundle import --file /input/python.tar.gz --dir /tmp/offline-repo --yes
dnf -y --setopt=reposdir=/tmp/offline-repo --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install python3 python3-pip
python3 -m pip --version
python3 -m venv /tmp/offline-venv
/tmp/offline-venv/bin/python -c 'import sys; assert sys.prefix != sys.base_prefix; print("OFFLINE_VENV_PASS")'
echo OFFLINE_INSTALL_PASS
