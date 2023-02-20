#!/usr/bin/env bash
set -euo pipefail
FILEPATH="${1:-integration.env}"
truncate -s0 "${FILEPATH}"

function write_to_file() {
  local varname="${1}"
  local value="${2}"
  echo "${varname}=\"${value}\"" >> "${FILEPATH}"
}

function paas-pass() {
  PASSWORD_STORE_DIR="${HOME}/.paas-pass" pass "${@}"
}

write_to_file "EGRESS_IP"       "$(curl --silent icanhazip.com)"

write_to_file "AIVEN_API_TOKEN" "$(paas-pass aiven.io/ci/api_token)"
write_to_file "AIVEN_PROJECT"   "ci-testing"
write_to_file "AIVEN_CLOUD"     "aws-eu-west-1"

write_to_file "AIVEN_USERNAME"  "broker_username"
write_to_file "AIVEN_PASSWORD"  "broker_password"

write_to_file "DEPLOY_ENV"      "test"
write_to_file "BROKER_NAME"     "test-broker"