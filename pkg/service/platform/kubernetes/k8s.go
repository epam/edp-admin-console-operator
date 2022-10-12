package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	authV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-admin-console-operator/v2/pkg/helper"
	platformHelper "github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/helper"
)

var log = ctrl.Log.WithName("platform")

type K8SService struct {
	Scheme             *runtime.Scheme
	CoreClient         coreV1Client.CoreV1Client
	NetworkingV1Client networkingV1Client.NetworkingV1Interface
	client             client.Client
	AppsClient         appsV1Client.AppsV1Client
	AuthClient         authV1Client.RbacV1Client
}

func (service K8SService) GenerateDbSettings(ac adminConsoleApi.AdminConsole) ([]coreV1Api.EnvVar, error) {
	if !ac.Spec.DbSpec.Enabled {
		return []coreV1Api.EnvVar{
			{
				Name:  "DB_ENABLED",
				Value: "false",
			},
		}, nil
	}

	log.V(1).Info("Generating DB settings for Admin Console ",
		"Namespace", ac.Namespace, "Name", ac.Name)
	if platformHelper.ContainsEmptyString(ac.Spec.DbSpec.Name, ac.Spec.DbSpec.Hostname, ac.Spec.DbSpec.Port) {
		return nil, errors.New("One or many DB settings field are empty!")
	}

	return []coreV1Api.EnvVar{
		{
			Name:  "PG_HOST",
			Value: ac.Spec.DbSpec.Hostname,
		},
		{
			Name:  "PG_PORT",
			Value: ac.Spec.DbSpec.Port,
		},
		{
			Name:  "PG_DATABASE",
			Value: ac.Spec.DbSpec.Name,
		},
		{
			Name:  "DB_ENABLED",
			Value: strconv.FormatBool(ac.Spec.DbSpec.Enabled),
		},
	}, nil

}

func (service K8SService) GenerateKeycloakSettings(ac adminConsoleApi.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error) {

	log.V(1).Info("Generating Keycloak settings for Admin Console",
		"Namespace", ac.Namespace, "Name", ac.Name)

	if !ac.Spec.KeycloakSpec.Enabled {
		return []coreV1Api.EnvVar{}, nil
	}

	return []coreV1Api.EnvVar{
		{
			Name: "KEYCLOAK_CLIENT_ID",
			ValueFrom: &coreV1Api.EnvVarSource{
				SecretKeyRef: &coreV1Api.SecretKeySelector{
					LocalObjectReference: coreV1Api.LocalObjectReference{
						Name: "admin-console-client",
					},
					Key: "username",
				},
			},
		},
		{
			Name: "KEYCLOAK_CLIENT_SECRET",
			ValueFrom: &coreV1Api.EnvVarSource{
				SecretKeyRef: &coreV1Api.SecretKeySelector{
					LocalObjectReference: coreV1Api.LocalObjectReference{
						Name: "admin-console-client",
					},
					Key: "password",
				},
			},
		},
		{
			Name:  "KEYCLOAK_URL",
			Value: keycloakUrl,
		},
		{
			Name:  "AUTH_KEYCLOAK_ENABLED",
			Value: strconv.FormatBool(ac.Spec.KeycloakSpec.Enabled),
		},
	}, nil
}

func (service K8SService) PatchDeploymentEnv(ac adminConsoleApi.AdminConsole, env []coreV1Api.EnvVar) error {
	if len(env) == 0 {
		return nil
	}

	dc, err := service.AppsClient.Deployments(ac.Namespace).Get(context.TODO(), ac.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Deployment not found!", "Namespace", ac.Namespace, "Name", ac.Name)
			return nil
		}
		return err
	}

	container, err := platformHelper.SelectContainer(dc.Spec.Template.Spec.Containers, ac.Name)
	if err != nil {
		return err
	}

	container.Env = platformHelper.UpdateEnv(container.Env, env)

	dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	_, err = service.AppsClient.Deployments(dc.Namespace).Patch(context.TODO(), dc.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return err
}

func (service K8SService) GetExternalUrl(namespace string, name string) (*string, error) {
	ingress, err := service.NetworkingV1Client.Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ingress not found", "Namespace", namespace, "Name", name)
			return nil, nil
		}
		return nil, err
	}

	routeScheme := "https"
	u := fmt.Sprintf("%s://%s%s", routeScheme, ingress.Spec.Rules[0].Host, strings.TrimRight(ingress.Spec.Rules[0].HTTP.Paths[0].Path, platformHelper.UrlCutset))

	return &u, nil
}

func (service K8SService) IsDeploymentReady(instance adminConsoleApi.AdminConsole) (bool, error) {
	return helper.IsDeploymentReady(service.AppsClient, instance.Name, instance.Namespace)
}

func (service K8SService) CreateSecret(ac adminConsoleApi.AdminConsole, name string, data map[string][]byte) error {
	labels := platformHelper.GenerateLabels(ac.Name)

	consoleSecretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&ac, consoleSecretObject, service.Scheme); err != nil {
		return err
	}

	_, err := service.CoreClient.Secrets(consoleSecretObject.Namespace).Get(context.TODO(), consoleSecretObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Secret %s/%s for Admin Console", consoleSecretObject.Namespace, consoleSecretObject.Name)
			log.V(1).Info(msg)
			consoleSecret, err := service.CoreClient.Secrets(consoleSecretObject.Namespace).Create(context.TODO(), consoleSecretObject, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Secret %s/%s has been created", consoleSecret.Namespace, consoleSecret.Name))
			// Successfully created
			return nil
		}
		// Some error occurred
		return err
	}
	// Nothing to do
	return nil
}

func (s K8SService) UpdateAdminConsole(ac adminConsoleApi.AdminConsole) (*adminConsoleApi.AdminConsole, error) {
	if err := s.client.Update(context.TODO(), &ac); err != nil {
		return nil, err
	}
	return &ac, nil
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Core Client initialization failed!")
	}

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("appsV1 client initialization failed!")
	}

	networkingClient, err := networkingV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("networkingV1 client initialization failed!")
	}

	rbacClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("extensionsV1 client initialization failed!")
	}

	service.CoreClient = *coreClient
	service.Scheme = scheme
	service.client = *k8sClient
	service.AppsClient = *appsClient
	service.NetworkingV1Client = networkingClient
	service.AuthClient = *rbacClient
	return nil
}

func (service K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := service.client.Get(context.TODO(), nsn, kc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err := service.client.Create(context.TODO(), kc)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
			}
			log.Info(fmt.Sprintf("Keycloak client %s/%s created", kc.Namespace, kc.Name))
			// Successfully created
			return nil
		}
		// Some error occurred
		return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
	}

	// Nothing to do
	return nil
}

func (service K8SService) GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := service.client.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	// Success
	return out, nil
}

func (s K8SService) CreateEDPComponentIfNotExist(ac adminConsoleApi.AdminConsole, url string, icon string) error {
	if _, err := s.getEDPComponent(ac.Name, ac.Namespace); err != nil {
		if k8serrors.IsNotFound(err) {
			return s.createEDPComponent(ac, url, icon)
		}
		return errors.Wrapf(err, "failed to get edp component: %v", ac.Name)
	}
	log.Info("edp component already exists", "name", ac.Name)
	return nil
}

func (s K8SService) getEDPComponent(name, namespace string) (*edpCompApi.EDPComponent, error) {
	c := &edpCompApi.EDPComponent{}
	err := s.client.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s K8SService) createEDPComponent(ac adminConsoleApi.AdminConsole, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "admin-console",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}
	if err := controllerutil.SetControllerReference(&ac, obj, s.Scheme); err != nil {
		return err
	}

	return s.client.Create(context.TODO(), obj)
}
