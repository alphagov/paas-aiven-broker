#!/usr/bin/env bash
set -euo pipefail
FILEPATH="${1}"
PASS="$(which pass)"

if [[ -z "${PASS}" ]]; then
  echo "pass not found, please install it"
  exit 1
fi

function write_to_file() {
  local varname="${1}"
  local value="${2}"
  echo "${varname}=${value}" >>"${FILEPATH}"
}

if [[ ! -d "${PASSWORD_STORE_DIR}" ]]; then
  echo "paas-credentials not found, please clone it to ${PASSWORD_STORE_DIR}"
  rm "${FILEPATH}"
  exit 1
fi

truncate -s0 "${FILEPATH}"

write_to_file "EGRESS_IP" "$(curl --silent icanhazip.com)"

write_to_file "AIVEN_API_TOKEN" "$(${PASS} aiven.io/ci/api_token)"
write_to_file "AIVEN_PROJECT" "ci-testing"
write_to_file "AIVEN_CLOUD" "aws-eu-west-1"

write_to_file "AIVEN_USERNAME" "broker_username"
write_to_file "AIVEN_PASSWORD" "broker_password"

write_to_file "DEPLOY_ENV" "test"
write_to_file "BROKER_NAME" "test-broker"

echo "Created ${FILEPATH} with environment variables for integration tests"
