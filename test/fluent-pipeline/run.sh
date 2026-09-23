#!/bin/sh

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
TEST_HOME_PATH=${TEST_HOME_PATH:-$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)}
TEST_CONTENT_PATH=${TEST_CONTENT_PATH:-${TEST_HOME_PATH}/build/fluent-pipeline}
CONTENT_MARKER=.fluent-pipeline-test-content
CONTENT_MARKER_VALUE='owned by test/fluent-pipeline/run.sh'
RESOURCE_PREFIX="fluent-pipeline-$(date +%s)-$$"
FLUENTD_CONTAINER="${RESOURCE_PREFIX}-fluentd"
FLUENTBIT_CONTAINER="${RESOURCE_PREFIX}-fluent-bit"
FLUENTBIT_FORWARDER_CONTAINER="${RESOURCE_PREFIX}-forwarder"
FLUENTBIT_AGGREGATOR_CONTAINER="${RESOURCE_PREFIX}-aggregator"
PARSER_CONTRACT_CONTAINER="${RESOURCE_PREFIX}-parser-contract"
CONFIG_REPLACER_CONTAINER="${RESOURCE_PREFIX}-config-replacer"
COMPARISON_CONTAINER="${RESOURCE_PREFIX}-comparison"
FLUENTBIT_RENDER_CONTAINER="${RESOURCE_PREFIX}-fluent-bit-render"
FLUENTD_RENDER_CONTAINER="${RESOURCE_PREFIX}-fluentd-render"
KUBE_API_NAME="${RESOURCE_PREFIX}-kube-api"
CLEANUP_CONTAINER="${RESOURCE_PREFIX}-cleanup"
NETWORK_NAME="${RESOURCE_PREFIX}-net"
# The agents under test are the images the chart deploys; Renovate keeps the two in step through
# the annotations below, the same way it does for charts/.../templates/_helpers.tpl.
# renovate: datasource=docker depName=fluent/fluent-bit
FLUENTBIT_IMAGE=${FLUENTBIT_IMAGE:-docker.io/fluent/fluent-bit:5.1.2}
# renovate: datasource=github-releases depName=Netcracker/qubership-fluentd versioning=loose
FLUENTD_IMAGE=${FLUENTD_IMAGE:-ghcr.io/netcracker/qubership-fluentd:1.19.3-2}
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
# The log_to_metrics filters flush every 20 seconds, so the exporter serves nothing before that.
METRICS_TIMEOUT=${METRICS_TIMEOUT:-60}
OUTPUT_TIMEOUT=${OUTPUT_TIMEOUT:-60}
OUTPUT_SETTLE_POLLS=${OUTPUT_SETTLE_POLLS:-3}

cleanup() {
    docker rm -f "${FLUENTD_CONTAINER}" "${FLUENTBIT_CONTAINER}" "${FLUENTBIT_FORWARDER_CONTAINER}" \
        "${FLUENTBIT_AGGREGATOR_CONTAINER}" "${PARSER_CONTRACT_CONTAINER}" "${CONFIG_REPLACER_CONTAINER}" \
        "${COMPARISON_CONTAINER}" "${FLUENTBIT_RENDER_CONTAINER}" "${FLUENTD_RENDER_CONTAINER}" \
        "${KUBE_API_NAME}" "${CLEANUP_CONTAINER}" >/dev/null 2>&1 || true
    docker network rm "${NETWORK_NAME}" >/dev/null 2>&1 || true
}

run_parser_contracts() {
    rendered_config_dir=$1
    suite_name=$2
    contract_dir="${TEST_CONTENT_PATH}/parser-contracts-${suite_name}"
    mkdir -p "${contract_dir}"

    echo "=> Generate isolated Fluent Bit parser contract inputs and expectations"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
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
    docker run -d --security-opt label=disable --name "${PARSER_CONTRACT_CONTAINER}" \
        -v "${contract_dir}":/fluent-bit/etc:ro \
        -v "${contract_dir}/input":/parser-input:ro \
        -v "${contract_dir}/output":/parser-output:rw \
        "${FLUENTBIT_IMAGE}"

    wait_for_records "${contract_dir}/output/output-log" "$(count_expected_records "${contract_dir}/expected")" \
        "${PARSER_CONTRACT_CONTAINER}"
    docker stop "${PARSER_CONTRACT_CONTAINER}"

    run_comparison docker run --rm --security-opt label=disable --user "${HELPER_USER}" \
        --name "${COMPARISON_CONTAINER}" \
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

# initialize_test_content accepts the default directory, a new directory, or an empty directory. A
# marker prevents an override from turning an unrelated nonempty directory into a deletion target.
initialize_test_content() {
    requested_path=${TEST_CONTENT_PATH%/}
    [ -n "${requested_path}" ] || requested_path=/
    mkdir -p "${requested_path}"
    resolved_path=$(CDPATH='' cd -- "${requested_path}" && pwd -P)
    resolved_home=$(CDPATH='' cd -- "${TEST_HOME_PATH}" && pwd -P)
    resolved_default="${resolved_home}/build/fluent-pipeline"

    case ${resolved_path} in
    / | "${resolved_home}")
        echo "Unsafe TEST_CONTENT_PATH '${resolved_path}': choose a dedicated output directory." >&2
        return 1
        ;;
    esac

    marker_path="${resolved_path}/${CONTENT_MARKER}"
    if { [ -e "${marker_path}" ] || [ -L "${marker_path}" ]; } &&
        { [ ! -f "${marker_path}" ] || [ -L "${marker_path}" ] ||
            [ "$(cat "${marker_path}" 2>/dev/null || true)" != "${CONTENT_MARKER_VALUE}" ]; }; then
        echo "TEST_CONTENT_PATH '${resolved_path}' has an invalid ownership marker." >&2
        echo "Choose a new or empty directory. No files were removed." >&2
        return 1
    fi
    if [ ! -e "${marker_path}" ] && [ "${resolved_path}" != "${resolved_default}" ] &&
        [ -n "$(find "${resolved_path}" -mindepth 1 -print -quit)" ]; then
        echo "TEST_CONTENT_PATH '${resolved_path}' is nonempty and is not owned by this test runner." >&2
        echo "Choose a new or empty directory. No files were removed." >&2
        return 1
    fi

    TEST_CONTENT_PATH=${resolved_path}
    printf '%s\n' "${CONTENT_MARKER_VALUE}" >"${marker_path}"
}

# reset_test_content empties a marked directory from a previous run. Logging agents can leave
# root-owned files, so the fallback mounts only that directory into a root helper container.
reset_test_content() {
    initialize_test_content
    find "${TEST_CONTENT_PATH}" -mindepth 1 -maxdepth 1 ! -name "${CONTENT_MARKER}" \
        -exec rm -rf -- {} + 2>/dev/null || true
    if [ -n "$(find "${TEST_CONTENT_PATH}" -mindepth 1 -maxdepth 1 ! -name "${CONTENT_MARKER}" -print -quit)" ]; then
        docker run --rm --security-opt label=disable --user 0 --name "${CLEANUP_CONTAINER}" \
            --entrypoint find -v "${TEST_CONTENT_PATH}":/content:rw \
            "${FLUENT_PIPELINE_TEST_IMAGE}" /content -mindepth 1 -maxdepth 1 \
            ! -name "${CONTENT_MARKER}" -exec rm -rf -- '{}' +
    fi
}

# save_agent_log prints the log of a container and leaves a copy beside the rendered configuration,
# so that a failed run carries the agent's own warnings in its artifact.
save_agent_log() {
    container_name=$1
    log_name=${2:-$1}
    docker logs "${container_name}" >"${TEST_CONTENT_PATH}/${log_name}.log" 2>&1 || true
    echo "=> Log of ${container_name}"
    cat "${TEST_CONTENT_PATH}/${log_name}.log"
}

# run_comparison runs one comparison, appends its report to the report of the scenario, and
# preserves its exit status; a pipe would report the status of the last command instead. A scenario
# compares more than once: the pipeline output, then the isolated parser contracts.
run_comparison() {
    report_file="${TEST_CONTENT_PATH}/report.txt"
    if "$@" >>"${report_file}" 2>&1; then
        status=0
    else
        status=1
    fi
    tail -n +"${report_lines:-1}" "${report_file}"
    report_lines=$(($(wc -l <"${report_file}") + 1))
    return "${status}"
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

# wait_for_system_inputs waits until the agent's tail inputs have opened the system and audit files, so the lines
# appended afterwards are read; the inputs skip whatever a file held before they opened it. The second argument is
# the log line the agent writes per opened file, as a printf format with one %s for the path.
FLUENTBIT_INPUT_OPENED='inotify_fs_add\(\).*name=%s$'
FLUENTD_INPUT_OPENED='following tail of %s$'
wait_for_system_inputs() {
    container_name=$1
    line_format=$2
    for host_log in /var/log/syslog /var/log/audit/audit.log /var/log/kubernetes/audit/audit.log; do
        # shellcheck disable=SC2059 # the format comes from the constants above, not from input
        wait_for_log "${container_name}" "$(printf "${line_format}" "${host_log}")"
    done
}

# check_metrics compares what the Fluent Bit Prometheus exporter serves with the checked-in metrics of the
# scenario. The metrics come from the log_to_metrics filters, which no output file carries: they are exported on
# their own port, so the scrape runs in the network namespace of the agent container. The scrape is retried until
# it carries every expected line, because the filters flush on their own interval.
check_metrics() {
    container_name=$1
    expected_file=$2
    actual_file="${TEST_CONTENT_PATH}/metrics.prom"
    if [ ! -s "${expected_file}" ]; then
        echo "The expected metrics file ${expected_file} is missing or empty" >&2
        return 1
    fi
    waited=0
    while [ "${waited}" -lt "${METRICS_TIMEOUT}" ]; do
        scrape_metrics "${container_name}" >"${actual_file}" 2>/dev/null || true
        if [ -s "${actual_file}" ] && ! grep -q -v -F -x -f "${actual_file}" "${expected_file}"; then
            break
        fi
        ensure_running "${container_name}" || break
        sleep 1
        waited=$((waited + 1))
    done
    if diff -u "${expected_file}" "${actual_file}"; then
        echo "=> The exporter served the expected metrics after ${waited} seconds"
        return 0
    fi
    echo "The exporter served different metrics after ${waited} seconds; the diff is above" >&2
    return 1
}

scrape_metrics() {
    docker run --rm --security-opt label=disable --network "container:$1" \
        --entrypoint wget "${FLUENT_PIPELINE_TEST_IMAGE}" -qO- http://127.0.0.1:2021/metrics
}

# count_expected_records counts the records the comparison expects in the output: each expected record carries one
# _test block, and a dropped probe describes a line that produces no record.
count_expected_records() {
    records=$(cat "$1"/*.log.json | grep -c '"_test"' || true)
    probes=$(cat "$1"/*.log.json | grep -c '"dropped": true' || true)
    echo $((${records:-0} - ${probes:-0}))
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
        "${logs_root}/var/log/messages"
}

###################################################################################################
# Run FluentD DaemonSet test logic
###################################################################################################
run_fluentd_test_logic() {
    FLD_DOCKER_NAME="${FLUENTD_CONTAINER}"
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

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
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

    echo "=> Waiting for FluentD to open the system and audit inputs"
    wait_for_system_inputs "${FLD_DOCKER_NAME}" "${FLUENTD_INPUT_OPENED}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"

    echo "=> Waiting until FluentD writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/fake-fluent.log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentd")" "${FLD_DOCKER_NAME}"

    echo "=> Stop and remove FluentD docker container"
    save_agent_log "${FLD_DOCKER_NAME}" fluentd
    docker stop "${FLD_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentD parsed logs and compare with expected data"
    run_comparison docker run --rm --security-opt label=disable --user "${HELPER_USER}" \
        --name "${COMPARISON_CONTAINER}" \
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
    FLB_DOCKER_NAME="${FLUENTBIT_CONTAINER}"
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

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
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
    wait_for_system_inputs "${FLB_DOCKER_NAME}" "${FLUENTBIT_INPUT_OPENED}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"

    echo "=> Waiting until FluentBit writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/output-log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit")" "${FLB_DOCKER_NAME}"

    echo "=> Waiting for the metrics exporter to serve the log_to_metrics counters"
    check_metrics "${FLB_DOCKER_NAME}" "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/metrics/fluentbit.prom"

    echo "=> Stop and remove FluentBit docker container"
    save_agent_log "${FLB_DOCKER_NAME}" fluent-bit
    docker stop "${FLB_DOCKER_NAME}"

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    run_comparison docker run --rm --security-opt label=disable --user "${HELPER_USER}" \
        --name "${COMPARISON_CONTAINER}" \
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
    FLB_FRW_DOCKER_NAME="${FLUENTBIT_FORWARDER_CONTAINER}"
    FLB_AGR_DOCKER_NAME="${FLUENTBIT_AGGREGATOR_CONTAINER}"
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

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
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

    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
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

    docker network create "${NETWORK_NAME}"

    docker run -d --security-opt label=disable --name "${FLB_AGR_DOCKER_NAME}" \
        --network="${NETWORK_NAME}" \
        --network-alias=fluent-bit-aggregator \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/aggregator-config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/output/":/fluentbit-output:rw \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for the FluentBit aggregator to listen for the forwarder"
    wait_for_log "${FLB_AGR_DOCKER_NAME}" 'input:forward.*listening on'

    docker run -d --security-opt label=disable --name "${FLB_FRW_DOCKER_NAME}" \
        --network="${NETWORK_NAME}" \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/forwarder-config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:rw \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting for the FluentBit forwarder to open the system and audit inputs"
    wait_for_system_inputs "${FLB_FRW_DOCKER_NAME}" "${FLUENTBIT_INPUT_OPENED}"

    echo "=> Start print prepared test data in logs"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/kubernetes/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/kubernetes/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/audit/audit.log" "${TEST_CONTENT_PATH}/logs/var/log/audit/audit.log"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/syslog" "${TEST_CONTENT_PATH}/logs/var/log/syslog"
    add_lines "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/input/system/messages" "${TEST_CONTENT_PATH}/logs/var/log/messages"

    echo "=> Waiting until the FluentBit aggregator writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/output-log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit-ha")" "${FLB_AGR_DOCKER_NAME}"

    echo "=> Waiting for the metrics exporter to serve the log_to_metrics counters"
    check_metrics "${FLB_AGR_DOCKER_NAME}" "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/metrics/fluentbit-ha.prom"

    echo "=> Print FlintBit forwarder logs"
    save_agent_log "${FLB_FRW_DOCKER_NAME}" fluent-bit-forwarder

    echo "=> Print FlintBit aggregator logs"
    save_agent_log "${FLB_AGR_DOCKER_NAME}" fluent-bit-aggregator

    echo "=> Stop and remove FluentBit docker container"
    docker stop "${FLB_FRW_DOCKER_NAME}"
    docker stop "${FLB_AGR_DOCKER_NAME}"

    docker network rm "${NETWORK_NAME}"

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    run_comparison docker run --rm --security-opt label=disable --user "${HELPER_USER}" \
        --name "${COMPARISON_CONTAINER}" \
        -v "${TEST_CONTENT_PATH}/output/":/output-logs/actual:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/fluentbit-ha/":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbitha \
        -stage test \
        -ignore "${INT_TESTS_IGNORE}"

    run_parser_contracts "${TEST_CONTENT_PATH}/forwarder-config" forwarder
}

###################################################################################################
# Run the Fluent Bit DaemonSet against a fake Kubernetes API server
###################################################################################################
KUBE_API_HOST="kubernetes.default.svc"

# start_fake_kube_api serves the pod metadata of the scenario over HTTPS under the name the rendered
# configuration hardcodes, and writes the certificate and token the agent mounts as its service
# account. The agent reaches it by an added host entry, so no name resolution is needed.
start_fake_kube_api() {
    metadata_dir=$1
    credentials_dir=$2
    mkdir -p "${credentials_dir}"
    docker run -d --security-opt label=disable --name "${KUBE_API_NAME}" --network "${NETWORK_NAME}" --user 0 \
        -v "${metadata_dir}":/pod-metadata:ro \
        -v "${credentials_dir}":/serviceaccount:rw \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -stage kube-api \
        -podMetadata /pod-metadata \
        -credentials /serviceaccount \
        -apiAddress :443
    wait_for_log "${KUBE_API_NAME}" 'Serving pod metadata'
}

run_kube_metadata_test_logic() {
    FLB_DOCKER_NAME="${FLUENTBIT_CONTAINER}"
    echo "=> Prepare test environment and test data"
    reset_test_content
    mkdir -p "${TEST_CONTENT_PATH}/config/" "${TEST_CONTENT_PATH}/logs/" "${TEST_CONTENT_PATH}/output/" \
        "${TEST_CONTENT_PATH}/serviceaccount/"

    echo "=> Prepare FluentBit configurations"
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
        -v "${TEST_HOME_PATH}/controllers/fluentbit/fluentbit.configmap/":/config-templates.d/:ro \
        -v "${TEST_CONTENT_PATH}/config/":/configuration.d/:z \
        -v "${TEST_CONTENT_PATH}/logs/":/testdata:z \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/kube-metadata.yaml":/assets/kube-metadata.yaml:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/kube-metadata/":/logs:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbit \
        -cr /assets/kube-metadata.yaml \
        -stage prepare \
        -loglevel warn \
        -ignore "${INT_TESTS_IGNORE}"

    speed_up_file_discovery "${TEST_CONTENT_PATH}/config"

    docker network create "${NETWORK_NAME}"

    echo "=> Run the fake Kubernetes API server"
    start_fake_kube_api "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/kube-metadata/pod-metadata" \
        "${TEST_CONTENT_PATH}/serviceaccount"
    api_address=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${KUBE_API_NAME}")

    echo "=> Run FluentBit to read, parse and output processed logs"
    docker run -d --security-opt label=disable --name "${FLB_DOCKER_NAME}" \
        --network "${NETWORK_NAME}" \
        --add-host "${KUBE_API_HOST}:${api_address}" \
        -e HOSTNAME=fake-fluent \
        -e NODE_NAME=fake-node \
        -v "${TEST_CONTENT_PATH}/config/":/fluent-bit/etc \
        -v "${TEST_CONTENT_PATH}/logs/var/log/":/var/log:z \
        -v "${TEST_CONTENT_PATH}/serviceaccount/":/var/run/secrets/kubernetes.io/serviceaccount:ro \
        -v "${TEST_CONTENT_PATH}/output/":/fluentbit-output:z \
        "${FLUENTBIT_IMAGE}"

    echo "=> Waiting until FluentBit writes the processed logs"
    wait_for_records "${TEST_CONTENT_PATH}/output/output-log" \
        "$(count_expected_records "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/kube-metadata")" \
        "${FLB_DOCKER_NAME}"

    echo "=> Print the pod requests the Kubernetes filter made"
    save_agent_log "${KUBE_API_NAME}" fake-kube-api

    echo "=> Stop and remove FluentBit docker container"
    save_agent_log "${FLB_DOCKER_NAME}" fluent-bit
    docker stop "${FLB_DOCKER_NAME}"
    docker stop "${KUBE_API_NAME}"
    docker network rm "${NETWORK_NAME}"

    echo "=> Run the docker container to analyze FluentBit parsed logs and compare with expected data"
    run_comparison docker run --rm --security-opt label=disable --user "${HELPER_USER}" \
        --name "${COMPARISON_CONTAINER}" \
        -v "${TEST_CONTENT_PATH}/output/":/output-logs/actual:ro \
        -v "${TEST_HOME_PATH}/test/fluent-pipeline/testdata/output/kube-metadata/":/output-logs/expected:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent fluentbit \
        -stage test \
        -ignore "${INT_TESTS_IGNORE}"
}

###################################################################################################
# Render every custom resource under testdata/assets/render and let the agent validate the result
###################################################################################################
RENDER_ASSETS="${TEST_HOME_PATH}/test/fluent-pipeline/testdata/assets/render"

# render_configuration renders the templates of one agent for one custom resource into a directory,
# the same way the prepare stage does for a scenario, without laying out any log file.
render_configuration() {
    agent=$1
    templates_dir=$2
    custom_resource=$3
    target_dir=$4
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${CONFIG_REPLACER_CONTAINER}" \
        -v "${templates_dir}":/config-templates.d:ro \
        -v "${target_dir}":/configuration.d:rw \
        -v "${custom_resource}":/assets/custom-resource.yaml:ro \
        "${FLUENT_PIPELINE_TEST_IMAGE}" \
        -agent "${agent}" \
        -cr /assets/custom-resource.yaml \
        -stage render \
        -loglevel warn
}

# validate_fluentbit_configuration asks Fluent Bit to load the rendered configuration without
# starting the engine; a plugin it cannot instantiate or a section it cannot parse fails the check.
validate_fluentbit_configuration() {
    docker run --rm --security-opt label=disable --name "${FLUENTBIT_RENDER_CONTAINER}" \
        -v "$1":/fluent-bit/etc:ro \
        "${FLUENTBIT_IMAGE}" --dry-run -c /fluent-bit/etc/fluent-bit.conf
}

# prepare_fluentd_mounts creates the files the operator's DaemonSet mounts into the Fluentd container and
# the dry run opens: the service account credentials and the TLS material of the Graylog, Loki, and
# HTTP outputs. A self-signed certificate stands in for each of them.
prepare_fluentd_mounts() {
    mounts_dir=$1
    mkdir -p "${mounts_dir}/serviceaccount" "${mounts_dir}/tls" "${mounts_dir}/loki-tls" "${mounts_dir}/http-tls"
    # The Fluentd image ships openssl; the helper image does not.
    docker run --rm --security-opt label=disable --user "${HELPER_USER}" --name "${FLUENTD_RENDER_CONTAINER}" \
        --entrypoint /bin/sh -v "${mounts_dir}":/mounts:rw "${FLUENTD_IMAGE}" -c '
            openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=render-test \
                -keyout /mounts/tls/tls.key -out /mounts/tls/tls.crt 2>/dev/null &&
            cp /mounts/tls/tls.crt /mounts/tls/ca.crt &&
            cp /mounts/tls/tls.key /mounts/tls/tls.crt /mounts/tls/ca.crt /mounts/loki-tls/ &&
            cp /mounts/tls/tls.key /mounts/tls/tls.crt /mounts/tls/ca.crt /mounts/http-tls/ &&
            echo placeholder-token >/mounts/serviceaccount/token &&
            cp /mounts/tls/ca.crt /mounts/serviceaccount/ca.crt'
}

# validate_fluentd_configuration runs Fluentd's dry run with the environment and the mounts the
# operator's DaemonSet gives the container; the rendered configuration reads the Graylog connection
# from the environment and opens the credential and certificate files.
validate_fluentd_configuration() {
    mounts_dir="${TEST_CONTENT_PATH}/render/fluentd-mounts"
    [ -f "${mounts_dir}/tls/tls.crt" ] || prepare_fluentd_mounts "${mounts_dir}"
    docker run --rm --security-opt label=disable --name "${FLUENTD_RENDER_CONTAINER}" \
        -e GRAYLOG_HOST=graylog.logging.svc -e GRAYLOG_PORT=12201 -e GRAYLOG_PROTOCOL=tcp \
        -e QUEUE_LIMIT_LENGTH=128 -e WATCH_KUBERNETES_METADATA=true -e MA_HOST=monitoring-agent.logging.svc \
        -v "$1":/fluentd/etc:ro \
        -v "${mounts_dir}/serviceaccount":/var/run/secrets/kubernetes.io/serviceaccount:ro \
        -v "${mounts_dir}/tls":/fluentd/tls:ro \
        -v "${mounts_dir}/loki-tls":/fluentd/output/loki/tls:ro \
        -v "${mounts_dir}/http-tls":/fluentd/output/http/tls:ro \
        "${FLUENTD_IMAGE}" fluentd --dry-run -c /fluentd/etc/fluent.conf
}

# check_rendered_configuration renders and validates one configuration and records the outcome. A
# custom resource that documents a known defect starts with "# expect-failure: <reason>": its
# validation has to fail, and a pass means the defect is gone and the line is due for removal.
check_rendered_configuration() {
    label=$1
    agent=$2
    templates_dir=$3
    custom_resource=$4
    validator=$5
    target_dir="${TEST_CONTENT_PATH}/render/${label}"
    log_file="${target_dir}.log"
    expected_failure=$(sed -n 's/^# expect-failure: //p' "${custom_resource}" | head -n 1)
    mkdir -p "${target_dir}"
    if render_configuration "${agent}" "${templates_dir}" "${custom_resource}" "${target_dir}" >"${log_file}" 2>&1 &&
        "${validator}" "${target_dir}" >>"${log_file}" 2>&1; then
        validated=true
    else
        validated=false
    fi
    if [ -z "${expected_failure}" ] && [ "${validated}" = true ]; then
        render_report="${render_report}${label}\tSucceeded\n"
    elif [ -n "${expected_failure}" ] && [ "${validated}" = false ]; then
        render_report="${render_report}${label}\tSucceeded\tfails as expected: ${expected_failure}\n"
    elif [ -n "${expected_failure}" ]; then
        render_report="${render_report}${label}\tFailed\tvalidation passed, remove the expect-failure line: ${expected_failure}\n"
        render_failures=$((render_failures + 1))
    else
        render_report="${render_report}${label}\tFailed\tsee ${log_file}\n"
        render_failures=$((render_failures + 1))
        echo "=> ${label}: the rendered configuration failed validation" >&2
        tail -n 20 "${log_file}" >&2
    fi
}

run_render_test_logic() {
    echo "=> Prepare test environment"
    reset_test_content
    mkdir -p "${TEST_CONTENT_PATH}/render"
    render_report=""
    render_failures=0

    echo "=> Render and validate the Fluent Bit configurations"
    for custom_resource in "${RENDER_ASSETS}"/fluentbit/*.yaml; do
        name=$(basename "${custom_resource}" .yaml)
        check_rendered_configuration "fluentbit/${name}" fluentbit \
            "${TEST_HOME_PATH}/controllers/fluentbit/fluentbit.configmap/" "${custom_resource}" \
            validate_fluentbit_configuration
    done

    echo "=> Render and validate the Fluent Bit forwarder and aggregator configurations"
    for custom_resource in "${RENDER_ASSETS}"/fluentbit-ha/*.yaml; do
        name=$(basename "${custom_resource}" .yaml)
        check_rendered_configuration "forwarder/${name}" fluentbitha \
            "${TEST_HOME_PATH}/controllers/fluentbit-forwarder-aggregator/forwarder.configmap/" "${custom_resource}" \
            validate_fluentbit_configuration
        check_rendered_configuration "aggregator/${name}" fluentbitha \
            "${TEST_HOME_PATH}/controllers/fluentbit-forwarder-aggregator/aggregator.configmap/" "${custom_resource}" \
            validate_fluentbit_configuration
    done

    echo "=> Render and validate the Fluentd configurations"
    for custom_resource in "${RENDER_ASSETS}"/fluentd/*.yaml; do
        name=$(basename "${custom_resource}" .yaml)
        check_rendered_configuration "fluentd/${name}" fluentd \
            "${TEST_HOME_PATH}/controllers/fluentd/fluentd.configmap/" "${custom_resource}" \
            validate_fluentd_configuration
    done

    {
        echo "--- Report of configuration rendering ---"
        printf 'CONFIGURATION\tSTATUS\tDETAILS\n'
        printf "%b" "${render_report}"
    } | tee "${TEST_CONTENT_PATH}/report.txt"
    [ "${render_failures}" -eq 0 ]
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

'kube-metadata')
    run_kube_metadata_test_logic
    ;;

'render')
    run_render_test_logic
    ;;

*)
    echo "Usage: $0 {fluentd|fluentbit|fluentbit-ha|kube-metadata|render}" >&2
    exit 2
    ;;

esac
