import os


def _read_file_strip(path: str) -> str:
    try:
        with open(path, encoding="utf-8") as handle:
            return handle.read().strip()
    except OSError:
        return ""


def _read_secret_file(environ: dict, filename: str) -> str:
    secrets_dir = environ.get("INTEGRATION_TESTS_SECRETS_DIR", "")
    if not secrets_dir:
        return ""
    path = os.path.join(secrets_dir, filename)
    if not os.path.isfile(path):
        return ""
    return _read_file_strip(path)


def get_excluded_tags(environ) -> list:
    external_graylog = environ.get("EXTERNAL_GRAYLOG_SERVER")
    ssh_key = _read_secret_file(environ, "ssh-key")
    vm_user = _read_secret_file(environ, "vm-user")
    if external_graylog == "false" or not ssh_key or not vm_user:
        return ["archiving-plugin"]
    return []
