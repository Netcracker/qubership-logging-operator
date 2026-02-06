*** Variables ***
${GRAYLOG_PROTOCOL}             %{GRAYLOG_PROTOCOL}
${GRAYLOG_HOST}                 %{GRAYLOG_HOST}
${OPENSHIFT_DEPLOY}             %{OPENSHIFT_DEPLOY}
${GRAYLOG_PORT}                 %{GRAYLOG_PORT}
${GRAYLOG_USER}                 %{GRAYLOG_USER}
${GRAYLOG_PASS}                 %{GRAYLOG_PASS}
${VICTORIALOGS_URL}             %{VICTORIALOGS_URL=}
${VL_USER}                      %{VL_USER=}
${VL_PASSWORD}                  %{VL_PASSWORD=}
${VL_TOKEN}                     %{VL_TOKEN=}
${OPERATION_RETRY_COUNT}        %{OPERATION_RETRY_COUNT}
${RETRY_COUNT_FOR_FIRST_TEST}   250x
${OPERATION_RETRY_INTERVAL}     %{OPERATION_RETRY_INTERVAL}
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
Library  String
Library    Collections
Library    BuiltIn
Suite Setup  Run Keywords  Setup
...  AND  Create Config
...  AND  Create Log Generators
Suite Teardown  Run Keywords  Delete Log Generators
...  AND  Delete Config Map
Resource        keywords.robot

*** Keywords ***
Setup
    Check Graylog Install
    Run Keyword If  ${graylog_available}
    ...  Create Graylog Session
    Check Fluentbit And Fluentd
    IF  ${fluentd_exists} == True
        ${TAG_PREFIX}=  Set Variable  parsed.kubernetes.var.log.pods.${NAMESPACE}_
    ELSE
        ${TAG_PREFIX}=  Set Variable  var.log.pods.${NAMESPACE}_
    END
    Set Suite Variable  ${TAG_PREFIX}

    ${victorialogs_enabled}=  Run Keyword And Return Status
    ...  Should Not Be Equal  ${EMPTY}  ${VICTORIALOGS_URL}
    Set Suite Variable  ${victorialogs_enabled}
    IF    not ${graylog_available} and ${victorialogs_enabled}
        Run Keyword  Create VictoriaLogs Session
    END

Create Graylog Session
    ${auth}=  Create List  ${GRAYLOG_USER}  ${GRAYLOG_PASS}
    ${headers}  Create Dictionary  Content-Type=application/json  Accept=application/json
    Set Global Variable  ${headers}
    Create Session  graylog  ${GRAYLOG_PROTOCOL}://${GRAYLOG_HOST}:${GRAYLOG_PORT}  auth=${auth}  disable_warnings=1  verify=False  timeout=10

Create VictoriaLogs Session
    &{headers}=  Create Dictionary    Content-Type=application/x-www-form-urlencoded
    ${has_token}=  Run Keyword And Return Status  Should Not Be Empty  ${VL_TOKEN}
    Run Keyword If  ${has_token}
    ...  Set To Dictionary  ${headers}  Authorization=Bearer ${VL_TOKEN}
    ${has_user}=  Run Keyword And Return Status  Should Not Be Empty  ${VL_USER}
    ${has_pass}=  Run Keyword And Return Status  Should Not Be Empty  ${VL_PASSWORD}
    ${use_basic}=  Evaluate  ${has_user} and ${has_pass}
    IF    ${use_basic}
        ${auth}=  Create List  ${VL_USER}  ${VL_PASSWORD}
        Create Session  vl_session  ${VICTORIALOGS_URL}
        ...  headers=${headers}  verify=False  timeout=10  auth=${auth}
    ELSE
        Create Session  vl_session  ${VICTORIALOGS_URL}
        ...  headers=${headers}  verify=False  timeout=10
    END
    Set Suite Variable  ${use_basic}

Get Log Generator Names
    ${gen1_names}=  Get Pod Names For Deployment Entity  qubership-log-generator  ${NAMESPACE}
    ${gen1_pod_name}=  Set Variable  ${gen1_names}[0]
    Set Suite Variable  ${generator_pod_name}  ${gen1_pod_name}
    ${gen2_names}=  Get Pod Names For Deployment Entity  kube-log-generator  ${NAMESPACE}
    ${gen2_pod_name}=  Set Variable    ${gen2_names}[0]
    Set Suite Variable    ${kube_generator_pod_name}    ${gen2_pod_name}
    ${gen3_names}=  Get Pod Names For Deployment Entity  json-log-generator  ${NAMESPACE}
    ${gen3_pod_name}=  Set Variable    ${gen3_names}[0]
    Set Suite Variable    ${json_generator_pod_name}    ${gen3_pod_name}

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
    IF    ${graylog_available}
        ${resp}=  GET On Session  graylog  url=/api/search/universal/relative?query=pod:${query}&range=3600&limit=50&sort=timestamp:desc&pretty=true  headers=${headers}
        ${messages}=  Get From Dictionary  ${resp.json()}  messages
    ELSE IF    ${victorialogs_enabled}
        ${messages}=  Search Messages In Victorialogs By Query  ${query}
    ELSE
        Fail    No storage backend available: Graylog=${graylog_available}, VictoriaLogs=${victorialogs_enabled}
    END
    Set Suite Variable  ${messages}
    Should Not Be Empty  ${messages}

Search Messages In Victorialogs By Query
    [Arguments]  ${query}
    ${logsql}=    Set Variable    pod:${query} | sort by (_time) desc | limit 50
    ${body}=    Create Dictionary    query=${logsql}
    ${resp}=  POST On Session  vl_session  /select/logsql/query
    ...  data=${body}
    ${response_text}=    Set Variable    ${resp.text}
    @{lines}=    Split To Lines    ${response_text}
    @{messages}=  Create List
    FOR  ${line}  IN  @{lines}
        ${line}=    Strip String    ${line}
        Continue For Loop If    '${line}' == ''
        ${parse_result}=    Run Keyword And Ignore Error
        ...    Evaluate    __import__('json').loads($line)    modules=json
        IF    '${parse_result[0]}' == 'FAIL'
            Continue For Loop
        END
        ${obj}=    Set Variable    ${parse_result[1]}
        ${is_dict}=    Run Keyword And Return Status
        ...    Evaluate    isinstance($obj, dict)
        Continue For Loop If    not ${is_dict}
        ${has_level}=    Run Keyword And Return Status
        ...    Dictionary Should Contain Key    ${obj}    level
        ${has_msg}=    Run Keyword And Return Status
        ...    Dictionary Should Contain Key    ${obj}    _msg
        ${has_pod}=    Run Keyword And Return Status
        ...    Dictionary Should Contain Key    ${obj}    pod

        Continue For Loop If    not ${has_level} or not ${has_msg} or not ${has_pod}

        ${vl_level}=    Get From Dictionary    ${obj}    level
        ${vl_msg}=    Get From Dictionary    ${obj}    _msg
        ${vl_pod}=    Get From Dictionary    ${obj}    pod
        ${level}=    Set Variable    ${INFO_LEVEL}
        IF    '${vl_level}' == 'err' or '${vl_level}' == 'error' or '${vl_level}' == 'emerg'
            ${level}=    Set Variable    ${ERROR_LEVEL}
        ELSE IF    '${vl_level}' == 'warn' or '${vl_level}' == 'warning'
            ${level}=    Set Variable    ${WARN_LEVEL}
        ELSE IF    '${vl_level}' == 'info'
            ${level}=    Set Variable    ${INFO_LEVEL}
        ELSE IF    '${vl_level}' == 'debug'
            ${level}=    Set Variable    ${DEBUG_LEVEL}
        ELSE IF    '${vl_level}' == 'alert' or '${vl_level}' == 'fatal' or '${vl_level}' == 'crit'
            ${level}=    Set Variable    ${FATAL_LEVEL}
        END

        ${namespace}=    Run Keyword And Ignore Error
        ...    Get From Dictionary    ${obj}    namespace
        ${namespace}=    Set Variable If
        ...    '${namespace[0]}' == 'PASS'    ${namespace[1]}    ${EMPTY}

        ${parsed}=    Run Keyword And Ignore Error
        ...    Get From Dictionary    ${obj}    parsed
        ${parsed}=    Set Variable If
        ...    '${parsed[0]}' == 'PASS'    ${parsed[1]}    ${EMPTY}

        ${time}=    Run Keyword And Ignore Error
        ...    Get From Dictionary    ${obj}    _time
        ${time}=    Set Variable If
        ...    '${time[0]}' == 'PASS'    ${time[1]}    ${EMPTY}

        &{msg}=    Create Dictionary
        ...    level=${level}
        ...    message=${vl_msg}
        ...    pod=${vl_pod}
        ...    namespace=${namespace}
        ...    parsed=${parsed}
        ...    time=${time}

        &{wrapper}=    Create Dictionary    message=${msg}
        Append To List    ${messages}    ${wrapper}
    END

    ${messages_count}=    Get Length    ${messages}
    Run Keyword If    ${messages_count} == 0
    ...    Fail    No valid messages parsed from VictoriaLogs

    RETURN    @{messages}

Check Message Parsing
    [Arguments]  ${log_type}  ${expected_level}  ${pod_name}  ${multiline}=${False}
    ${child_suffix}=  Set Variable If  ${multiline}  _child  ${EMPTY}
    IF    ${graylog_available}
        ${query}=  Set Variable  "${pod_name}"+AND+message%3A+"${log_type}${child_suffix}"+NOT+message%3A+"templates"
    ELSE IF    ${victorialogs_enabled}
        ${query}=  Set Variable  ${pod_name} AND _msg:${log_type}${child_suffix} NOT _msg:templates
    END

    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages by query  ${query}
    ${messages_count}=    Get Length    ${messages}
    ${message_index}=    Set Variable If
    ...    ${messages_count} >= 2    1
    ...    ${messages_count} == 1    0
    ...    0
    ${message}=  Get From Dictionary  ${messages}[${message_index}]  message
    ${level}=   Get From Dictionary  ${message}  level
    Should Be Equal As Strings  ${level}  ${expected_level}  Severity level is not matching expected
    ${message_field}=  Get From Dictionary  ${message}  message
    Set Suite Variable  ${message_field}
    Should Contain  ${message_field}  ${log_type}
    ${pod}=  Get From Dictionary  ${message}  pod
    ${log_namespace}=  Get From Dictionary  ${message}  namespace
    Should Be Equal As Strings  ${log_namespace}  ${NAMESPACE}
    Should Contain    ${pod}    ${pod_name}
    IF  ${fluentd_exists} != True
        ${parsed}=  Get From Dictionary  ${message}  parsed
        Should Be Equal As Strings  ${parsed}  true  Log message is not parsed
    END
    IF    ${graylog_available}
        IF    ${fluentd_exists} == True
            ${time}=  Get From Dictionary  ${message}  time
        ELSE
            ${time}=  Get From Dictionary  ${message}  timestamp
        END
    ELSE IF    ${victorialogs_enabled}
        ${time}=    Get From Dictionary  ${message}  time
    END
    Should Match Regexp  ${time}  ${DATE_TIME_REGEXP}  Time format is not matching expected

*** Test Cases ***
Test Create Log Generator And Check Messages Exist
    [Tags]  log-generator
    Wait Until Keyword Succeeds  ${RETRY_COUNT_FOR_FIRST_TEST}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages by query  ${generator_pod_name}

Check Parsing Go Info Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_info_log
    Check Message Parsing  ${log_type}  ${INFO_LEVEL}  ${generator_pod_name}

Check Parsing Go Warning Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_warn_log
    Check Message Parsing  ${log_type}  ${WARN_LEVEL}  ${generator_pod_name}

Check Parsing Go Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_error_log
    Check Message Parsing  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}

Check Parsing Go Debug Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_debug_log
    Check Message Parsing  ${log_type}  ${DEBUG_LEVEL}  ${generator_pod_name}

Check Parsing Go Fatal Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_fatal_log
    Check Message Parsing  ${log_type}  ${FATAL_LEVEL}  ${generator_pod_name}

Check Parsing Go Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  go_multiline_log
    Check Message Parsing  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}  ${True}
    Should Contain  ${message_field}  ${log_type}_child

Check Parsing Java Info Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_info_log
    Check Message Parsing  ${log_type}  ${INFO_LEVEL}  ${generator_pod_name}

Check Parsing Java Warning Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_warn_log
    Check Message Parsing  ${log_type}  ${WARN_LEVEL}  ${generator_pod_name}

Check Parsing Java Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_error_log
    Check Message Parsing  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}

Check Parsing Java Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  java_multiline_log
    Check Message Parsing  ${log_type}  ${ERROR_LEVEL}  ${generator_pod_name}  ${True}
    Should Contain  ${message_field}  ${log_type}_child

Check Parsing Json Info Logs
    [Tags]  log-generator
    Log To Console  ${\n}Config for json log does not match format from documentation. Level is not parsed. Default level = 6
    ${log_type}=  Set Variable  json_error_log
    Check Message Parsing  ${log_type}  6  ${json_generator_pod_name}

Check Parsing Klog Error Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  klog_error_log
    Check Message Parsing  ${log_type}  ${ERROR_LEVEL}  ${kube_generator_pod_name}

Check Parsing Klog Multiline Logs
    [Tags]  log-generator
    ${log_type}=  Set Variable  klog_multiline_log
    Check Message Parsing  ${log_type}  ${INFO_LEVEL}  ${kube_generator_pod_name}  ${True}
    Should Contain  ${message_field}  ${log_type}_child
