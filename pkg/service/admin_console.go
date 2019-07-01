package service

import (
	"admin-console-operator/pkg/apis/edp/v1alpha1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConsoleService interface {
	// This is an entry point for service package. Invoked in err = r.service.Install(*instance) sonar_controller.go, Reconcile method.
	Install(instance v1alpha1.AdminConsole) error
	Configure(instance v1alpha1.AdminConsole) error
	ExposeConfiguration() error
	Integration() error
}

func NewAdminConsoleService(platformService PlatformService, k8sClient client.Client) AdminConsoleService {
	return AdminConsoleServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type AdminConsoleServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService PlatformService
	k8sClient       client.Client
}

func (s AdminConsoleServiceImpl) Configure(instance v1alpha1.AdminConsole) error {
	return nil
}

func (s AdminConsoleServiceImpl) ExposeConfiguration() error {
	return nil
}

func (s AdminConsoleServiceImpl) Integration() error {
	return nil
}

func (s AdminConsoleServiceImpl) Install(instance v1alpha1.AdminConsole) error {
	log.Printf("Starting installation for Admin console")

	sa, err := s.platformService.CreateServiceAccount(instance)
	if err != nil {
		return logErrorAndReturn(err)
	}

	err = s.platformService.CreateSecurityContext(instance, sa)
	if err != nil {
		return logErrorAndReturn(err)
	}

	err = s.platformService.CreateUserRole(instance)
	if err != nil {
		return logErrorAndReturn(err)
	}


	err = s.platformService.CreateUserRoleBinding(instance, "resources-admin")
	if err != nil {
		return logErrorAndReturn(err)
	}

	err = s.platformService.CreateUserRoleBinding(instance, "admin")
	if err != nil {
		return logErrorAndReturn(err)
	}


	err = s.platformService.CreateService(instance)
	if err != nil {
		return logErrorAndReturn(err)
	}

	err = s.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return logErrorAndReturn(err)
	}

	err = s.platformService.CreateDeployConf(instance)
	if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}
