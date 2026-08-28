package fluentbit

import (
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestLoggingService(fluentbit *loggingService.Fluentbit) *loggingService.LoggingService {
	return &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logging-service",
			Namespace: "logging",
		},
		Spec: loggingService.LoggingServiceSpec{
			Fluentbit: fluentbit,
		},
	}
}

func renderConfigMapData(t *testing.T, fluentbit *loggingService.Fluentbit) map[string]string {
	t.Helper()
	cm, err := fluentbitConfigMap(newTestLoggingService(fluentbit), util.DynamicParameters{ContainerRuntimeType: "containerd"})
	if err != nil {
		t.Fatalf("cannot build the ConfigMap: %v", err)
	}
	return cm.Data
}

func TestFluentbitConfigMapStorageDefaults(t *testing.T) {
	data := renderConfigMapData(t, &loggingService.Fluentbit{
		ContainerLogging:   true,
		SystemAuditLogging: true,
		SystemLogging:      true,
		SystemLogType:      "systemd",
	})

	t.Run("flush interval is 5 seconds", func(t *testing.T) {
		if !strings.Contains(data["fluent-bit.conf"], "Flush         5") {
			t.Errorf("expected the default flush interval 5, got:\n%s", data["fluent-bit.conf"])
		}
	})

	t.Run("inputs buffer chunks in memory", func(t *testing.T) {
		for _, name := range []string{"input-containerd.conf", "input-audit.conf", "input-messages-systemd.conf"} {
			if !strings.Contains(data[name], "storage.type       memory") {
				t.Errorf("expected the memory storage type in %s, got:\n%s", name, data[name])
			}
			if strings.Contains(data[name], "storage.type       filesystem") {
				t.Errorf("unexpected the filesystem storage type in %s, got:\n%s", name, data[name])
			}
		}
	})

	t.Run("inputs bound the memory buffer", func(t *testing.T) {
		for _, name := range []string{"input-containerd.conf", "input-audit.conf", "input-messages-systemd.conf"} {
			if !strings.Contains(data[name], "Mem_Buf_Limit      10M") {
				t.Errorf("expected the memory buffer limit in %s, got:\n%s", name, data[name])
			}
		}
	})

	t.Run("multiline emitters bound the memory buffer", func(t *testing.T) {
		conf := data["filter-concat.conf"]
		if got := strings.Count(conf, "emitter_mem_buf_limit  32MB"); got != 2 {
			t.Errorf("expected 2 bounded emitters, got %d:\n%s", got, conf)
		}
		if got := strings.Count(conf, "emitter_storage.type   memory"); got != 2 {
			t.Errorf("expected 2 emitters buffering in memory, got %d:\n%s", got, conf)
		}
	})

	t.Run("tail input keeps the database without disk sync", func(t *testing.T) {
		conf := data["input-containerd.conf"]
		for _, expected := range []string{
			"DB                 /var/log/containers.db",
			"DB.journal_mode    memory",
			"DB.sync            off",
			"DB.locking         true",
		} {
			if !strings.Contains(conf, expected) {
				t.Errorf("expected %q in the input config, got:\n%s", expected, conf)
			}
		}
	})

	t.Run("systemd input does not use tail-only database parameters", func(t *testing.T) {
		conf := data["input-messages-systemd.conf"]
		if !strings.Contains(conf, "DB.sync            off") {
			t.Errorf("expected the database sync parameter, got:\n%s", conf)
		}
		for _, unexpected := range []string{"DB.journal_mode", "DB.locking"} {
			if strings.Contains(conf, unexpected) {
				t.Errorf("unexpected %q in the systemd input config, got:\n%s", unexpected, conf)
			}
		}
	})
}

func TestFluentbitConfigMapStorageOverrides(t *testing.T) {
	disabled := false
	data := renderConfigMapData(t, &loggingService.Fluentbit{
		ContainerLogging: true,
		Flush:            10,
		StorageType:      "filesystem",
		InputMemBufLimit: "25M",
		DB: loggingService.FluentbitDB{
			JournalMode: "WAL",
			Sync:        "normal",
			Locking:     &disabled,
		},
	})

	if !strings.Contains(data["fluent-bit.conf"], "Flush         10") {
		t.Errorf("expected the flush interval from the CR, got:\n%s", data["fluent-bit.conf"])
	}

	conf := data["input-containerd.conf"]
	for _, expected := range []string{
		"storage.type       filesystem",
		"Mem_Buf_Limit      25M",
		"DB.journal_mode    WAL",
		"DB.sync            normal",
		"DB.locking         false",
	} {
		if !strings.Contains(conf, expected) {
			t.Errorf("expected %q in the input config, got:\n%s", expected, conf)
		}
	}

	if got := strings.Count(data["filter-concat.conf"], "emitter_storage.type   filesystem"); got != 2 {
		t.Errorf("expected 2 emitters buffering on the disk, got %d:\n%s", got, data["filter-concat.conf"])
	}
}

func TestFluentbitConfigMapDisabledDB(t *testing.T) {
	disabled := false
	data := renderConfigMapData(t, &loggingService.Fluentbit{
		ContainerLogging:   true,
		SystemAuditLogging: true,
		SystemLogging:      true,
		SystemLogType:      "systemd",
		DB:                 loggingService.FluentbitDB{Enabled: &disabled},
	})

	for _, name := range []string{"input-containerd.conf", "input-audit.conf", "input-messages-systemd.conf"} {
		if strings.Contains(data[name], "DB") {
			t.Errorf("expected no database parameters in %s, got:\n%s", name, data[name])
		}
	}
}
