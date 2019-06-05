package service

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConsoleService interface {
	// This is an entry point for service package. Invoked in err = r.service.Install(*instance) sonar_controller.go, Reconcile method.
	Install() error
	Configure() error
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

func (s AdminConsoleServiceImpl) Configure() error {
	return nil
}

func (s AdminConsoleServiceImpl) ExposeConfiguration() error {
	return nil
}

func (s AdminConsoleServiceImpl) Integration() error {
	return nil
}

func (s AdminConsoleServiceImpl) Install() error {
	return nil
}

