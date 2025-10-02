*** Variables ***
${GRAYLOG_PROTOCOL}             %{GRAYLOG_PROTOCOL}
${GRAYLOG_HOST}                 %{GRAYLOG_HOST}
${OPENSHIFT_DEPLOY}             %{OPENSHIFT_DEPLOY}
${GRAYLOG_PORT}                 %{GRAYLOG_PORT}
${GRAYLOG_USER}                 %{GRAYLOG_USER}
${GRAYLOG_PASS}                 %{GRAYLOG_PASS}
${OPERATION_RETRY_COUNT}        60x
${RETRY_COUNT_FOR_FIRST_TEST}   250x
${OPERATION_RETRY_INTERVAL}     5s
${FILES_PATH}                   ./source_files/log_generator
${DEPLOYMENT_FILE}              ${FILES_PATH}/deployment.yaml
${KLOG_DEPLOYMENT_FILE}         ${FILES_PATH}/klog_deployment.yaml
${JSON_DEPLOYMENT_FILE}         ${FILES_PATH}/json_deployment.yaml
${CONFIG_FILE}                  ${FILES_PATH}/config.yaml
${DATE_TIME_REGEXP}             [0-9]{4}\-[0-9]{2}\-[0-9]{2}T[0-9]{2}\:[0-9]{2}:[0-9]{2}
${TAG_PREFIX}                   parsed.kubernetes.var.log.pods.${NAMESPACE}_
${FATAL_LEVEL}                  1
${ERROR_LEVEL}                  3
${WARN_LEVEL}                   4
${INFO_LEVEL}                   6
${DEBUG_LEVEL}                  7

*** Settings ***
Library  OperatingSystem
Suite Setup  Run Keywords  Setup
...  AND  Create Config
...  AND  Create Log Generators
Suite Teardown  Run Keywords  Delete Log Generators
...  AND  Delete Config Map
Resource        keywords.robot

*** Keywords ***
Setup
    ${headers}  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Global Variable  ${headers}
    ${auth}=  Create List  ${GRAYLOG_USER}  ${GRAYLOG_PASS}
    Create Session  graylog  ${GRAYLOG_PROTOCOL}://${GRAYLOG_HOST}:${GRAYLOG_PORT}  auth=${auth}  disable_warnings=1  verify=False  timeout=10
    Check Fluentbit And Fluentd
    IF  ${fluentd_exists} == True
        ${TAG_PREFIX}=  Set Variable  parsed.kubernetes.var.log.pods.${NAMESPACE}_
    ELSE
        ${TAG_PREFIX}=  Set Variable  var.log.pods.${NAMESPACE}_
    END
    Set Suite Variable  ${TAG_PREFIX}

Get Log Generator Names
    ${gen1_names}=  Get Pod Names For Deployment Entity  qubership-log-generator  ${NAMESPACE}
    ${gen1_pod_name}=  Set Variable  ${gen1_names}[0]
    Set Suite Variable  ${generator_pod_name}  ${gen1_pod_name}
    Log To Console    qubership generator pod name: ${gen1_pod_name}

    ${gen2_names}=  Get Pod Names For Deployment Entity  kube-log-generator  ${NAMESPACE}
    ${gen2_pod_name}=  Set Variable    ${gen2_names}[0]
    Set Suite Variable    ${kube_generator_pod_name}    ${gen2_pod_name}
    Log To Console    klog generator pod name: ${gen2_pod_name}

    ${gen3_names}=  Get Pod Names For Deployment Entity  json-log-generator  ${NAMESPACE}
    ${gen3_pod_name}=  Set Variable    ${gen3_names}[0]
    Set Suite Variable    ${json_generator_pod_name}    ${gen3_pod_name}
    Log To Console    json generator pod name: ${gen3_pod_name}

Create Log Generators
    [Documentation]    Creating 3 pods: qubership-log-generator, json-log-generator and kube-log-generator
    IF  "${OPENSHIFT_DEPLOY}" == "true"
        Create Deployment Entity From File  ${DEPLOYMENT_FILE}  ${NAMESPACE}
        Create Deployment Entity From File    ${KLOG_DEPLOYMENT_FILE}    ${NAMESPACE}
        Create Deployment Entity From File    ${JSON_DEPLOYMENT_FILE}    ${NAMESPACE}
    ELSE
        ${new_deployment}=  Add Security Context To Deployment  ${DEPLOYMENT_FILE}  ${NAMESPACE}
        Create Deployment Entity  ${new_deployment}  ${NAMESPACE}
        ${new_kube_deployment}=  Add Security Context To Deployment    ${KLOG_DEPLOYMENT_FILE}    ${NAMESPACE}
        Create Deployment Entity    ${new_kube_deployment}    ${NAMESPACE}
        ${new_json_deployment}=  Add Security Context To Deployment    ${JSON_DEPLOYMENT_FILE}    ${NAMESPACE}
        Create Deployment Entity    ${new_json_deployment}    ${NAMESPACE}
    END
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Get Log Generator Names

Create Config
    Create Config Map From File  ${NAMESPACE}  ${CONFIG_FILE}

Delete Log Generators
    Delete Deployment Entity    qubership-log-generator    ${NAMESPACE}
    Delete Deployment Entity    kube-log-generator         ${NAMESPACE}
    Delete Deployment Entity    json-log-generator         ${NAMESPACE}

Delete Config Map
    Delete Config Map By Name  qubership-log-generator-config  ${NAMESPACE}

Search messages by query
    [Arguments]  ${query}
    ${resp}=  GET On Session  graylog  url=/api/search/universal/relative?query=pod:${query}&range=3600&limit=50&sort=timestamp:desc&pretty=true  headers=${headers}
    ${messages}=  Get From Dictionary  ${resp.json()}  messages
    Set Suite Variable  ${messages}
    Should Not Be Empty  ${messages}

Check Message Parsing
    [Arguments]  ${query}  ${log_type}  ${expected_level}  ${pod_name}
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages by query  ${query}
    ${message}=  Get From Dictionary  ${messages}[1]  message
    ${level}=   Get From Dictionary  ${message}  level
    Should Be Equal As Strings  ${level}  ${expected_level}
    ${message_field}=  Get From Dictionary  ${message}  message
    Set Suite Variable  ${message_field}
    Should Contain  ${message_field}  ${log_type}
    ${pod}=  Get From Dictionary  ${message}  pod
    ${log_namespace}=  Get From Dictionary  ${message}  namespace
    Should Be Equal As Strings  ${log_namespace}  ${NAMESPACE}
    Should Contain    ${pod}    ${pod_name}
    IF  ${fluentd_exists} == True
        ${time}=  Get From Dictionary  ${message}  time
    ELSE
        ${time}=  Get From Dictionary  ${message}  timestamp
        ${parsed}=  Get From Dictionary  ${message}  parsed
        Should Be Equal As Strings  ${parsed}  true
    END
    Should Match Regexp  ${time}  ${DATE_TIME_REGEXP}

*** Test Cases ***
Test Create Log Generator And Check Messages Exist
    [Tags]  log-generator
    Wait Until Keyword Succeeds  ${RETRY_COUNT_FOR_FIRST_TEST}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages by query  "${generator_pod_name}"

Check Parsing Go Info Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_info_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${INFO_LEVEL}  ${generator_pod_name}

Check Parsing Go Warning Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_warn_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${WARN_LEVEL}  ${generator_pod_name}

Check Parsing Go Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_error_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}

Check Parsing Go Debug Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_debug_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${DEBUG_LEVEL}  ${generator_pod_name}

Check Parsing Go Fatal Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_fatal_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${FATAL_LEVEL}  ${generator_pod_name}

Check Parsing Go Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_multiline_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}_parent"+AND+message%3A+"${log_type}_child"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}
    Should Contain  ${message_field}  ${log_type}_child

Check Parsing Java Info Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_info_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${INFO_LEVEL}  ${generator_pod_name}

Check Parsing Java Warning Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_warn_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${WARN_LEVEL}  ${generator_pod_name}

Check Parsing Java Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_error_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}

Check Parsing Java Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_multiline_log
    ${query}=  Set Variable  "${generator_pod_name}"+AND+message%3A+"${log_type}_parent"+AND+message%3A+"${log_type}_child"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}
    Should Contain  ${message_field}  ${log_type}_child

Check Parsing Json Info Logs
    [Tags]  log-generator
    Log To Console  ${\n}Config for json log does not match format from documentation. Level is not parsed. Default level = 6
    ${log_type}=  Set Variable  json_error_log
    ${query}=  Set Variable  "${json_generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  6  ${json_generator_pod_name}

Check Parsing Klog Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  klog_error_log
    ${query}=  Set Variable  "${kube_generator_pod_name}"+AND+message%3A+"${log_type}"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${ERROR_LEVEL}  ${kube_generator_pod_name}

Check Parsing Klog Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  klog_multiline_log
    ${query}=  Set Variable  "${kube_generator_pod_name}"+AND+message%3A+"${log_type}_parent"+AND+message%3A+"${log_type}_child"+NOT+message%3A+"templates"
    Check Message Parsing  ${query}  ${log_type}  ${INFO_LEVEL}  ${kube_generator_pod_name}
