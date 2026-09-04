package preparing

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	util "github.com/Netcracker/qubership-logging-operator/controllers/utils"

	agents "github.com/Netcracker/qubership-logging-operator/test/fluent-pipeline/agent"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	sourceConfigPath = "/config-templates.d"
	targetConfigPath = "/configuration.d"
)

func PrepareConfiguration(crPath string, agent agents.Agent) {
	slog.Info("Reading test Custom Resource...")
	cr, err := readCustomResource(crPath)
	if err != nil {
		slog.Error("Error occurred when reading Custom Resource", "err", err)
		os.Exit(1)
	}
	slog.Info("Custom Resource read successfully", "LoggingService", cr)
	//fill templates to final config
	data, err := getConfiguration(agent, cr)
	if err != nil {
		slog.Error("Error occurred when reading fluent configuration files", "err", err)
		os.Exit(1)
	}
	for fileName, content := range data {
		slog.Debug("Configuration data", fileName, content)
	}
	slog.Info("Configuration files successfully read and filled")
	//place all files to the shared directory /configuration.d
	err = saveDataToDirectory(targetConfigPath, data)
	if err != nil {
		slog.Error("Error occurred when saving fluent configuration files to shared directory", "err", err)
		os.Exit(1)
	}
	slog.Info("Configuration load finished successfully")
}

func PrepareTestLogs(targetDir string) {
	slog.Info("Preparing logs...")
	// Read /logs and create the container log tree used by the logging agent.
	err := readInputLogs(targetDir)
	if err != nil {
		slog.Error("Error occurred when preparing logs", "err", err)
		os.Exit(1)
	}
	slog.Info("Input logs are ready")
}

func readInputLogs(targetDir string) error {
	logsDir := os.DirFS("/logs")
	return fs.WalkDir(logsDir, ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			bytes, err := fs.ReadFile(logsDir, path)
			if err != nil {
				return err
			}
			//set proper file name and save to target dir
			dirs, fileName := filepath.Split(path)
			var targetFilePath string
			if strings.HasPrefix(dirs, "logs/containers") {
				deploymentName := strings.ReplaceAll(dirs[:len(dirs)-1], "/", "-")
				targetFilePath = filepath.Join("/var/log/pods/", fmt.Sprintf("test-namespace_%s_%s", deploymentName, "100000000000000000000000000000000000"), fileName)
				fileName = "0.log"
			} else {
				slog.Info("Ignoring file", "file", path)
				return nil
			}
			err = os.MkdirAll(filepath.Join(targetDir, targetFilePath), 0o755)
			if err != nil {
				return err
			}
			f, err := os.Create(filepath.Join(targetDir, targetFilePath, fileName))
			if err != nil {
				return err
			}
			_, err = f.Write(bytes)
			if err != nil {
				return err
			}
			if err = f.Close(); err != nil {
				return err
			}
		}
		return nil
	})
}

func readCustomResource(path string) (*loggingService.LoggingService, error) {
	crFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cr *loggingService.LoggingService
	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	_ = loggingService.AddToScheme(sch)
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
	obj, gKV, err := decode(crFile, nil, nil)
	if err != nil {
		slog.Error("could not parse custom resource from file. Probably it contains errors.", "error", err)
		return nil, err
	}
	if gKV.Kind == "LoggingService" {
		cr = obj.(*loggingService.LoggingService)
	}
	return cr, nil
}

func getConfiguration(agent agents.Agent, cr *loggingService.LoggingService) (data map[string]string, err error) {
	data, err = fillConfigurationTemplates(sourceConfigPath, cr.ToParams())
	if err != nil {
		return
	}
	agent.UpdateCustomConfiguration(data, cr)
	return
}

func saveDataToDirectory(dir string, data map[string]string) error {
	for fileName, content := range data {
		file, err := os.Create(path.Join(dir, fileName))
		if err != nil {
			return err
		}
		_, err = file.Write([]byte(content))
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func fillConfigurationTemplates(directoryPath string, parameters interface{}) (map[string]string, error) {
	data := map[string]string{}

	err := fs.WalkDir(os.DirFS(directoryPath), ".", func(filePath string, d fs.DirEntry, err error) error {
		if d == nil {
			return fmt.Errorf("directory %q is not found. Probably you forgot to mount configuration templates", directoryPath)
		}
		if !d.IsDir() {
			_, fileName := path.Split(filePath)
			slog.Debug("Reading configuration template file", "filePath", filePath)
			configPart, err := os.ReadFile(path.Join(directoryPath, filePath))
			if err != nil {
				slog.Error("Failed to read file", "filePath", filePath)
				return err
			}
			data[fileName], err = util.ParseTemplate(string(configPart), filePath, parameters)
			if err != nil {
				slog.Error("Failed to parse template from file", "filePath", filePath)
				return err
			}
			slog.Debug("Configuration file was successfully parsed.", "filePath", filePath)
		}
		return nil
	})
	return data, err
}
