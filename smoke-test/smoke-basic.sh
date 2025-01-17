#!/usr/bin/env sh

CFCTL_CONFIG=${CFCTL_CONFIG:-"cfctl.yaml"}

set -e

. ./smoke.common.sh
trap cleanup EXIT

deleteCluster
createCluster

echo "* Starting apply"
../cfctl apply --config "${CFCTL_CONFIG}" --kubeconfig-out applykubeconfig --debug
echo "* Apply OK"

echo "* Verify hooks were executed on the host"
bootloose ssh root@manager0 -- grep -q hello apply.hook

echo "* Verify 'cfctl kubeconfig' output includes 'data' block"
../cfctl kubeconfig --config cfctl.yaml | grep -v -- "-data"

echo "* Run kubectl on controller"
bootloose ssh root@manager0 -- k0s kubectl get nodes

echo "* Downloading kubectl for local test"
downloadKubectl

echo "* Using the kubectl from apply"
./kubectl --kubeconfig applykubeconfig get nodes

echo "* Using cfctl kubecofig locally"
../cfctl kubeconfig --config cfctl.yaml >kubeconfig

echo "* Output:"
grep -v -- -data kubeconfig

echo "* Running kubectl"
./kubectl --kubeconfig kubeconfig get nodes
echo "* Done"
