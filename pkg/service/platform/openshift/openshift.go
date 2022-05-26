package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-admin-console-operator/v2/pkg/helper"
	platformHelper "github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-admin-console-operator/v2/pkg/service/platform/kubernetes"
)

var log = ctrl.Log.WithName("platform")

type OpenshiftService struct {
	kubernetes.K8SService

	authClient     authV1Client.AuthorizationV1Client
	templateClient templateV1Client.TemplateV1Client
	projectClient  projectV1Client.ProjectV1Client
	securityClient securityV1Client.SecurityV1Client
	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
	client         client.Client
}

const (
	deploymentTypeEnvName           = "DEPLOYMENT_TYPE"
	deploymentConfigsDeploymentType = "deploymentConfigs"
)

func (service OpenshiftService) GenerateDbSettings(ac adminConsoleApi.AdminConsole) ([]coreV1Api.EnvVar, error) {
	if !ac.Spec.DbSpec.Enabled {
		msg := fmt.Sprintf("DB_ENABLED flag in %s spec is false.", ac.Name)
		log.V(1).Info(msg)

		return []coreV1Api.EnvVar{
			{
				Name:  "DB_ENABLED",
				Value: "false",
			},
		}, nil
	}

	log.V(1).Info(fmt.Sprintf("Generating DB settings for Admin Console %s", ac.Name))
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

func (service OpenshiftService) GenerateKeycloakSettings(ac adminConsoleApi.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error) {
	var out []coreV1Api.EnvVar

	log.V(1).Info(fmt.Sprintf("Generating Keycloak settings for Admin Console %s", ac.Name))

	out = []coreV1Api.EnvVar{
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
	}

	return out, nil
}

func (service OpenshiftService) PatchDeploymentEnv(ac adminConsoleApi.AdminConsole, env []coreV1Api.EnvVar) error {
	if len(env) == 0 {
		return nil
	}

	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		dc, err := helper.GetDeploymentConfig(service.appClient, ac.Name, ac.Namespace)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				log.Info(fmt.Sprintf("Deployment %s not found!", ac.Name))
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

		_, err = service.appClient.DeploymentConfigs(dc.Namespace).Patch(context.TODO(), dc.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
		if err != nil {
			return err
		}

		return nil
	}

	return service.K8SService.PatchDeploymentEnv(ac, env)
}

func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {

	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return err
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.templateClient = *templateClient
	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.projectClient = *projectClient
	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.securityClient = *securityClient
	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return err
	}

	service.appClient = *appClient
	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.routeClient = *routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return err
	}
	service.authClient = *authClient
	service.client = *k8sClient

	return nil
}

// GetExternalUrl returns Route object from Openshift
func (service OpenshiftService) GetExternalUrl(namespace string, name string) (*string, error) {
	route, err := service.routeClient.Routes(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Route %v in namespace %v not found", name, namespace))
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var routeScheme = "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}

	u := fmt.Sprintf("%s://%s%s", routeScheme, route.Spec.Host, strings.TrimRight(route.Spec.Path, platformHelper.UrlCutset))
	return &u, nil
}

// IsDeploymentReady gets Deployment Config from Openshift, based on data from Admin Console
func (service OpenshiftService) IsDeploymentReady(instance adminConsoleApi.AdminConsole) (bool, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		return helper.IsDeploymentConfigReady(service.appClient, instance.Name, instance.Namespace)
	}
	return service.K8SService.IsDeploymentReady(instance)
}
