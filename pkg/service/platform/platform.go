package platform

import (
	"fmt"
	"strings"

	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/kubernetes"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/openshift"
)

type PlatformService interface {
	CreateSecret(ac adminConsoleApi.AdminConsole, name string, data map[string][]byte) error
	GenerateDbSettings(ac adminConsoleApi.AdminConsole) ([]coreV1Api.EnvVar, error)
	GenerateKeycloakSettings(ac adminConsoleApi.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error)
	PatchDeploymentEnv(ac adminConsoleApi.AdminConsole, env []coreV1Api.EnvVar) error
	UpdateAdminConsole(ac adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error)
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error)
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetExternalUrl(namespace string, name string) (*string, error)
	IsDeploymentReady(instance adminConsoleApi.AdminConsole) (bool, error)
}

const (
	Openshift  string = "openshift"
	Kubernetes string = "kubernetes"
)

func NewPlatformService(platformType string, scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(platformType) {
	case Kubernetes:
		platformService := kubernetes.K8SService{}
		err = platformService.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to initialize Kubernetes platform service!")
		}

		return platformService, nil
	case Openshift:
		platformService := openshift.OpenshiftService{}
		err = platformService.Init(restConfig, scheme, k8sClient)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to initialize OpenShift platform service!")
		}

		return platformService, nil
	default:
		err := errors.New(fmt.Sprintf("Platform %s is not supported!", platformType))
		return nil, err
	}
}
