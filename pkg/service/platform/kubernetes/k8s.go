package kubernetes

import (
	"context"
	"fmt"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/client/admin_console"
	platformHelper "github.com/epmd-edp/admin-console-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

type K8SService struct {
	Scheme                *runtime.Scheme
	CoreClient            coreV1Client.CoreV1Client
	EdpClient             admin_console.EdpV1Client
	k8sUnstructuredClient client.Client
	AppsClient            appsV1Client.AppsV1Client
}

func (service K8SService) AddServiceAccToSecurityContext(scc string, ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateDeployConf(ac v1alpha1.AdminConsole, url string) error {
	return nil
}

func (service K8SService) CreateSecurityContext(ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateUserRole(ac v1alpha1.AdminConsole) error {
	return nil
}

func (service K8SService) CreateUserRoleBinding(ac v1alpha1.AdminConsole, name string, binding string, kind string) error {
	return nil
}

func (service K8SService) GetDisplayName(ac v1alpha1.AdminConsole) (string, error) {
	return "", nil
}

func (service K8SService) GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {
	return []coreV1Api.EnvVar{}, nil
}

func (service K8SService) GenerateKeycloakSettings(ac v1alpha1.AdminConsole, keycloakUrl string) ([]coreV1Api.EnvVar, error) {
	return []coreV1Api.EnvVar{}, nil
}

func (service K8SService) PatchDeploymentEnv(ac v1alpha1.AdminConsole, env []coreV1Api.EnvVar) error {
	return nil
}

func (service K8SService) GetExternalUrl(namespace string, name string) (string, string, error) {
	return "", "", nil
}

func (service K8SService) IsDeploymentReady(instance v1alpha1.AdminConsole) (bool, error) {
	deploymentConfig, err := service.AppsClient.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (service K8SService) CreateSecret(ac v1alpha1.AdminConsole, name string, data map[string][]byte) error {
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

	consoleSecret, err := service.CoreClient.Secrets(consoleSecretObject.Namespace).Get(consoleSecretObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new Secret %s/%s for Admin Console", consoleSecretObject.Namespace, consoleSecretObject.Name)
			log.V(1).Info(msg)
			consoleSecret, err = service.CoreClient.Secrets(consoleSecretObject.Namespace).Create(consoleSecretObject)
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

func (service K8SService) CreateService(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)

	consoleServiceObject := &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports: []coreV1Api.ServicePort{
				{
					TargetPort: intstr.IntOrString{StrVal: ac.Name},
					Port:       8080,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleServiceObject, service.Scheme); err != nil {
		return err
	}

	consoleService, err := service.CoreClient.Services(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating a new service %s/%s for Admin Console %s",
				consoleServiceObject.Namespace, consoleServiceObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleService, err = service.CoreClient.Services(consoleServiceObject.Namespace).Create(consoleServiceObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Service %s/%s has been created", consoleService.Namespace, consoleService.Name))
			// Created successfully
			return nil
		}
		// Some error occurred
		return err
	}

	// Nothing to do
	return nil
}

func (service K8SService) CreateServiceAccount(ac v1alpha1.AdminConsole) error {

	labels := platformHelper.GenerateLabels(ac.Name)

	consoleServiceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleServiceAccountObject, service.Scheme); err != nil {
		return err
	}

	consoleServiceAccount, err := service.CoreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Get(consoleServiceAccountObject.Name, metav1.GetOptions{})

	if err != nil {
		if k8serr.IsNotFound(err) {
			msg := fmt.Sprintf("Creating ServiceAccount %s/%s for Admin Console %s", consoleServiceAccountObject.Namespace, consoleServiceAccountObject.Name, ac.Name)
			log.V(1).Info(msg)
			consoleServiceAccount, err = service.CoreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Create(consoleServiceAccountObject)
			if err != nil {
				return err
			}
			msg = fmt.Sprintf("ServiceAccount %s/%s has been created", consoleServiceAccount.Namespace, consoleServiceAccount.Name)
			log.Info(msg)
			// Successfully created
			return nil
		}
		// Some error occurred
		return err
	}

	// Nothing to do
	return nil
}

func (service K8SService) CreateExternalEndpoint(ac v1alpha1.AdminConsole) error {
	log.Info("Not implemented.")
	// Nothing to do
	return nil
}

func (service K8SService) GetConfigmap(namespace string, name string) (map[string]string, error) {
	out := map[string]string{}
	configmap, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info(fmt.Sprintf("Config map %v in namespace %v not found", name, namespace))
			return out, nil
		}
		// Some error occurred
		return out, err
	}
	out = configmap.Data
	// Success
	return out, nil
}

func (service K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	out := map[string][]byte{}
	adminDBSecret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
			return nil, nil
		}
		return out, err
	}
	out = adminDBSecret.Data
	return out, nil
}

func (service K8SService) GetAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	AdminConsoleInstance, err := service.EdpClient.Get(ac.Name, ac.Namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return AdminConsoleInstance, nil
}

func (service K8SService) GetPods(namespace string) (*coreV1Api.PodList, error) {

	PodList, err := service.CoreClient.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return &coreV1Api.PodList{}, err
	}

	return PodList, nil
}

func (service K8SService) UpdateAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {
	instance, err := service.EdpClient.Update(&ac)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Core Client initialization failed!")
	}

	edpClient, err := admin_console.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return errors.New("appsV1 client initialization failed!")
	}

	service.EdpClient = *edpClient
	service.CoreClient = *coreClient
	service.Scheme = scheme
	service.k8sUnstructuredClient = *k8sClient
	service.AppsClient = *appsClient
	return nil
}

func (service K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, kc)
	if err != nil {
		if k8serr.IsNotFound(err) {
			err := service.k8sUnstructuredClient.Create(context.TODO(), kc)
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

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	// Success
	return out, nil
}
