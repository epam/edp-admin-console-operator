package service

import (
	"admin-console-operator/pkg/apis/edp/v1alpha1"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AdminConsoleService interface {
	// This is an entry point for service package. Invoked in err = r.service.Install(*instance) sonar_controller.go, Reconcile method.
	Install(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	Configure(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	ExposeConfiguration(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
}

func NewAdminConsoleService(platformService PlatformService, k8sClient client.Client) AdminConsoleService {
	return AdminConsoleServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

type AdminConsoleServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService PlatformService
	k8sClient       client.Client
}

func (s AdminConsoleServiceImpl) Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	deployConf, err := s.platformService.GetDeployConf(instance)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("Failed to get Deployment Config for %s!", instance.Name))
	}

	dbEnvironmentValue, err := s.platformService.GenerateDbSettings(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to generate environment variables for shared database!" )
	}

	keycloakEnvironmentValue, err := s.platformService.GenerateKeycloakSettings(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to generate environment variables for Keycloack!")
	}

	adminConsoleEnvironment := append(dbEnvironmentValue, keycloakEnvironmentValue...)

	err = s.platformService.PatchDeployConfEnv(instance, deployConf, adminConsoleEnvironment)
	if err != nil {
		return &instance, nil
	}

	result, err := s.platformService.UpdateAdminConsole(instance)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("Failed to update Admin Console %s!", instance.Name))
	}

	return result, nil
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

	AdminConsoleReader := v1alpha1.ExternalConfigurationItem{
		Name:        "admin-console-reader",
		Kind:        "Secret",
		Description: "Credentials for admin console reader user",
	}

	adminConsoleCreatorPassword := uniuri.New()
	adminConsoleCreatorCredentials := map[string][]byte{
		"username": []byte("admin-console-creator"),
		"password": []byte(adminConsoleCreatorPassword),
	}

	err = s.platformService.CreateSecret(instance, "admin-console-creator", adminConsoleCreatorCredentials)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to create credentials for Admin Console read user.")
	}

	AdminConsoleCreator := v1alpha1.ExternalConfigurationItem{
		Name:        "admin-console-creator",
		Kind:        "Secret",
		Description: "Credentials for admin console creator user",
	}

	adminConsoleClientPassword := uniuri.New()
	adminConsoleClientCredentials := map[string][]byte{
		"username": []byte("admin-console-client"),
		"password": []byte(adminConsoleClientPassword),
	}

	err = s.platformService.CreateSecret(instance, "admin-console-client", adminConsoleClientCredentials)

	AdminConsoleClient := v1alpha1.ExternalConfigurationItem{
		Name:        "admin-console-client",
		Kind:        "Secret",
		Description: "Credentials for admin console client",
	}

	DbUserValue := v1alpha1.ExternalConfigurationItem{
		Name:        "admin-console-db",
		Kind:        "Secret",
		Description: "Credentials for shared database",
	}

	newExternalConfig := []v1alpha1.ExternalConfigurationItem{
		AdminConsoleReader,
		AdminConsoleCreator,
		AdminConsoleClient,
		DbUserValue,
	}

	if reflect.DeepEqual(newExternalConfig, instance.Spec.ExternalConfiguration) {
		return &instance, nil
	}

	instance.Spec.ExternalConfiguration = newExternalConfig

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
	log.Printf("Starting installation for Admin console")

	sa, err := s.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, logErrorAndReturn(err)
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

	err = s.platformService.CreateSecurityContext(instance, sa)
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateUserRole(instance)
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateUserRoleBinding(instance,"edp-resources-admin","edp-resources-admin","Role")
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateUserRoleBinding(instance, "edp-admin", "admin", "ClusterRole")
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateService(instance)
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	err = s.platformService.CreateDeployConf(instance)
	if err != nil {
		return &instance, logErrorAndReturn(err)
	}

	result, err := s.platformService.UpdateAdminConsole(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to update Admin Console after installation!")
	}

	return result, nil
}
