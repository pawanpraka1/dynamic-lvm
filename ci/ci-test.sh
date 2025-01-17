#!/usr/bin/env bash

# This ci scripts expects the kubernetes worker to be running on this very node.
# The lvm container image must be pre-loaded into the node, example: ./ci-test.sh load
# Alternatively, you can run the tests using ./ci-test.sh run -b

set -e

SCRIPT_DIR="$(dirname "$(realpath "${BASH_SOURCE[0]:-"$0"}")")"
SNAP_CLASS="$(realpath deploy/sample/lvmsnapclass.yaml)"
export OPENEBS_NAMESPACE=${OPENEBS_NAMESPACE:-openebs}
export TEST_DIR="$SCRIPT_DIR"/../tests

# foreign systemid for the testing environment.
FOREIGN_LVM_SYSTEMID="openebs-ci-test-system"
FOREIGN_LVM_CONFIG="global{system_id_source=lvmlocal}local{system_id=${FOREIGN_LVM_SYSTEMID}}"
CRDS_TO_DELETE_ON_CLEANUP="lvmnodes.local.openebs.io lvmsnapshots.local.openebs.io lvmvolumes.local.openebs.io volumesnapshotclasses.snapshot.storage.k8s.io volumesnapshotcontents.snapshot.storage.k8s.io volumesnapshots.snapshot.storage.k8s.io"

help() {
  cat <<EOF >&2
Usage: $(basename "${0}") [COMMAND] [OPTIONS]

Commands:
  run                          Run the tests.
  load                         Build and load the image into the K8s cluster.
  clean                        Clean the leftovers.

Options:
  -h, --help                   Display this text.

Options for run:
  -r, --reset                  Clean before running the tests.
  -x, --no-cleanup             Don't cleanup after running the tests.
  -b, --build-always           Build and load the images before running the tests. [ By default image is built if not present only ]

Examples:
  $(basename "${0}") run -rxb
EOF
}

echo_err() {
  echo -e "ERROR: $1" >&2
}

needs_help() {
  [ -n "$1" ] && echo_err "$1\n"
  help
  exit 1
}

die() {
  echo_err "FATAL: $1"
  exit 1
}

# Clean up generated resources for successive tests.
cleanup_loopdev() {
  losetup -l | grep '(deleted)' | awk '{print $1}' \
    | while IFS= read -r disk
      do
        sudo losetup -d "${disk}"
      done
}

cleanup_foreign_lvmvg() {
  if [ -f /tmp/openebs_ci_foreign_disk.img ]
  then
    sudo vgremove foreign_lvmvg --config="${FOREIGN_LVM_CONFIG}" -y || true
    sudo rm /tmp/openebs_ci_foreign_disk.img
  fi
  cleanup_loopdev
}

# Clean up loop devices and vgs created by the ginkgo lvm_utils.go
cleanup_ginkgo_loop_lvm() {
  for device in $(losetup -l -J | jq -r '.loopdevices[]|select(."back-file" | startswith("/tmp/openebs_lvm_localpv_disk_"))' | jq -r '.name'); do
    echo "Found stale loop device: $device"

    sudo "$(which vgremove)" -y --select="pv_name=$device" || :
    sudo losetup -d "$device" 2>/dev/null || :
  done
}

cleanup() {
  set +e

  echo "Cleaning up test resources"

  cleanup_foreign_lvmvg
  cleanup_ginkgo_loop_lvm

  if kubectl get nodes 2>/dev/null; then
    kubectl delete pvc -n "$OPENEBS_NAMESPACE" lvmpv-pvc

    sleep 3

    # shellcheck disable=SC2068
    for cr in ${CRDS_TO_DELETE_ON_CLEANUP[@]}; do
      kubectl delete "$cr" -n "$OPENEBS_NAMESPACE" --all
    done

    if helm uninstall lvm-localpv -n "$OPENEBS_NAMESPACE" --ignore-not-found --timeout 1m --wait; then
      # shellcheck disable=SC2086
      kubectl delete crds $CRDS_TO_DELETE_ON_CLEANUP
      kubectl delete -f "${SNAP_CLASS}"
    fi
  fi

  set -e
}

dumpAgentLogs() {
  NR=$1
  AgentPOD=$(kubectl get pods -l app=openebs-lvm-node -o jsonpath='{.items[0].metadata.name}' -n "$OPENEBS_NAMESPACE")
  kubectl describe po "$AgentPOD" -n "$OPENEBS_NAMESPACE"
  printf "\n\n"
  kubectl logs --tail="${NR}" "$AgentPOD" -n "$OPENEBS_NAMESPACE" -c openebs-lvm-plugin
  printf "\n\n"
}

dumpControllerLogs() {
  NR=$1
  ControllerPOD=$(kubectl get pods -l app=openebs-lvm-controller -o jsonpath='{.items[0].metadata.name}' -n "$OPENEBS_NAMESPACE")
  kubectl describe po "$ControllerPOD" -n "$OPENEBS_NAMESPACE"
  printf "\n\n"
  kubectl logs --tail="${NR}" "$ControllerPOD" -n "$OPENEBS_NAMESPACE" -c openebs-lvm-plugin
  printf "\n\n"
}

dump_logs() {
  sudo pvscan --cache

  sudo lvdisplay

  sudo vgdisplay

  echo "******************** LVM Controller logs***************************** "
  dumpControllerLogs 1000

  echo "********************* LVM Agent logs *********************************"
  dumpAgentLogs 1000

  echo "get all the pods"
  kubectl get pods -owide --all-namespaces

  echo "get pvc and pv details"
  kubectl get pvc,pv -oyaml --all-namespaces

  echo "get snapshot details"
  kubectl get volumesnapshot.snapshot -oyaml --all-namespaces

  echo "get sc details"
  kubectl get sc --all-namespaces -oyaml

  echo "get lvm volume details"
  kubectl get lvmvolumes.local.openebs.io -n "$OPENEBS_NAMESPACE" -oyaml

  echo "get lvm snapshot details"
  kubectl get lvmsnapshots.local.openebs.io -n "$OPENEBS_NAMESPACE" -oyaml
}

isPodReady(){
  [ "$(kubectl get po "$1" -o 'jsonpath={.status.conditions[?(@.type=="Ready")].status}' -n "$OPENEBS_NAMESPACE")" = 'True' ]
}

isDriverReady(){
  for pod in $lvmDriver;do
    isPodReady "$pod" || return 1
  done
}

waitForLVMDriver() {
  period=120
  interval=1

  i=0
  while [ "$i" -le "$period" ]; do
    lvmDriver="$(kubectl get pods -l role=openebs-lvm -o 'jsonpath={.items[*].metadata.name}' -n "$OPENEBS_NAMESPACE")"
    if isDriverReady "$lvmDriver"; then
      return 0
    fi

    i=$(( i + interval ))
    echo "Waiting for lvm-driver to be ready..."
    sleep "$interval"
  done

  echo "Waited for $period seconds, but all pods are not ready yet."
  return 1
}

run() {
  # setup a foreign lvm to test
  cleanup_foreign_lvmvg
  truncate -s 100G /tmp/openebs_ci_foreign_disk.img
  foreign_disk="$(sudo losetup -f /tmp/openebs_ci_foreign_disk.img --show)"
  sudo pvcreate "${foreign_disk}"
  sudo vgcreate foreign_lvmvg "${foreign_disk}" --config="${FOREIGN_LVM_CONFIG}"

  # install snapshot and thin volume module for lvm
  sudo modprobe dm-snapshot
  sudo modprobe dm_thin_pool

  # Set the configuration for thin pool autoextend in lvm.conf
  # WARNING: this is modifying the host's settings!!!
  sudo sed -i '/^[^#]*thin_pool_autoextend_threshold/ s/= .*/= 50/' /etc/lvm/lvm.conf
  sudo sed -i '/^[^#]*thin_pool_autoextend_percent/ s/= .*/= 20/' /etc/lvm/lvm.conf

  # Prepare env for running BDD tests
  helm install lvm-localpv ./deploy/helm/charts -n "$OPENEBS_NAMESPACE" --create-namespace --set lvmPlugin.image.pullPolicy=Never --set analytics.enabled=false
  kubectl apply -f "${SNAP_CLASS}"

  # wait for lvm-driver to be up
  waitForLVMDriver

  cd "$TEST_DIR"

  kubectl get po -n "$OPENEBS_NAMESPACE"

  echo "running ginkgo test case"

  if ! ginkgo -v -coverprofile=bdd_coverage.txt -covermode=atomic; then
    dump_logs
    [ "$CLEAN_AFTER" = "true" ] && cleanup
    exit 1
  fi

  printf "\n\n######### All test cases passed #########\n\n"
}

load_image() {
  make lvm-driver-image
  if [ "${CI_K3S:-}" = "true" ]; then
    docker save openebs/lvm-driver | ctr images import -
  fi
}

maybe_load_image() {
  if [ "$BUILD_ALWAYS" = "true" ]; then
    load_image
    return 0
  fi

  local id
  id=$(crictl image --output json | jq --arg image "$(make image-ref -s -C "$SCRIPT_DIR"/.. 2>/dev/null)" '.images[]|select(.repoTags[0] == $image)|.id')
  if [ -n "$id" ]; then
    return 0
  fi

  load_image
}

# allow override
if [ -z "${KUBECONFIG}" ]
then
  export KUBECONFIG="${HOME}/.kube/config"
fi

COMMAND=
CLEAN_BEFORE="false"
CLEAN_AFTER="true"
BUILD_ALWAYS="false"

while test $# -gt 0; do
  arg="$1"
  case "$arg" in
    run | clean)
      [ -n "$COMMAND" ] && needs_help "Can't specify two commands"
      COMMAND="$1"
      ;;
    -r | --reset)
      CLEAN_BEFORE="true"
      ;;
    -x | --no-cleanup)
      CLEAN_AFTER="false"
      ;;
    -b | --build-always)
      BUILD_ALWAYS="true"
      ;;
    -h | --help)
      needs_help
      ;;
    -*)
      singleLetterOpts="${1:1}"
      while [ -n "$singleLetterOpts" ]; do
        case "${singleLetterOpts:0:1}" in
          r)
            CLEAN_BEFORE="true"
            ;;
          x)
            CLEAN_AFTER="false"
            ;;
          b)
            BUILD_ALWAYS="true"
            ;;
          *)
            needs_help "Unrecognized argument $singleLetterOpts"
            ;;
        esac
        singleLetterOpts="${singleLetterOpts:1}"
      done
      ;;
    *)
      needs_help "Unrecognized argument $1"
      ;;
  esac
  shift
done

case "$COMMAND" in
  clean)
    cleanup
    ;;
  load)
    load_image
    ;;
  run)
    # trap "cleanup 2>/dev/null" EXIT
    [ "$CLEAN_BEFORE" = "true" ] && cleanup

    maybe_load_image
    run

    [ "$CLEAN_AFTER" = "true" ] && cleanup
    ;;
  *)
    needs_help "Missing Command"
    ;;
esac
