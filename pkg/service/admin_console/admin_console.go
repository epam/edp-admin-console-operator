package admin_console

import (
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	adminConsoleSpec "github.com/epmd-edp/admin-console-operator/v2/pkg/service/admin_console/spec"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakControllerHelper "github.com/epmd-edp/keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConsoleService interface {
	// This is an entry point for service package. Invoked in err = r.service.Install(*instance) sonar_controller.go, Reconcile method.
	Install(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	Configure(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	ExposeConfiguration(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error)
}

func NewAdminConsoleService(platformService platform.PlatformService, k8sClient client.Client) AdminConsoleService {
	return AdminConsoleServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type AdminConsoleServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService platform.PlatformService
	k8sClient       client.Client
}

func (s AdminConsoleServiceImpl) Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	if instance.Spec.KeycloakSpec.Enabled {

		keycloakClient, err := s.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to get Keycloak client data!")
		}

		keycloakRealm, err := keycloakControllerHelper.GetOwnerKeycloakRealm(s.k8sClient, keycloakClient.ObjectMeta)
		if err != nil {
			return &instance, nil
		}

		if keycloakRealm == nil {
			return &instance, errors.New("Keycloak CR is not created yet!")
		}

		keycloak, err := keycloakControllerHelper.GetOwnerKeycloak(s.k8sClient, keycloakRealm.ObjectMeta)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)
			return &instance, errors.Wrap(err, errMsg)
		}

		if keycloak == nil {
			return &instance, errors.New("Keycloak CR is not created yet!")
		}

		dbEnvironmentValue, err := s.platformService.GenerateDbSettings(instance)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to generate environment variables for shared database!")
		}

		discoveryUrl := fmt.Sprintf("%s/auth/realms/%s", keycloak.Spec.Url, keycloakRealm.Spec.RealmName)
		keycloakEnvironmentValue, err := s.platformService.GenerateKeycloakSettings(instance, discoveryUrl)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to generate environment variables for Keycloack!")
		}

		adminConsoleEnvironment := append(dbEnvironmentValue, keycloakEnvironmentValue...)

		err = s.platformService.PatchDeploymentEnv(instance, adminConsoleEnvironment)
		if err != nil {
			return &instance, nil
		}

		result, err := s.platformService.UpdateAdminConsole(instance)
		if err != nil {
			return &instance, errors.Wrap(err, fmt.Sprintf("Failed to update Admin Console %s!", instance.Name))
		}
		return result, nil
	}

	return &instance, nil
}

func (s AdminConsoleServiceImpl) ExposeConfiguration(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	adminConsoleReaderPassword := uniuri.New()
	adminConsoleReaderCredentials := map[string][]byte{
		"username": []byte("admin-console-reader"),
		"password": []byte(adminConsoleReaderPassword),
	}

	err := s.platformService.CreateSecret(instance, "admin-console-reader", adminConsoleReaderCredentials)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to create credentials for Admin Console read user.")
	}

	if instance.Spec.KeycloakSpec.Enabled {

		adminConsoleClientPassword := uniuri.New()
		adminConsoleClientCredentials := map[string][]byte{
			"username":     []byte(adminConsoleSpec.DefaultKeycloakSecretName),
			"password":     []byte(adminConsoleClientPassword),
			"clientSecret": []byte(adminConsoleClientPassword),
		}

		err = s.platformService.CreateSecret(instance, adminConsoleSpec.DefaultKeycloakSecretName, adminConsoleClientCredentials)

		webUrl, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return &instance, errors.Wrapf(err, "Failed to get Route %s!", instance.Name)
		}

		keycloakClient := keycloakV1Api.KeycloakClient{}
		keycloakClient.Name = instance.Name
		keycloakClient.Namespace = instance.Namespace
		keycloakClient.Spec.ClientId = adminConsoleSpec.DefaultKeycloakSecretName
		keycloakClient.Spec.DirectAccess = true
		keycloakClient.Spec.WebUrl = webUrl
		keycloakClient.Spec.Secret = adminConsoleSpec.DefaultKeycloakSecretName

		err = s.platformService.CreateKeycloakClient(&keycloakClient)
		if err != nil {
			return &instance, errors.Wrapf(err, "Failed to create Keycloak Client!")
		}

	}

	result, err := s.platformService.UpdateAdminConsole(instance)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("Failed to update Admin Console %s!", instance.Name))
	}

	return result, nil
}

func (s AdminConsoleServiceImpl) Configure(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	err := s.platformService.AddServiceAccToSecurityContext("anyuid", instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to add user in anyuid Security Context.")
	}

	return &instance, nil
}

func (s AdminConsoleServiceImpl) Install(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	err := s.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, err
	}

	dbAdminPassword := uniuri.New()
	dbAdminSecret := map[string][]byte{
		"username": []byte(fmt.Sprintf("admin-%s", instance.Spec.EdpSpec.Name)),
		"password": []byte(dbAdminPassword),
	}

	err = s.platformService.CreateSecret(instance, "admin-console-db", dbAdminSecret)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to create admin credentials for tenant database.")
	}

	err = s.platformService.CreateSecurityContext(instance)
	if err != nil {
		return &instance, err
	}

	err = s.platformService.CreateUserRole(instance)
	if err != nil {
		return &instance, err
	}

	err = s.platformService.CreateRoleBinding(instance, "edp-resources-admin", "edp-resources-admin")
	if err != nil {
		return &instance, err
	}

	err = s.platformService.CreateClusterRoleBinding(instance, "edp-admin", "admin")
	if err != nil {
		return &instance, err
	}

	err = s.platformService.CreateService(instance)
	if err != nil {
		return &instance, err
	}

	err = s.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return &instance, err
	}

	webUrl, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to get Route %s!", instance.Name)
	}

	err = s.platformService.CreateDeployConf(instance, webUrl)
	if err != nil {
		return &instance, err
	}

	return &instance, nil
}

func (s AdminConsoleServiceImpl) IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error) {
	return s.platformService.IsDeploymentReady(instance)
}
