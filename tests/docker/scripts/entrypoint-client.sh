#!/bin/sh
set -e

until wget -q -O /dev/null --header="Host: server" http://server/ 2>/dev/null; do
  sleep 1
done
sleep 1

export FWDX_SERVER=http://server
export FWDX_AGENT_NAME=${FWDX_AGENT_NAME:-e2e-agent}

fwdx login
fwdx whoami
fwdx tunnel create -l app:8080 -s myapp --name myapp
exec fwdx tunnel start myapp --watch
