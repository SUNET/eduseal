#!/usr/bin/env bash

set -e

. /opt/sunet/venv/bin/activate

chmod +x -R /opt/sunet/src/eduseal

app_entrypoint="eduseal.validator.run"
app_name="eduseal_validator"
base_dir="/opt/sunet"
project_dir="${base_dir}/src"

#mkdir /etc/ssl/certs

# set PYTHONPATH if it is not already set using Docker environment
export PYTHONPATH=${PYTHONPATH-${project_dir}}
echo "PYTHONPATH=${PYTHONPATH}"

export PYTHONPATH="${PYTHONPATH:+${PYTHONPATH}:}/opt/sunet/venv"

echo ""
echo "$0: Starting ${app_name}"

exec start-stop-daemon --start -c root:root --exec \
     /opt/sunet/venv/bin/python \
     --pidfile eduseal_validator.pid --\
     -m ${app_entrypoint}