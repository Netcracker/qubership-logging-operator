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

func renderConfigData(t *testing.T, fluentbit *loggingService.Fluentbit) map[string]string {
	t.Helper()
	secret, err := fluentbitConfigSecret(newTestLoggingService(fluentbit),
		util.DynamicParameters{ContainerRuntimeType: "containerd"}, outputCredentials{})
	if err != nil {
		t.Fatalf("cannot build the config Secret: %v", err)
	}
	data := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		data[key] = string(value)
	}
	return data
}

func TestFluentbitConfigStorageDefaults(t *testing.T) {
	data := renderConfigData(t, &loggingService.Fluentbit{
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

	t.Run("tail input keeps the database without disk sync", func(t *testing.T) {
		conf := data["input-containerd.conf"]
		for _, expected := range []string{
			"DB                 /fluent-bit/state/containers.db",
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
		for _, expected := range []string{"DB.sync            off", "Mem_Buf_Limit      10M"} {
			if !strings.Contains(conf, expected) {
				t.Errorf("expected %q in the systemd input config, got:\n%s", expected, conf)
			}
		}
		for _, unexpected := range []string{"DB.journal_mode", "DB.locking"} {
			if strings.Contains(conf, unexpected) {
				t.Errorf("unexpected %q in the systemd input config, got:\n%s", unexpected, conf)
			}
		}
	})
}

func TestFluentbitPersistentOffsetsProfileUsesMemoryBuffer(t *testing.T) {
	data := renderConfigData(t, &loggingService.Fluentbit{
		ContainerLogging: true,
		StorageProfile:   loggingService.FluentbitStorageProfilePersistentOffsets,
	})

	if !strings.Contains(data["input-containerd.conf"], "storage.type       memory") {
		t.Errorf("expected memory input storage, got:\n%s", data["input-containerd.conf"])
	}
	if strings.Contains(data["fluent-bit.conf"], "storage.path") {
		t.Errorf("persistent-offsets must not configure filesystem buffering, got:\n%s", data["fluent-bit.conf"])
	}
	if !strings.Contains(data["filter-concat.conf"], "emitter_storage.type   memory") {
		t.Errorf("persistent-offsets must keep multiline emitters in memory, got:\n%s", data["filter-concat.conf"])
	}
}

func TestFluentbitNodePersistentProfileUsesFilesystemBuffer(t *testing.T) {
	data := renderConfigData(t, &loggingService.Fluentbit{
		ContainerLogging: true,
		StorageProfile:   loggingService.FluentbitStorageProfileNodePersistent,
	})

	if !strings.Contains(data["input-containerd.conf"], "storage.type       filesystem") {
		t.Errorf("expected filesystem input storage, got:\n%s", data["input-containerd.conf"])
	}
	if !strings.Contains(data["fluent-bit.conf"], "storage.path                         /fluent-bit/storage/") {
		t.Errorf("expected dedicated filesystem buffer path, got:\n%s", data["fluent-bit.conf"])
	}
	if strings.Count(data["filter-concat.conf"], "emitter_storage.type   filesystem") != 2 {
		t.Errorf("node-persistent must store both multiline emitters on the filesystem, got:\n%s",
			data["filter-concat.conf"])
	}
}

func TestFluentbitConfigStorageOverrides(t *testing.T) {
	disabled := false
	data := renderConfigData(t, &loggingService.Fluentbit{
		ContainerLogging: true,
		Flush:            10,
		StorageProfile:   loggingService.FluentbitStorageProfileNodePersistent,
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
		"DB.journal_mode    WAL",
		"DB.sync            normal",
		"DB.locking         false",
	} {
		if !strings.Contains(conf, expected) {
			t.Errorf("expected %q in the input config, got:\n%s", expected, conf)
		}
	}
}

func TestFluentbitConfigDisabledDB(t *testing.T) {
	disabled := false
	data := renderConfigData(t, &loggingService.Fluentbit{
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
