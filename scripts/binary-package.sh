#!/usr/bin/env bash

set -euo pipefail

serviceName="thrustOauth2idServer"
packageDir="${serviceName}-binary"
archive="${packageDir}.tar.gz"
binaryPath="cmd/${serviceName}/${serviceName}"

rm -rf "${packageDir}" "${archive}"
mkdir -p "${packageDir}/configs"

if [[ ! -f "${binaryPath}" ]]; then
  echo "error: ${binaryPath} does not exist. run 'make build' first." >&2
  exit 1
fi

cp -f deployments/binary/run.sh "${packageDir}"
chmod +x "${packageDir}/run.sh"

cp -f deployments/binary/deploy.sh "${packageDir}"
chmod +x "${packageDir}/deploy.sh"

cp -f "${binaryPath}" "${packageDir}"
cp -f "configs/${serviceName}.yml" "${packageDir}/configs"
cp -f "configs/${serviceName}_cc.yml" "${packageDir}/configs"

tar zcf "${archive}" "${packageDir}"
rm -rf "${packageDir}"

echo ""
echo "package binary successfully, output file = ${archive}"
