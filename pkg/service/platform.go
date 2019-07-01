package service

import (
	"admin-console-operator/pkg/apis/edp/v1alpha1"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type PlatformService interface {
	CreateDeployConf(ac v1alpha1.AdminConsole) error
	CreateSecret(ac v1alpha1.AdminConsole, name string, data map[string][]byte) error
	CreateExternalEndpoint(ac v1alpha1.AdminConsole) error
	CreateService(ac v1alpha1.AdminConsole) error
	CreateServiceAccount(ac v1alpha1.AdminConsole) (*coreV1Api.ServiceAccount, error)
	CreateSecurityContext(ac v1alpha1.AdminConsole, sa *coreV1Api.ServiceAccount) error
	GetConfigmap(namespace string, name string) (map[string]string, error)
	CreateUserRole(ac v1alpha1.AdminConsole) error
	CreateUserRoleBinding(ac v1alpha1.AdminConsole, name string) error
}

func NewPlatformService(scheme *runtime.Scheme) (PlatformService, error) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, logErrorAndReturn(err)
	}

	platform := OpenshiftService{}

	err = platform.Init(restConfig, scheme)
	if err != nil {
		return nil, logErrorAndReturn(err)
	}
	return platform, nil
}

func logErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

func generateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}
