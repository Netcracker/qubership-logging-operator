#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TEST_HOME_PATH=${TEST_HOME_PATH:-$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)}
TEST_CONTENT_PATH=${TEST_CONTENT_PATH:-${TEST_HOME_PATH}/build/fluent-pipeline}
FLUENTBIT_IMAGE=${FLUENTBIT_IMAGE:-docker.io/fluent/fluent-bit:5.1.0}
FLUENTD_IMAGE=${FLUENTD_IMAGE:-ghcr.io/netcracker/qubership-fluentd:1.19.3-1}
FLUENT_PIPELINE_TEST_IMAGE=${FLUENT_PIPELINE_TEST_IMAGE:-qubership-fluent-pipeline-tests:local}
INT_TESTS_IGNORE=${INT_TESTS_IGNORE:-}
CFG_TIMEOUT=${CFG_TIMEOUT:-2}
PARSE_TIMEOUT=${PARSE_TIMEOUT:-20}
PARSER_CONTRACT_TIMEOUT=${PARSER_CONTRACT_TIMEOUT:-5}

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
    chmod ugo+rwx "${contract_dir}"

    echo "=> Generate isolated Fluent Bit parser contract inputs and expectations"
    docker run --rm --security-opt label=disable --name fluent-config-replacer \
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

    sleep "${PARSER_CONTRACT_TIMEOUT}"
    ensure_running fluent-bit-parser-contract
    docker stop fluent-bit-parser-contract

    docker run --rm --security-opt label=disable --name fluent-pipeline-test \
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
    rm -rf "${TEST_CONTENT_PATH}"

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # Grant permissions
    chmod -R ugo+rw "${TEST_CONTENT_PATH}/"
    #chmod -R u+x fluent-pipeline-test/scripts
    # prepare fluentd configs
    echo "=> Prepare FluentD configurations"

    docker run --rm --security-opt label=disable --name fluent-config-replacer \
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

    # wait until prepare container stop
    echo "=> Waiting for stop container rendered FluentD configuration (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"

    # run FluentD
    echo "=> Run FluentD to read, parse and output processed logs"
    docker run --security-opt label=disable -d --name "${FLD_DOCKER_NAME}" \
        -e HOSTNAME=fake-fluent \
        -e K8S_NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/config/":/fluentd/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:rw \
        -v "${TEST_CONTENT_PATH}/output/":/fluentd-output:rw \
        "${FLUENTD_IMAGE}"

    echo "=> Waiting for FluentD start (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"
    ensure_running "${FLD_DOCKER_NAME}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until FluentD process all logs (${PARSE_TIMEOUT} seconds)"
    sleep "${PARSE_TIMEOUT}"

    echo "=> Stop and remove FluentD docker container"
    docker logs "${FLD_DOCKER_NAME}"
    docker stop "${FLD_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentD parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --name fluent-pipeline-test \
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
    rm -rf "${TEST_CONTENT_PATH}"

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # Grant permissions
    chmod -R 777 "${TEST_CONTENT_PATH}/"
    #chmod -R u+x fluent-pipeline-test/scripts
    # prepare fluent bit configs
    echo "=> Prepare FluentBit configurations"

    docker run --rm --security-opt label=disable --name fluent-config-replacer \
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

    # wait until prepare container stop
    echo "=> Waiting for stop container rendered FluentBit configuration (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"

    # run fluent bit
    echo "=> Run FluentBit to read, parse and output processed logs"
    docker run -d --name "${FLB_DOCKER_NAME}" \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:z \
        -v "${TEST_CONTENT_PATH}/output/":/fluentbit-output:z \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for FluentBit start (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"
    ensure_running "${FLB_DOCKER_NAME}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until FluentBit process all logs (${PARSE_TIMEOUT} seconds)"
    sleep "${PARSE_TIMEOUT}"

    echo "=> Stop and remove FluentBit docker container"
    docker logs "${FLB_DOCKER_NAME}"
    docker stop "${FLB_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --name fluent-pipeline-test \
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
    rm -rf "${TEST_CONTENT_PATH}"

    # Create test directories
    mkdir -p \
        "${TEST_CONTENT_PATH}/forwarder-config/" \
        "${TEST_CONTENT_PATH}/aggregator-config/" \
        "${TEST_CONTENT_PATH}/logs/var/log/audit/" \
        "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/" \
        "${TEST_CONTENT_PATH}/output/"
    create_empty_host_logs "${TEST_CONTENT_PATH}/logs"

    # Grant permissions
    chmod -R ugo+rw "${TEST_CONTENT_PATH}/"
    # prepare fluent bit configs
    echo "=> Prepare FluentBit configurations"

    docker run --rm --security-opt label=disable --name fluent-config-replacer \
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

    # wait until prepare container stop
    echo "=> Waiting for stop container rendered FluentBit configuration (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"

    docker run --rm --security-opt label=disable --name fluent-config-replacer \
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

    # wait until prepare container stop
    echo "=> Waiting for stop container rendered FluentBit configuration (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"

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

    echo "=> Waiting for FluentBit aggregator start (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"
    ensure_running "${FLB_AGR_DOCKER_NAME}"

    docker run -d --security-opt label=disable --name "${FLB_FRW_DOCKER_NAME}" \
        --network=fluent-net \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/forwarder-config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:rw \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for FluentBit forwarder start (${CFG_TIMEOUT} seconds)"
    sleep "${CFG_TIMEOUT}"
    ensure_running "${FLB_FRW_DOCKER_NAME}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/journal" "${TEST_CONTENT_PATH}/logs/var/log/journal"

    echo "=> Waiting until FluentBit process all logs (${PARSE_TIMEOUT} seconds)"
    sleep "${PARSE_TIMEOUT}"

    echo "=> Print FlintBit forwarder logs"
    docker logs "${FLB_FRW_DOCKER_NAME}"

    echo "=> Print FlintBit aggregator logs"
    docker logs "${FLB_AGR_DOCKER_NAME}"

    echo "=> Stop and remove FluentBit docker container"
    docker stop "${FLB_FRW_DOCKER_NAME}"
    docker stop "${FLB_AGR_DOCKER_NAME}"

    docker network rm fluent-net

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    docker run --rm --security-opt label=disable --name fluent-pipeline-test \
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
