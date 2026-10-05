package fluentbit

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectorLogExclusions(t *testing.T) {
	for _, runtime := range []string{"containerd", "cri-o", "docker"} {
		for _, collect := range []bool{false, true} {
			for _, custom := range []string{"", "/custom/*.log,/other/*.log"} {
				t.Run(fmt.Sprintf("%s/collect=%t/custom=%t", runtime, collect, custom != ""), func(t *testing.T) {
					fb := &loggingService.Fluentbit{ContainerLogging: true, CollectOwnLogs: collect, ExcludePath: custom, Aggregator: &loggingService.FluentbitAggregator{Install: false}}

					cr := &loggingService.LoggingService{ObjectMeta: metav1.ObjectMeta{Namespace: "observability"}, Spec: loggingService.LoggingServiceSpec{Fluentbit: fb}}
					secret, err := fluentbitConfigSecret(cr, util.DynamicParameters{ContainerRuntimeType: runtime}, outputCredentials{})
					data := map[string]string{}
					if err == nil {
						for k, v := range secret.Data {
							data[k] = string(v)
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					input := data["input-containerd.conf"]
					if runtime == "docker" {
						input = data["input-docker.conf"]
					}
					exclusions := ""
					for _, line := range strings.Split(input, "\n") {
						fields := strings.Fields(line)
						if len(fields) == 2 && fields[0] == "Exclude_Path" {
							exclusions = fields[1]
						}
					}
					if collect {
						if exclusions != custom {
							t.Fatalf("exclusions = %q, want %q", exclusions, custom)
						}
						return
					}
					expected := []string{}
					if custom != "" {
						expected = append(expected, custom)
					}
					names := []string{"logging-fluentbit"}
					for _, name := range names {
						pattern := fmt.Sprintf("/var/log/pods/observability_%s-*_*/%s/*.log", name, name)
						own := fmt.Sprintf("/var/log/pods/observability_%s-abc_uid/%s/0.log", name, name)
						other := fmt.Sprintf("/var/log/pods/other_%s-abc_uid/%s/0.log", name, name)
						app := "/var/log/pods/observability_application-abc_uid/app/0.log"
						if runtime == "docker" {
							pattern = fmt.Sprintf("/var/log/containers/%s-*_observability_%s-*.log", name, name)
							own = fmt.Sprintf("/var/log/containers/%s-abc_observability_%s-id.log", name, name)
							other = fmt.Sprintf("/var/log/containers/%s-abc_other_%s-id.log", name, name)
							app = "/var/log/containers/application-abc_observability_app-id.log"
						}
						expected = append(expected, pattern)
						for _, file := range []string{own, other, app} {
							matched, err := filepath.Match(pattern, file)
							if err != nil || matched != (file == own) {
								t.Errorf("pattern %q file %q matched=%t err=%v", pattern, file, matched, err)
							}
						}
					}
					if exclusions != strings.Join(expected, ",") {
						t.Fatalf("exclusions = %q, want %q", exclusions, strings.Join(expected, ","))
					}
					if fb.ExcludePath != custom {
						t.Fatal("rendering changed user exclusions")
					}
				})
			}
		}
	}
}
