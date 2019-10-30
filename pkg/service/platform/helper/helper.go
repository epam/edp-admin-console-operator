package helper

import (
	"fmt"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/helper"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"github.com/totherme/unstructured"
	coreV1Api "k8s.io/api/core/v1"
	"net/url"
)

const (
	defaultConfigsAbsolutePath        = "/usr/local/configs/"
	localConfigsRelativePath          = "configs"
)

func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}


// webConsoleConfiguration defines required properties of a data structure used by YAML-formatted payload
// of the openshift-web-console/webconsole-config ConfigMap object
type WebConsoleConfiguration struct {
	ClusterInfo struct {
		ConsolePublicURL string `yaml:"consolePublicURL"`
	} `yaml:"clusterInfo"`
}

// parseWebConsoleConfig unmarshals YAML-formatted data into webConsoleConfiguration object
func ParseWebConsoleConfig(data string) (*WebConsoleConfiguration, error) {
	config := WebConsoleConfiguration{}

	responseYaml, err := unstructured.ParseYAML(data)
	if err != nil {
		return &config, err
	}

	if ok := responseYaml.HasKey("clusterInfo"); ok {
		clusterInfo, _ := responseYaml.GetByPointer("/clusterInfo")
		config.ClusterInfo.ConsolePublicURL, err = clusterInfo.F("consolePublicURL").StringValue()
		if err != nil {
			return &config, nil
		}
		return &config, nil
	}
	// Success
	return &config, nil
}

// stripClusterURL returns ClusterURL as url parameter value without relative path
func StripClusterURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse a URL string")
	}
	// Success
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func SelectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	out := coreV1Api.Container{}
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}

	return out, errors.New("No matching container in spec found!")
}

func UpdateEnv(existing []coreV1Api.EnvVar, env []coreV1Api.EnvVar) []coreV1Api.EnvVar {
	var out []coreV1Api.EnvVar
	var covered []string

	for _, e := range existing {
		newer, ok := findEnv(env, e.Name)
		if ok {
			covered = append(covered, e.Name)
			out = append(out, newer)
			continue
		}
		out = append(out, e)
	}
	for _, e := range env {
		if StringInSlice(e.Name, covered) {
			continue
		}
		covered = append(covered, e.Name)
		out = append(out, e)
	}
	return out
}

func findEnv(env []coreV1Api.EnvVar, name string) (coreV1Api.EnvVar, bool) {
	for _, e := range env {
		if e.Name == name {
			return e, true
		}
	}
	return coreV1Api.EnvVar{}, false
}

func ContainsEmptyString(ss ...string) bool {
	for _, s := range ss {
		if s == "" {
			return true
		}
	}
	return false
}

func CreatePathToTemplateDirectory(directory string) (string, error) {
	localRun := checkIfRunningLocally()
	return createPath(directory, localRun)
}

func checkIfRunningLocally() bool {
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		return true
	}
	return false
}

func createPath(directory string, localRun bool) (string, error) {
	if localRun {
		executableFilePath, err := helper.GetExecutableFilePath()
		if err != nil {
			return "", errors.Wrapf(err, "Unable to get executable file path")
		}
		templatePath := fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, directory)
		return templatePath, nil
	}

	templatePath := fmt.Sprintf("%s/%s", defaultConfigsAbsolutePath, directory)
	return templatePath, nil

}