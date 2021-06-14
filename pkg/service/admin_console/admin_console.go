package admin_console

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	adminConsoleSpec "github.com/epam/edp-admin-console-operator/v2/pkg/service/admin_console/spec"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imgFolder = "img"
	acIcon    = "admin-console.svg"
)

type AdminConsoleService interface {
	ExposeConfiguration(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error)
	IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error)
}

func NewAdminConsoleService(ps platform.PlatformService, client client.Client, scheme *runtime.Scheme) AdminConsoleService {
	return AdminConsoleServiceImpl{
		platformService: ps,
		keycloakHelper:  keycloakHelper.MakeHelper(client, scheme),
		log:             ctrl.Log.WithName("sso-integration"),
	}
}

type AdminConsoleServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService platform.PlatformService
	keycloakHelper  *keycloakHelper.Helper
	log             logr.Logger
}

func (s AdminConsoleServiceImpl) Integrate(instance v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {
	s.log.Info("Integration method is invoked", "keycloak enabled", instance.Spec.KeycloakSpec.Enabled)

	if instance.Spec.KeycloakSpec.Enabled {

		keycloakClient, err := s.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to get Keycloak client data!")
		}
		s.log.Info("keycloak client is gotten", "val", keycloakClient.Name)

		keycloakRealm, err := s.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
		if err != nil {
			s.log.Info("ERRRRROR")
			return &instance, nil
		}
		s.log.Info("keycloak realm owner is gotten", "val", keycloakRealm.Name)

		if keycloakRealm == nil {
			return &instance, errors.New("Keycloak CR is not created yet!")
		}

		keycloak, err := s.keycloakHelper.GetOwnerKeycloak(keycloakRealm.ObjectMeta)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)
			return &instance, errors.Wrap(err, errMsg)
		}
		s.log.Info("keycloak owner is gotten", "val", keycloak.Name)

		if keycloak == nil {
			return &instance, errors.New("Keycloak CR is not created yet!")
		}

		dbEnvironmentValue, err := s.platformService.GenerateDbSettings(instance)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to generate environment variables for shared database!")
		}
		s.log.Info("db envs", "vals", dbEnvironmentValue)

		discoveryUrl := fmt.Sprintf("%s/auth/realms/%s", keycloak.Spec.Url, keycloakRealm.Spec.RealmName)
		keycloakEnvironmentValue, err := s.platformService.GenerateKeycloakSettings(instance, discoveryUrl)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to generate environment variables for Keycloack!")
		}
		s.log.Info("keycloak envs", "vals", keycloakEnvironmentValue)

		adminConsoleEnvironment := append(dbEnvironmentValue, keycloakEnvironmentValue...)
		s.log.Info("ac envs", "vals", adminConsoleEnvironment)

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

		u, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return &instance, errors.Wrapf(err, "Failed to get Route %s!", instance.Name)
		}

		keycloakClient := keycloakV1Api.KeycloakClient{}
		keycloakClient.Name = instance.Name
		keycloakClient.Namespace = instance.Namespace
		keycloakClient.Spec.ClientId = adminConsoleSpec.DefaultKeycloakSecretName
		keycloakClient.Spec.DirectAccess = true
		keycloakClient.Spec.WebUrl = *u
		keycloakClient.Spec.Secret = adminConsoleSpec.DefaultKeycloakSecretName
		keycloakClient.Spec.AudRequired = true
		keycloakClient.Spec.ServiceAccount = &keycloakV1Api.ServiceAccount{Enabled: true,
			RealmRoles: []string{"developer"}}

		err = s.platformService.CreateKeycloakClient(&keycloakClient)
		if err != nil {
			return &instance, errors.Wrapf(err, "Failed to create Keycloak Client!")
		}

	}

	result, err := s.platformService.UpdateAdminConsole(instance)
	if err != nil {
		return &instance, errors.Wrap(err, fmt.Sprintf("Failed to update Admin Console %s!", instance.Name))
	}

	err = s.createEDPComponent(instance)
	return result, err
}

func (s AdminConsoleServiceImpl) createEDPComponent(ac v1alpha1.AdminConsole) error {
	url, err := s.getUrl(ac)
	if err != nil {
		return err
	}

	icon, err := s.getIcon()
	if err != nil {
		return err
	}

	return s.platformService.CreateEDPComponentIfNotExist(ac, *url, *icon)
}

func (s AdminConsoleServiceImpl) getUrl(ac v1alpha1.AdminConsole) (*string, error) {
	u, err := s.platformService.GetExternalUrl(ac.Namespace, ac.Name)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (j AdminConsoleServiceImpl) getIcon() (*string, error) {
	p, err := platformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, err
	}

	fp := fmt.Sprintf("%v/%v", p, acIcon)
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}

func (s AdminConsoleServiceImpl) IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error) {
	return s.platformService.IsDeploymentReady(instance)
}
