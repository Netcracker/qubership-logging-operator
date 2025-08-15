## Table of Contents
- [Logging on different OS](#logging-on-different-os)
  * [System logs](#system-logs)
  * [Audit logs](#audit-logs)
  * [Kubernetes and container logs](#kubernetes-and-container-logs)

### Logging on different OS

Logging agents (Fluentd, FluentBit) in the logging-service are configured to scrape logs from certain log files
from the node: system logs, audit logs, kube logs, containers logs.
But some OS have different locations for these files or may not contain them at all.

#### System logs

Different OS have different locations for their system logs files. The most important system logs are global log journal
(`/var/log/syslog` by (r)syslogd, `/var/log/messages` by systemd, `/var/log/journal` by systemd-journald)
and auth logs (`/var/log/auth.log`, `/var/log/secure`).

The following table contains frequently used and recommended OS and paths to system logs for them:

<!-- markdownlint-disable line-length -->
| OS name                                | OS versions        | Global system logs                                    | Auth logs            |
| -------------------------------------- | ------------------ | ----------------------------------------------------- | -------------------- |
| Ubuntu                                 | 20.04.x, 22.04.x   | /var/log/syslog (/var/log/journal is available too)   | /var/log/auth.log    |
| Rocky Linux                            | 9.x                | /var/log/messages                                     | /var/log/secure      |
| CentOS                                 | 8.x                | /var/log/messages                                     | /var/log/secure      |
| RHEL                                   | 8.x                | /var/log/messages                                     | /var/log/secure      |
| Oracle Linux                           | 8.x                | /var/log/messages                                     | /var/log/secure      |
| Azure Linux (CBL-Mariner)              | 2.x                | /var/log/journal                                      | /var/log/journal     |
| Amazon Linux                           | 2.x                | /var/log/messages (/var/log/journal is available too) | /var/log/secure      |
| BottleRocket OS                        | 1.x                | /var/log/journal                                      | not present[^1]      |
| COS (Container-Optimized OS by Google) | 101, 105, 109, 113 | /var/log/journal (?)[^2]                              | /var/log/journal (?) |
<!-- markdownlint-enable line-length -->

 [^1]: BottleRocket is an OS created specifically for hosting containers, and it doesn't have a standard shell.
You can manage the BottleRocket OS only through a special in-built container with privileged rights,
so auth logs on the host would be useless for such concept.

 [^2]: **COS uses journald** as a main solution for system logs, and most likely the logs are located in
the default path for journald.

#### Audit logs

Audit logs are managed by `auditd` daemon that is installed by default on the most OS, but there are several exceptions.

Audit logs by `auditd` are always located on `/var/log/audit/audit.log` by default.

The following table describes which OS have auditd by default:

<!-- markdownlint-disable line-length -->
| OS name         | Is auditd present by default                                                                                                                                           |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Ubuntu          | ✓ Yes                                                                                                                                                                  |
| Rocky Linux     | ✓ Yes                                                                                                                                                                  |
| CentOS          | ✓ Yes                                                                                                                                                                  |
| RHEL            | ✓ Yes                                                                                                                                                                  |
| Oracle Linux    | ✓ Yes                                                                                                                                                                  |
| Azure Linux     | ✗ No (auditd is not installed by default)                                                                                                                              |
| Amazon Linux    | ✓ Yes                                                                                                                                                                  |
| BottleRocket OS | ✗ No (auditd is not presented due the lack of the shell)                                                                                                               |
| COS             | ✗ No (disabled by default, [can be installed by using the special DaemonSet with auditd](https://cloud.google.com/kubernetes-engine/docs/how-to/linux-auditd-logging)) |
<!-- markdownlint-enable line-length -->

#### Kubernetes and container logs

The location of Kubernetes and containers logs is independent of the OS the node is running on.

The location of Kubernetes logs depends on the Kubernetes version and the type of k8s cluster (pure Kubernetes,
OpenShift).

The location of containers logs depends on the container engine (docker, containerd, cri-o).