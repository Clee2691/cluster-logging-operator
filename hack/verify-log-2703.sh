#!/bin/bash

# Author: Calvin Lee
#
# Purpose: A script to automate testing of LOG-2703: Collector DaemonSet is not removed when CLF is deleted for fluentd/vector only CL instance
# PR: https://github.com/openshift/cluster-logging-operator/pull/1559
#
# The user must have a working AWS cluster set up!
#
# This script will:
#   1. Deploy Elasticsearch operator & Cluster Logging Operator
#   2. Test LOG-2703
#   3. Remove generated resources for the test
#     1. Remove ClusterLogging instance
#     2. Undeploy ESO and CLO
#     3. Clean make files
#
# If ESO & CLO are already deployed, just run the script without flags
#
# Test Explanation
# 
# There are 5 steps to the test:
#   1. Create the ClusterLogForwarder instance to an external store
#         i. Forwards to AWS Cloudwatch
#   2. Provisions a vector only ClusterLogging instance with no default log store.
#   3. Verifies the collector daemonset is up and running by:
#         i.  Checking the daemonset's {.status.numberReady} == 6.
#         ii. Checking if ClusterLogging's CollectorDeadEnd condition is false
#   4. Removes the ClusterLogForwarder instance
#   5. Verifies deletion of the collector daemonset by:
#         i.  Checking for deletion of the daemonset
#         ii. Checking if ClusterLogging's CollectorDeadEnd condition is true

set -ueo pipefail

# colors can be commented out if unwanted
bold=$(tput bold)
red=$(tput setaf 1)
green=$(tput setaf 2)
yellow=$(tput setaf 3)
blue=$(tput setaf 4)
cyan=$(tput setaf 6)
reset=$(tput sgr0)

#Color    Value
#black     0 
#red       1 
#green     2 
#yellow    3 
#blue      4 
#magenta   5 
#cyan      6 
#white     7 

# leave these
read_color() {
  echo
  read -p "${bold}$1${reset}"
}

echo_bold() {
  echo
  echo -e "${bold}$1${reset}"
}

echo_green() {
  echo -e "${green}$1${reset}"
}

echo_yellow() {
  echo "${yellow}$1${reset}"
}

echo_cyan() {
  echo "${cyan}$1${reset}"
}

echo_blue() {
  echo "${blue}$1${reset}"
}

echo_red() {
  echo -e "${red}$1${reset}"
}

ME=$(basename "$0")
repo_dir="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )/.."
KUBECONFIG=~/tmp/installer/auth/kubeconfig
REGION=${REGION:-"us-west-1"}
SECRET_NAME=${SECRET_NAME:-"vector-cw-secret"}
COLLECTOR=${COLLECTOR:-"vector"}
LOG_PREFIX=$(whoami)-log-$(date +'%m%d')

DEBUG='info'

usage() {
  echo ${blue}
	cat <<-EOF

	Automate Testing of LOG-2703: Collector DaemonSet is not removed when CLF is deleted for fluentd/vector only CL instance

	Usage:
	  ${ME} [flags]

	Flags:
	  -c, --collector          Specify collector (default "${COLLECTOR}" or "fluentd")
	  -s, --secret-name        Specify AWS secret name (default "${SECRET_NAME}")
	  -r, --region             Specify AWS region (default "${REGION}")
	  -d, --debug              Use debug log level for install
	  -h, --help               Help for ${ME}
    -p, --undeploy-all      Undeploy ESO + CLO
    -v, --setup-ops       Set up operators before test
	EOF
  echo ${reset}
}

main() {
  SETUP_OPERATORS=0
  UNDEPLOY_ALL=0

  while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
      -r|--region)
        REGION="$2"
        shift # past argument
        shift # past value
        ;;
      -c|--collector)
        COLLECTOR="$2"
        shift # past argument
        shift # past value
        ;;
      -s|--secret-name)
        SECRET_NAME="$2"
        shift # past argument
        shift # past value
        ;;
      -d|--debug)
        DEBUG='debug'
        shift # past argument
        ;;
      -p|--undeploy-all)
        UNDEPLOY_ALL=1
        shift # past argument
        ;;
      -v|--setup-ops)
        SETUP_OPERATORS=1
        shift # past argument
        ;;
      -h|--help)
        usage && exit 0
        ;;
      *)
        echo -e "${red}Unknown flag $1${reset}" > /dev/stderr
        echo
        usage
        exit 1
        ;;
    esac
  done

  echo_red "This test assumes you have an AWS cluster running along with"
  echo_red "the Cluster Logging Operator and Elasticsearch Operator installed."
  echo_red "If not, run this script with -v or --setup-ops."
  echo_red "Additionally, you can delete all resources after the run with the -p flag"
  
  confirm

  # Set up the operators
  if [[ "${SETUP_OPERATORS}" -eq 1 ]]; then
    setup
    echo_green "Successfully deployed ESO and CLO"
  fi
  # Run the tests
  sleep 3
  test-log-2703

  # Delete operator instances and undeploy everything.
  if [[ "${UNDEPLOY_ALL}" -eq 1 ]]; then
    teardown
    echo_green "Resources cleaned up"
  fi

  exit 0
}

setup() {
  # Deploy Elasticsearch Operator
  # Deploy latest Cluster Logging Operator
  echo_cyan "==Deploying Elasticsearch Operator & Cluster Logging Operator=="

  export KUBECONFIG=${KUBECONFIG}
  oc login -u kubeadmin -p $(cat ~/tmp/installer/auth/kubeadmin-password)
  make deploy
  
  echo_cyan "==============================================================="
}

teardown() {
    echo_bold "=============Cleaning up test resources============="

    export KUBECONFIG=${KUBECONFIG}
    oc login -u kubeadmin -p $(cat ~/tmp/installer/auth/kubeadmin-password)
    delete_test_resources

    echo_bold "===============Undeploying ESO & CLO==============="
    make undeploy-all
    make clean
}

clusterlogging_instance_no_store() {
  cat <<-EOF
apiVersion: "logging.openshift.io/v1"
kind: "ClusterLogging"
metadata:
  name: "instance"
  namespace: "openshift-logging"
spec:
  managementState: Managed
  collection:
    logs:
      type: ${COLLECTOR}
      fluentd: {}
EOF
}

vector_cw_sec() {
  # Not a real aws user
  # key_id: test
  # access key: test
  cat <<-EOF
apiVersion: v1
kind: Secret
metadata:
  name: vector-cw-secret
  namespace: openshift-logging
data:
  aws_access_key_id: dGVzdAo=
  aws_secret_access_key: dGVzdAo=
EOF
}

create_clf() {
  echo -e "\nCreating & applying CloudWatch secret"
  vector_cw_sec | oc apply -f -
  echo

  echo -e "\nCreating logforwarder instance resource file"
  cat <<-EOF > hack/cw-logforwarder.yaml
---
apiVersion: "logging.openshift.io/v1"
kind: ClusterLogForwarder
metadata:
  name: instance
  namespace: openshift-logging
spec:
  outputs:
    - name: cw
      type: cloudwatch
      cloudwatch:
        groupBy: logType
        groupPrefix: ${LOG_PREFIX}
        region: ${REGION}
      secret:
        name: ${SECRET_NAME}
  pipelines:
    - name: all-logs
      inputRefs:
        - infrastructure
        - audit
        - application
      outputRefs:
        - cw
EOF

  echo -e "\nApply logforwarder yaml file"
  oc apply -f hack/cw-logforwarder.yaml
  echo
}

test-log-2703() {
  echo_bold "===============Pre-test Cleanup==============="
  export KUBECONFIG=${KUBECONFIG}
  oc login -u kubeadmin -p $(cat ~/tmp/installer/auth/kubeadmin-password)
  delete_test_resources

  echo -e "\n==============================================\n"
  echo_cyan "               TESTING LOG-2703"
  echo -e "\n==============================================\n"

  echo_bold "------Creating Custom Resource Instances (CLF/CL)------\n"
  echo "Creating ClusterLogForwarder instance to CloudWatch..."
  create_clf
  echo "Creating the CL with no store..."
  clusterlogging_instance_no_store | oc apply -f -
  echo

  echo_bold "---------------Collector Daemonset Status---------------\n"
  echo "Waiting for collector daemonset to be ready."
  # Verify daemonset is alive
  resource_watch 20 "jsonpath={.status.numberReady}=6" "daemonsets/collector" "daemonset.apps/collector condition met"
  isCollectorUp=$?

  # Continue only if the collector watch returns 0
  if [[ "${isCollectorUp}" -eq 0 ]]; then
    echo_green "\nCollector Daemonset is ready."
  fi

  resource_watch 20 "condition=CollectorDeadEnd=false" "ClusterLogging/instance" "clusterlogging.logging.openshift.io/instance condition met"
  collDeadFalse=$?
  if [[ "${collDeadFalse}" -eq 0 ]]; then
    echo_green "\nCollectorDeadEnd is False."
  fi

  echo_green "\nCheckpoint #1: Complete"

  sleep 5

  # Delete CLF
  echo_bold "----------------Testing Daemonset Removal----------------"
  echo -e "\nDeleting the ClusterLogForwarder Instance..."

  clf_del=$(oc delete -n openshift-logging ClusterLogForwarder instance)

  if [ "${clf_del}" = 'clusterlogforwarder.logging.openshift.io "instance" deleted' ]; then
    echo_green "ClusterLogForwarder successfully removed.\n"
  fi
  
  # Verify daemonset is gone
  echo "Verify collector daemonset is removed."
  resource_watch 20 "delete" "daemonsets/collector" "daemonset.apps/collector condition met"
  isDSDeleted=$?
  if [ "${isDSDeleted}" -eq 0 ]; then
    echo_green "Collector daemonset successfully removed."
  fi

  resource_watch 20 "condition=CollectorDeadEnd" "ClusterLogging/instance" "clusterlogging.logging.openshift.io/instance condition met"
  isCollectorDeadEnd=$?
  if [ "${isCollectorDeadEnd}" -eq 0 ]; then
    echo_green "CollectorDeadEnd is true."
  fi

  echo_green "\nPASS: Collector daemonset is deleted after CLF removal"
  echo "========================================================="

  _notify_send -t 5000 \
    "Test of LOG-2703" \
    "Status: PASSED"
}

resource_watch() {

  retries=$1
  for_cond=$2
  res_to_watch=$3
  oc_result=$4

  until [[ "$retries" -le "0" ]]; do
    result=$(oc wait -n openshift-logging --timeout=30s --for=${for_cond} ${res_to_watch}) || echo "Waiting for ${res_to_watch}"
    if [ "${result}" = "${oc_result}" ]; then
      return 0
    fi

    retries=$(( retries - 1))
    echo "${result} - remaining attempts: ${retries}"
    sleep 3
  done
  echo_red "FAIL: ${res_to_watch} failed to enter into the desired state."
  echo_red "Resources were not removed, remove manually."
  _notify_send -t 5000 \
    "Resource Watch" \
    "${res_to_watch} failed to enter into the desired state."
  exit 1
}

delete_test_resources() {
  echo -e "\nDeleting CL & CLF Resources"
  oc delete -n openshift-logging --ignore-not-found=true ClusterLogging instance 
  oc wait -n openshift-logging --timeout=10s --for=delete ClusterLogging/instance
  oc delete -n openshift-logging --ignore-not-found=true ClusterLogForwarder instance
  oc wait -n openshift-logging --timeout=10s --for=delete ClusterLogForwarder/instance
}

_notify_send() {
	notify-send "$@"
}

confirm() {
    echo
    read -p "Do you want to continue (y/N)? " CONT
    if [ "$CONT" != "y" ]; then
      echo "Okay, Exiting."
      exit 0
    fi
    echo
}

# ---
# Never put anything below this line. This is to prevent any partial execution
# if curl ever interrupts the download prematurely. In that case, this script
# will not execute since this is the last line in the script.
err_report() { echo "Error on line $1"; }
trap 'err_report $LINENO' ERR

main "$@"
