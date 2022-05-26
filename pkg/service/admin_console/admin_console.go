package admin_console

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dchest/uniuri"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	adminConsoleSpec "github.com/epam/edp-admin-console-operator/v2/pkg/service/admin_console/spec"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/helper"
)

const (
	imgFolder = "img"
	acIcon    = "admin-console.svg"
)

type AdminConsoleService interface {
	ExposeConfiguration(instance adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error)
	Integrate(instance adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error)
	IsDeploymentReady(instance adminConsoleApi.AdminConsole) (bool, error)
}

func NewAdminConsoleService(ps platform.PlatformService, client client.Client, scheme *runtime.Scheme) AdminConsoleService {
	return AdminConsoleServiceImpl{
		platformService: ps,
		keycloakHelper:  keycloakHelper.MakeHelper(client, scheme, ctrl.Log.WithName("admin_console_service")),
	}
}

type AdminConsoleServiceImpl struct {
	// Providing sonar service implementation through the interface (platform abstract)
	platformService platform.PlatformService
	keycloakHelper  *keycloakHelper.Helper
}

func (s AdminConsoleServiceImpl) Integrate(instance adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error) {

	if instance.Spec.KeycloakSpec.Enabled {

		keycloakClient, err := s.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to get Keycloak client data!")
		}

		keycloakRealm, err := s.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
		if err != nil {
			return &instance, errors.Wrap(err, "unable to get keycloak realm cr")
		}

		if keycloakRealm == nil {
			return &instance, errors.New("Keycloak CR is not created yet!")
		}

		keycloak, err := s.keycloakHelper.GetOwnerKeycloak(keycloakRealm.ObjectMeta)
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

func (s AdminConsoleServiceImpl) ExposeConfiguration(instance adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error) {
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
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to create secret")
		}

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
		keycloakClient.Spec.DefaultClientScopes = []string{"edp"}

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

func (s AdminConsoleServiceImpl) createEDPComponent(ac adminConsoleApi.AdminConsole) error {
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

func (s AdminConsoleServiceImpl) getUrl(ac adminConsoleApi.AdminConsole) (*string, error) {
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

func (s AdminConsoleServiceImpl) IsDeploymentReady(instance adminConsoleApi.AdminConsole) (bool, error) {
	return s.platformService.IsDeploymentReady(instance)
}
