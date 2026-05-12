*** Settings ***
Library  OperatingSystem
Library  BuiltIn
Library  String

*** Variables ***
${GRAYLOG_USER}      ${EMPTY}
${GRAYLOG_PASS}      ${EMPTY}
${VM_USER}           ${EMPTY}
${SSH_KEY}           ${EMPTY}
${VL_TOKEN}          ${EMPTY}
${VL_USER}           ${EMPTY}
${VL_PASSWORD}       ${EMPTY}

*** Keywords ***
Load Integration Test Secrets
    ${dir}=  Get Environment Variable  INTEGRATION_TESTS_SECRETS_DIR  ${EMPTY}
    Should Not Be Equal  ${dir}  ${EMPTY}
    ...  INTEGRATION_TESTS_SECRETS_DIR must be set to the mounted secret directory (sensitive values are not read from process environment)
    ${gu}=  Get Required Secret From Dir  ${dir}  graylog-user
    ${gp}=  Get Required Secret From Dir  ${dir}  graylog-password
    ${vu}=  Get Required Secret From Dir  ${dir}  vm-user
    ${sk}=  Get Required Secret From Dir  ${dir}  ssh-key
    ${vlt}=  Get Optional Secret From Path File Env  VL_TOKEN_FILE
    ${vlu}=  Get Optional Secret From Path File Env  VL_USER_FILE
    ${vlp}=  Get Optional Secret From Path File Env  VL_PASSWORD_FILE
    Set Suite Variable  ${GRAYLOG_USER}  ${gu}
    Set Suite Variable  ${GRAYLOG_PASS}  ${gp}
    Set Suite Variable  ${VM_USER}  ${vu}
    Set Suite Variable  ${SSH_KEY}  ${sk}
    Set Suite Variable  ${VL_TOKEN}  ${vlt}
    Set Suite Variable  ${VL_USER}  ${vlu}
    Set Suite Variable  ${VL_PASSWORD}  ${vlp}

Get Required Secret From Dir
    [Arguments]  ${secrets_dir}  ${filename}
    ${fullpath}=  Catenate  SEPARATOR=/  ${secrets_dir}  ${filename}
    OperatingSystem.File Should Exist  ${fullpath}
    ...  msg=Missing secret file ${fullpath}. Sensitive values must be supplied as mounted files under ${secrets_dir}.
    ${content}=  OperatingSystem.Get File  ${fullpath}
    ${stripped}=  Strip String  ${content}
    Return From Keyword  ${stripped}

Get Optional Secret From Path File Env
    [Arguments]  ${path_env_name}
    ${filepath}=  Get Environment Variable  ${path_env_name}  ${EMPTY}
    IF  '${filepath}' == '${EMPTY}'
        Return From Keyword  ${EMPTY}
    END
    OperatingSystem.File Should Exist  ${filepath}
    ...  msg=${path_env_name} points to ${filepath} but file is missing
    ${content}=  OperatingSystem.Get File  ${filepath}
    ${stripped}=  Strip String  ${content}
    Return From Keyword  ${stripped}
