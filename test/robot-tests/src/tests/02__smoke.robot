*** Variables ***
${GRAYLOG_PROTOCOL}             %{GRAYLOG_PROTOCOL}
${GRAYLOG_HOST}                 %{GRAYLOG_HOST}
${GRAYLOG_PORT}                 %{GRAYLOG_PORT}
${GRAYLOG_USER}                 %{GRAYLOG_USER}
${GRAYLOG_PASS}                 %{GRAYLOG_PASS}
${OPERATION_RETRY_COUNT}        %{OPERATION_RETRY_COUNT}
${OPERATION_RETRY_INTERVAL}     %{OPERATION_RETRY_INTERVAL}
${VICTORIALOGS_URL}             %{VICTORIALOGS_URL=}
${VL_USER}                      %{VL_USER=}
${VL_PASSWORD}                  %{VL_PASSWORD=}
${VL_TOKEN}                     %{VL_TOKEN=}


*** Settings ***
Library  String
Suite Setup    Run Keywords  Setup
...  AND  Check Fluentbit And Fluentd
Resource        keywords.robot

*** Keywords ***
Setup
    Check Graylog Install
    Log To Console  graylog_available=${graylog_available}
    Run Keyword If  ${graylog_available}
    ...  Create Graylog Session

    ${victorialogs_enabled}=  Run Keyword And Return Status
    ...  Should Not Be Equal  ${EMPTY}  ${VICTORIALOGS_URL}
    Set Suite Variable  ${victorialogs_enabled}
    Log To Console  victorialogs_enabled=${victorialogs_enabled}
    Run Keyword If  ${victorialogs_enabled}
    ...  Create VictoriaLogs Session

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

Check HTTP Connection
    ${logsql}=    Set Variable    * | limit 1
    ${body}=    Create Dictionary    query=${logsql}
    ${resp}=  POST On Session  vl_session  url=/select/logsql/query
    ...  data=${body}
    Should Be Equal As Integers  ${resp.status_code}  200

Check Graylog Availability
    ${resp}=  Get On Session  graylog  url=/
    Should Be Equal As Strings  ${resp.status_code}  200

Check Streams Logs
    [Arguments]  ${stream_element}
    ${resp}=  Get On Session  graylog  url=/api/streams
    ${streams}=  Get From Dictionary  ${resp.json()}  streams
    Should Contain  str(${streams})  'title': '${stream_element}'

Get Graylog Version
    ${resp}=  Get On Session  graylog  url=/api/cluster
    @{keys}=  Get Dictionary Keys  ${resp.json()}
    ${key}=  Get From List  ${keys}  0
    ${dict}=  Get From Dictionary  ${resp.json()}  ${key}
    ${version}=  Get From Dictionary  ${dict}  version
    RETURN  ${version}

Search messages
    IF  ${graylog_available}
        ${resp}=  GET On Session  graylog  url=/api/search/universal/relative?query=*&range=3600&limit=50&sort=timestamp:desc&pretty=true  headers=${headers}
        ${messages}=  Get From Dictionary  ${resp.json()}  messages
    ELSE IF  ${victorialogs_enabled}
        ${logsql}=  Set Variable  * | limit 10
        &{data}=    Create Dictionary    query=${logsql}
        ${resp}=  POST On Session  vl_session  url=/select/logsql/query   data=${data}
        Should Be Equal As Integers  ${resp.text}  200
        ${messages}= ${resp.text}
    END
    Should Not Be Empty  ${messages}
    Set Suite Variable  ${messages}

Check Indexer Cluster Status
    ${resp}=  GET On Session  graylog  url=/api/system/indexer/cluster/health
    ${status}=  Get From Dictionary  ${resp.json()}  status
    Should Be Equal As Strings  ${status}  green

Check Pods Are Ready
    [Arguments]  ${object}  ${ready_name}  ${expected_name}
    ${status}=  Set Variable  ${object.status}
    ${ready}=  Set Variable  ${status.${ready_name}}
    ${expected}=  Set Variable  ${status.${expected_name}}
    Should Be Equal  ${ready}  ${expected}

Get Source List From Messages
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages
    @{source_list}=    Create List
    FOR    ${el}    IN    @{messages}
        ${message}=   Get From Dictionary  ${el}  message
        ${source}=   Get From Dictionary  ${message}  source
        Append To List  ${source_list}  ${source}
    END
    RETURN  @{source_list}

Get Pod Names For Service
    [Arguments]  ${service_name}
    &{dict}=  Create Dictionary  component=${service_name}
    ${pods}=  Get Pod Names By Selector  ${NAMESPACE}  ${dict}
    RETURN  ${pods}

Search Messages In Graylog By Query
    [Arguments]  ${query}
    ${resp}=  GET On Session  graylog  url=/api/search/universal/relative?query=${query}&range=3600&limit=50&sort=timestamp:desc&pretty=true  headers=${headers}
    ${messages}=  Get From Dictionary  ${resp.json()}  messages
    Set Suite Variable  ${messages}
    Should Not Be Empty  ${messages}

Search Messages In Victorialogs By Query
    [Arguments]  ${query}
    &{data}=    Create Dictionary    query=${query}
    ${resp}=  POST On Session  vl_session  url=/select/logsql/query
    ...  data=${data}

    Should Be Equal As Integers  ${resp.status_code}  200
    Should Not Be Empty  ${resp.text}  No response from VictoriaLogs for query: ${query}
    RETURN  ${resp.text}

Check Message From Any Pod
    [Arguments]  ${service_name}
    @{pod_list}=  Get Pod Names For Service  ${service_name}
    Should Not Be Empty  @{pod_list}  No ${service_name} pods found
    FOR  ${pod}  IN  @{pod_list}
        IF  ${graylog_available}
            Run Keyword    Search Messages In Graylog By Query  source%3A+"${pod}"
        ELSE IF  ${victorialogs_enabled}
            Run Keyword    Search Messages In Victorialogs By Query  hostname:${pod} | sort by (_time) desc | limit 10
        END
        Exit For Loop
    END

Check Messages From All Pods
    [Arguments]  ${service_name}
    @{pod_list}=  Get Pod Names For Service  ${service_name}
    FOR  ${pod}  IN  @{pod_list}
        IF  ${graylog_available}
            Run Keyword    Search Messages In Graylog By Query  source%3A+"${pod}"
        ELSE IF  ${victorialogs_enabled}
            Run Keyword    Search Messages In Victorialogs By Query  hostname:${pod} | sort by (_time) desc | limit 10
        END
    END

*** Test Cases ***
Test Graylog Availability Check
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Graylog Availability

Check System Logs Stream Exists
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Streams Logs  System logs

Check Audit Logs Stream Exists
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Streams Logs  Audit logs

Check All Messages Stream Exists
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    ${version} =  Get Graylog Version
    @{numbers} =  Split String  ${version}  .
    ${first_number}  Convert To Integer  ${numbers[0]}
    IF  ${first_number} >= 5
        ${stream_logs_name}=  Set Variable  Default Stream
    ELSE
        ${stream_logs_name}=  Set Variable  All messages
    END
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Streams Logs  ${stream_logs_name}

Test Indexer Cluster Status
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Indexer Cluster Status

Test VictoriaLogs Connection
    [Tags]  smoke
    Skip If  ${graylog_available}  Graylog is installed
    Skip If  ${victorialogs_enabled} == ${False}  VictoriaLogs is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check HTTP Connection

Test Search Message
    [Tags]  smoke  graylog
    Skip If  ${graylog_available} == False  Graylog is not available
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Search messages

Check Message From Any Fluentbit Pod
    [Tags]  smoke
    Skip If  ${fluentbit_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Message From Any Pod  logging-fluentbit

Check Message From Any Fluentd Pod
    [Tags]  smoke
    Skip If  ${fluentd_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Message From Any Pod  logging-fluentd

Check Message From Any Fluentbit Forwarder Pod
    [Tags]  smoke
    Skip If  ${fluentbit_forw_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Message From Any Pod  logging-fluentbit-forwarder

Check Messages From All Fluentbit Pods
    [Tags]  smoke
    Skip If  ${fluentbit_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Messages From All Pods  logging-fluentbit

Check Messages From All Fluentd Pods
    [Tags]  smoke
    Skip If  ${fluentd_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Messages From All Pods  logging-fluentd

Check Messages From All Fluentbit Forwarder Pods
    [Tags]  smoke
    Skip If  ${fluentbit_forw_exists} != True
    Wait Until Keyword Succeeds  ${OPERATION_RETRY_COUNT}  ${OPERATION_RETRY_INTERVAL}
    ...  Check Messages From All Pods  logging-fluentbit-forwarder

Test Check Fluentbit Status
    [Tags]  smoke
    Skip If  ${fluentbit_exists} != True
    ${daemon}=  Get Daemon Set  logging-fluentbit  ${NAMESPACE}
    Check Pods Are Ready  ${daemon}  number_ready  current_number_scheduled

Test Check Fluentd Status
    [Tags]  smoke
    Skip If  ${fluentd_exists} != True
    ${daemon}=  Get Daemon Set  logging-fluentd  ${NAMESPACE}
    Check Pods Are Ready  ${daemon}  number_ready  current_number_scheduled

Test Check Fluentbit Forwarder Status
    [Tags]  smoke
    Skip If  ${fluentbit_forw_exists} != True
    ${stateful}=  Get Daemon Set  logging-fluentbit-forwarder  ${NAMESPACE}
    Check Pods Are Ready  ${stateful}  number_ready  current_number_scheduled

Test Check Events Reader Status
    [Tags]  smoke
    ${deployment}=  Get Deployment Entity  events-reader  ${NAMESPACE}
    Check Pods Are Ready  ${deployment}  ready_replicas  replicas

