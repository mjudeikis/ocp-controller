#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../../../k8s.io/code-generator)}

verify="${VERIFY:-}"

for group in tagcontroller; do
  ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,lister,informer" \
    github.com/mjudeikis/ocp-controller/pkg/${group} \
    github.com/mjudeikis/ocp-controller/pkg/apis \
    "${group}:v1alpha1" \
    --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.txt \
    ${verify}
done
