#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
TEST_HOME_PATH=${TEST_HOME_PATH:-$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)}
TEST_CONTENT_PATH=${TEST_CONTENT_PATH:-${TEST_HOME_PATH}/build/fluent-pipeline}
FLUENTBIT_IMAGE=${FLUENTBIT_IMAGE:-docker.io/fluent/fluent-bit:5.1.0}
FLUENTD_IMAGE=${FLUENTD_IMAGE:-ghcr.io/netcracker/qubership-fluentd:1.19.3-1}
FLUENT_PIPELINE_TEST_IMAGE=${FLUENT_PIPELINE_TEST_IMAGE:-qubership-fluent-pipeline-tests:local}
INT_TESTS_IGNORE=${INT_TESTS_IGNORE:-}
# The helper container writes the rendered configuration and the generated logs to bind mounts.
# Running it as the calling user keeps every generated file owned by that user, so neither the
# logging agents, which run as root, nor the cleanup of the next run need loose permissions.
HELPER_USER=${HELPER_USER:-$(id -u):$(id -g)}
# Readiness and completion are observed, not timed. STARTUP_TIMEOUT bounds the wait for an agent to open its
# inputs, and OUTPUT_TIMEOUT bounds the wait for the processed records to reach the output file. Records arrive
# in flushes, so a count that holds for OUTPUT_SETTLE_POLLS one-second polls is taken as final.
STARTUP_TIMEOUT=${STARTUP_TIMEOUT:-30}
OUTPUT_TIMEOUT=${OUTPUT_TIMEOUT:-60}
OUTPUT_SETTLE_POLLS=${OUTPUT_SETTLE_POLLS:-3}

cleanup() {
    docker rm -f fluentd fluent-bit fluent-bit-forwarder fluent-bit-aggregator fluent-bit-parser-contract fluent-config-replacer \
        fluent-pipeline-test >/dev/null 2>&1 || true
    docker network rm fluent-net >/dev/null 2>&1 || true
}

run_parser_contracts() {
    rendered_config_dir=$1
    suite_name=$2
    contract_dir="${TEST_CONTENT_PATH}/parser-contracts-${suite_name}"
    mkdir -p "${contract_dir}"

    echo "=> Generate isolated Fluent Bit parser contract inputs and expectations"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-config-replacer \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/parser-cases.json":/parser-contracts/cases.json:ro \
        -v "${rendered_config_dir}":/rendered-config:ro \
        -v "${contract_dir}":/parser-contracts/generated:rw \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -stage prepare-parser-contracts \
        -parserCases /parser-contracts/cases.json \
        -parsers /rendered-config/parsers.conf \
        -parserTarget /parser-contracts/generated \
        -loglevel warn

    echo "=> Run isolated Fluent Bit parser contracts"
    docker run -d --security-opt label=disable --name fluent-bit-parser-contract \
        -v "${contract_dir}":/fluent-bit/etc:ro \
        -v "${contract_dir}/input":/parser-input:ro \
        -v "${contract_dir}/output":/parser-output:rw \
        "${FLUENTBIT_IMAGE}"

    wait_for_records "${contract_dir}/output/output-log" "$(count_expected_records "${contract_dir}/expected")" \
        fluent-bit-parser-contract
    docker stop fluent-bit-parser-contract

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-pipeline-test \
        -v "${contract_dir}/output":/output-logs/actual:ro \
        -v "${contract_dir}/expected":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbit \
        -stage test \
        -loglevel warn
}

trap cleanup EXIT INT TERM

ensure_running() {
    container_name=$1
    state=$(docker inspect --format '{{.State.Running}}' "${container_name}" 2>/dev/null || true)
    if [ "${state}" != "true" ]; then
        echo "Container ${container_name} exited during startup" >&2
        docker logs "${container_name}" >&2 || true
        return 1
    fi
}

# reset_test_content empties the content directory of the previous run. The logging agents run as root and
# can leave files the calling user cannot delete, such as a Fluentd buffer directory that was never flushed;
# those are removed through the helper image running as root.
reset_test_content() {
    rm -rf "${TEST_CONTENT_PATH}" 2>/dev/null || true
    if [ -e "${TEST_CONTENT_PATH}" ]; then
        docker run --rm --security-opt label=disable --user 0 --entrypoint rm \
            -v "$(dirname "${TEST_CONTENT_PATH}")":/content:rw \
            "${FLUENT_PIPELINE_TEST_IMAGE}" -rf "/content/$(basename "${TEST_CONTENT_PATH}")"
    fi
}

# wait_for_log polls a container's log for the line that shows it is ready. The agents log it at the info level,
# which the test custom resources set. A container that exits or stays silent past STARTUP_TIMEOUT fails the run.
wait_for_log() {
    container_name=$1
    pattern=$2
    waited=0
    while [ "${waited}" -lt "${STARTUP_TIMEOUT}" ]; do
        if docker logs "${container_name}" 2>&1 | grep -qE -- "${pattern}"; then
            return 0
        fi
        ensure_running "${container_name}" || return 1
        sleep 1
        waited=$((waited + 1))
    done
    echo "Timed out after ${STARTUP_TIMEOUT} seconds waiting for ${container_name} to log '${pattern}'" >&2
    docker logs --tail 20 "${container_name}" >&2 || true
    return 1
}

# wait_for_system_inputs waits until the Fluent Bit tail inputs have opened the system and audit files, so the
# lines appended afterwards are read; the inputs skip whatever a file held before they opened it.
wait_for_system_inputs() {
    container_name=$1
    for host_log in /var/log/syslog /var/log/audit/audit.log /var/log/kubernetes/audit/audit.log; do
        wait_for_log "${container_name}" "inotify_fs_add\(\).*name=${host_log}\$"
    done
}

# count_expected_records counts the records the comparison reads from a directory of expected files; each record
# carries one _test block.
count_expected_records() {
    cat "$1"/*.log.json | grep -c '"_test"'
}

# wait_for_records polls the output file until it holds at least the expected number of records and the count has
# held for OUTPUT_SETTLE_POLLS polls. When OUTPUT_TIMEOUT passes it reports the count it saw and returns, because
# the comparison stage names the missing records better than an early exit would. A container that has exited
# ends the wait at once.
wait_for_records() {
    output_file=$1
    expected_records=$2
    container_name=$3
    waited=0
    previous=-1
    settled=0
    while [ "${waited}" -lt "${OUTPUT_TIMEOUT}" ]; do
        current=$(grep -c '^{' "${output_file}" 2>/dev/null || true)
        current=${current:-0}
        if [ "${current}" -ge "${expected_records}" ] && [ "${current}" -eq "${previous}" ]; then
            settled=$((settled + 1))
        else
            settled=0
        fi
        if [ "${settled}" -ge "${OUTPUT_SETTLE_POLLS}" ]; then
            echo "=> ${current} records in ${output_file} after ${waited} seconds"
            return 0
        fi
        ensure_running "${container_name}" || return 0
        previous=${current}
        sleep 1
        waited=$((waited + 1))
    done
    echo "Timed out after ${OUTPUT_TIMEOUT} seconds waiting for ${expected_records} records in ${output_file}; last count ${previous}" >&2
}

# Use sed to copy data from test data in files that fluent should read
add_lines() {
    input_file=$1
    output_file=$2
    echo "emulate logs generation in ${output_file} file (data will copy from ${input_file})"
    cat "${input_file}" >>"${output_file}"
}

# File discovery latency is not part of parser validation. Reduce the rendered production interval so late-created
# system and audit fixtures reach the same filters without adding a minute to every CI scenario.
speed_up_file_discovery() {
    config_dir=$1
    find "${config_dir}" -type f -exec sed -i \
        -e 's/Refresh_Interval   60/Refresh_Interval   1/g' \
        -e 's/refresh_interval 60/refresh_interval 1/g' {} +
}

create_empty_host_logs() {
    logs_root=$1
    touch \
        "${logs_root}/var/log/audit/audit.log" \
        "${logs_root}/var/log/kubernetes/audit/audit.log" \
        "${logs_root}/var/log/syslog" \
        "${logs_root}/var/log/messages" \
        "${logs_root}/var/log/journal"
}

###################################################################################################
# Run FluentD DaemonSet test logic
###################################################################################################
run_fluentd_test_logic() {
    FLD_DOCKER_NAME="fluentd"
    # Remove test directories from previous run
    echo "=> Prepare test environment and test data"
    reset_test_content

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # prepare fluentd configs
    echo "=> Prepare FluentD configurations"

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-config-replacer \
        -v "${TEST_HOME_PATH}/controllers/fluentd/fluentd.configmap/":/config-templates.d:ro \
        -v "${TEST_CONTENT_PATH}/config/":/configuration.d:rw \
        -v "${TEST_CONTENT_PATH}/logs/":/testdata:rw \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/fluentd.yaml":/assets/fluentd.yaml:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/":/logs:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentd \
        -cr /assets/fluentd.yaml \
        -stage prepare \
        -loglevel warn \
        -ignore "${INT_TESTS_IGNORE}"

    speed_up_file_discovery "${TEST_CONTENT_PATH}/config"

    # run FluentD
    echo "=> Run FluentD to read, parse and output processed logs"
    docker run --security-opt label=disable -d --name "${FLD_DOCKER_NAME}" \
        -e HOSTNAME=fake-fluent \
        -e K8S_NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/config/":/fluentd/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:rw \
        -v "${TEST_CONTENT_PATH}/output/":/fluentd-output:rw \
        "${FLUENTD_IMAGE}"

    echo "=> Waiting for FluentD to start"
    wait_for_log "${FLD_DOCKER_NAME}" 'fluentd worker is now running'

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until FluentD writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/fake-fluent.log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentd")" "${FLD_DOCKER_NAME}"

    echo "=> Stop and remove FluentD docker container"
    docker logs "${FLD_DOCKER_NAME}"
    docker stop "${FLD_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentD parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-pipeline-test \
        -v "${TEST_CONTENT_PATH}/output/":/output-logs/actual:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentd/":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -stage test \
        -agent fluentd \
        -ignore "${INT_TESTS_IGNORE}"

}

###################################################################################################
# Run FluentBit DaemonSet test logic
###################################################################################################
run_fluentbit_test_logic() {
    FLB_DOCKER_NAME="fluent-bit"
    # Remove test directories from previous run
    echo "=> Prepare test environment and test data"
    reset_test_content

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # prepare fluent bit configs
    echo "=> Prepare FluentBit configurations"

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-config-replacer \
        -v "${TEST_HOME_PATH}/controllers/fluentbit/fluentbit.configmap/":/config-templates.d/:ro \
        -v "${TEST_CONTENT_PATH}/config/":/configuration.d/:z \
        -v "${TEST_CONTENT_PATH}/logs/":/testdata:z \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/fluentbit.yaml":/assets/fluentbit.yaml:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/":/logs:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbit \
        -cr /assets/fluentbit.yaml \
        -stage prepare \
        -loglevel warn \
        -ignore "${INT_TESTS_IGNORE}"

    speed_up_file_discovery "${TEST_CONTENT_PATH}/config"

    # run fluent bit
    echo "=> Run FluentBit to read, parse and output processed logs"
    docker run -d --name "${FLB_DOCKER_NAME}" \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:z \
        -v "${TEST_CONTENT_PATH}/output/":/fluentbit-output:z \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for FluentBit to open the system and audit inputs"
    wait_for_system_inputs "${FLB_DOCKER_NAME}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until FluentBit writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/output-log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit")" "${FLB_DOCKER_NAME}"

    echo "=> Stop and remove FluentBit docker container"
    docker logs "${FLB_DOCKER_NAME}"
    docker stop "${FLB_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-pipeline-test \
        -v "${TEST_CONTENT_PATH}/output/":/output-logs/actual:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit/":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbit \
        -stage test \
        -ignore "${INT_TESTS_IGNORE}"

    run_parser_contracts "${TEST_CONTENT_PATH}/config" daemonset

}

###################################################################################################
# Run FluentBit DaemonSet + FluentBit StatefulSet (aka HA deployment) test logic
###################################################################################################
run_fluentbit_ha_test_logic() {
    FLB_FRW_DOCKER_NAME="fluent-bit-forwarder"
    FLB_AGR_DOCKER_NAME="fluent-bit-aggregator"
    # Remove test directories from previous run
    echo "=> Prepare test environment and test data"
    reset_test_content

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/forwarder-config/" \
        "${TEST_CONTENT_PATH}/aggregator-config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # prepare fluent bit configs
    echo "=> Prepare FluentBit configurations"

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-config-replacer \
        -v "${TEST_HOME_PATH}/controllers/fluentbit-forwarder-aggregator/forwarder.configmap/":/config-templates.d:ro \
        -v "${TEST_CONTENT_PATH}/forwarder-config/":/configuration.d:rw \
        -v "${TEST_CONTENT_PATH}/logs/":/testdata:rw \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/fluentbit-ha.yaml":/assets/fluentbit-ha.yaml:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/":/logs:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbitha \
        -cr /assets/fluentbit-ha.yaml \
        -stage prepare \
        -loglevel warn \
        -ignore "${INT_TESTS_IGNORE}"

    speed_up_file_discovery "${TEST_CONTENT_PATH}/forwarder-config"

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-config-replacer \
        -v "${TEST_HOME_PATH}/controllers/fluentbit-forwarder-aggregator/aggregator.configmap/":/config-templates.d:ro \
        -v "${TEST_CONTENT_PATH}/aggregator-config/":/configuration.d:rw \
        -v "${TEST_CONTENT_PATH}/logs/":/testdata:rw \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/fluentbit-ha.yaml":/assets/fluentbit-ha.yaml:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/":/logs:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbitha \
        -cr /assets/fluentbit-ha.yaml \
        -stage prepare \
        -loglevel warn \
        -ignore "${INT_TESTS_IGNORE}"

    speed_up_file_discovery "${TEST_CONTENT_PATH}/aggregator-config"

    # run fluent bit
    echo "=> Run FluentBit to read, parse and output processed logs"

    docker network create fluent-net

    docker run -d --security-opt label=disable --name "${FLB_AGR_DOCKER_NAME}" \
        --network=fluent-net \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/aggregator-config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/output/":/fluentbit-output:rw \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for the FluentBit aggregator to listen for the forwarder"
    wait_for_log "${FLB_AGR_DOCKER_NAME}" 'input:forward.*listening on'

    docker run -d --security-opt label=disable --name "${FLB_FRW_DOCKER_NAME}" \
        --network=fluent-net \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/forwarder-config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:rw \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for the FluentBit forwarder to open the system and audit inputs"
    wait_for_system_inputs "${FLB_FRW_DOCKER_NAME}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until the FluentBit aggregator writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/output-log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit-ha")" "${FLB_AGR_DOCKER_NAME}"

    echo "=> Print FlintBit forwarder logs"
    docker logs "${FLB_FRW_DOCKER_NAME}"

    echo "=> Print FlintBit aggregator logs"
    docker logs "${FLB_AGR_DOCKER_NAME}"

    echo "=> Stop and remove FluentBit docker container"
    docker stop "${FLB_FRW_DOCKER_NAME}"
    docker stop "${FLB_AGR_DOCKER_NAME}"

    docker network rm fluent-net

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name fluent-pipeline-test \
        -v "${TEST_CONTENT_PATH}/output/":/output-logs/actual:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit-ha/":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbitha \
        -stage test \
        -ignore "${INT_TESTS_IGNORE}"

    run_parser_contracts "${TEST_CONTENT_PATH}/forwarder-config" forwarder
}

###################################################################################################
# Entrypoint
###################################################################################################

case ${1:-} in

'fluentd')
    run_fluentd_test_logic
    ;;

'fluentbit')
    run_fluentbit_test_logic
    ;;

'fluentbit-ha')
    run_fluentbit_ha_test_logic
    ;;

*)
    echo "Usage: $0 {fluentd|fluentbit|fluentbit-ha}" >&2
    exit 2
    ;;

esac
