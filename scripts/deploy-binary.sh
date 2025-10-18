#!/usr/bin/env bash

set -euo pipefail

serviceName="thrustOauth2idServer"
tarball="${serviceName}-binary.tar.gz"
remote_host="${1:-ec2-user@ericsg}"
remote_tmp="${2:-/tmp}"

if [[ ! -f "${tarball}" ]]; then
  echo "error: ${tarball} not found. run 'make binary-package' first." >&2
  exit 1
fi

ssh "${remote_host}" "mkdir -p '${remote_tmp}'"
scp "${tarball}" "${remote_host}":"${remote_tmp}/${tarball}"
ssh "${remote_host}" "cd '${remote_tmp}' && rm -rf ${serviceName}-binary && tar zxf '${tarball}'"
ssh "${remote_host}" "cd '${remote_tmp}' && bash ${serviceName}-binary/deploy.sh"

echo "Deployment to ${remote_host} complete."
