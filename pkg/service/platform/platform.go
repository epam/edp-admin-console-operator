package platform

import (
	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/openshift"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	appsV1Api "github.com/openshift/api/apps/v1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PlatformService interface {
	AddServiceAccToSecurityContext(scc string, ac v1alpha1.AdminConsole) error
	CreateDeployConf(ac v1alpha1.AdminConsole, url string) error
	CreateSecret(ac v1alpha1.AdminConsole, name string, data map[string][]byte) error
	CreateExternalEndpoint(ac v1alpha1.AdminConsole) error
	CreateService(ac v1alpha1.AdminConsole) error
	CreateServiceAccount(ac v1alpha1.AdminConsole) error
	CreateSecurityContext(ac v1alpha1.AdminConsole) error
	CreateUserRole(ac v1alpha1.AdminConsole) error
	CreateUserRoleBinding(ac v1alpha1.AdminConsole, name string, binding string, kind string) error
	GetConfigmap(namespace string, name string) (map[string]string, error)
	GetDisplayName(ac v1alpha1.AdminConsole) (string, error)
	GetSecret(namespace string, name string) (map[string][]byte, error)
	GetAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	GetDeployConf(ac v1alpha1.AdminConsole) (*appsV1Api.DeploymentConfig, error)
	GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error)
	GenerateKeycloakSettings(ac v1alpha1.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error)
	PatchDeploymentEnv(ac v1alpha1.AdminConsole, env []coreV1Api.EnvVar) error
	UpdateAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient,error)
	CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error
	GetExternalUrl(namespace string, name string) (string, string, error)
	GetDeploymentConfig(instance v1alpha1.AdminConsole) (*appsV1Api.DeploymentConfig, error)
}

func NewPlatformService(scheme *runtime.Scheme, k8sClient *client.Client) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	platform := openshift.OpenshiftService{}

	err = platform.Init(restConfig, scheme, k8sClient)
	if err != nil {
		return nil, err
	}
	return platform, nil
}

