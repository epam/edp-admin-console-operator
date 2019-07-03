package service

import (
	"admin-console-operator/pkg/apis/edp/v1alpha1"
	"admin-console-operator/pkg/client"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type K8SService struct {
	scheme     *runtime.Scheme
	coreClient coreV1Client.CoreV1Client
	edpClient  client.EdpV1Client
}

func (service K8SService) CreateSecret(ac v1alpha1.AdminConsole, name string, data map[string][]byte) error {
	labels := generateLabels(ac.Name)

	consoleSecretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&ac, consoleSecretObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleSecret, err := service.coreClient.Secrets(consoleSecretObject.Namespace).Get(consoleSecretObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Creating a new Secret %s/%s for Admin Console", consoleSecretObject.Namespace, consoleSecretObject.Name)

		consoleSecret, err = service.coreClient.Secrets(consoleSecretObject.Namespace).Create(consoleSecretObject)

		if err != nil {
			return logErrorAndReturn(err)
		}
		log.Printf("Secret %s/%s has been created", consoleSecret.Namespace, consoleSecret.Name)

	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service K8SService) CreateService(ac v1alpha1.AdminConsole) error {

	labels := generateLabels(ac.Name)

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

	if err := controllerutil.SetControllerReference(&ac, consoleServiceObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleService, err := service.coreClient.Services(ac.Namespace).Get(ac.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Creating a new service %s/%s for Admin Console %s", consoleServiceObject.Namespace, consoleServiceObject.Name, ac.Name)

		consoleService, err = service.coreClient.Services(consoleServiceObject.Namespace).Create(consoleServiceObject)

		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Service %s/%s has been created", consoleService.Namespace, consoleService.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service K8SService) CreateServiceAccount(ac v1alpha1.AdminConsole) (*coreV1Api.ServiceAccount, error) {

	labels := generateLabels(ac.Name)

	consoleServiceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleServiceAccountObject, service.scheme); err != nil {
		return nil, logErrorAndReturn(err)
	}

	consoleServiceAccount, err := service.coreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Get(consoleServiceAccountObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Creating a new ServiceAccount %s/%s for Admin Console %s", consoleServiceAccountObject.Namespace, consoleServiceAccountObject.Name, ac.Name)

		consoleServiceAccount, err = service.coreClient.ServiceAccounts(consoleServiceAccountObject.Namespace).Create(consoleServiceAccountObject)

		if err != nil {
			return nil, logErrorAndReturn(err)
		}

		log.Printf("ServiceAccount %s/%s has been created", consoleServiceAccount.Namespace, consoleServiceAccount.Name)
	} else if err != nil {
		return nil, logErrorAndReturn(err)
	}

	return consoleServiceAccount, nil
}

func (service K8SService) CreateExternalEndpoint(ac v1alpha1.AdminConsole) error {
	log.Printf("Not implemented.")
	return nil
}

func (service K8SService) GetConfigmap(namespace string, name string) (map[string]string, error) {
	configmap, err := service.coreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Config map %v in namespace %v not found", name, namespace)
		return nil, nil
	} else if err != nil {
		return nil, logErrorAndReturn(err)
	}
	return configmap.Data, nil
}

func (service K8SService) GetSecret(namespace string, name string) (map[string][]byte, error) {
	adminDBSecret, err := service.coreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Printf("Secret %v in namespace %v not found", name, namespace)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return adminDBSecret.Data, nil
}

func (service K8SService) GetAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {

	AdminConsoleInstance, err := service.edpClient.Get(ac.Name, ac.Namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return AdminConsoleInstance, nil
}

func (service K8SService) GetPods(namespace string) (*coreV1Api.PodList, error) {

	PodList, err := service.coreClient.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return &coreV1Api.PodList{}, err
	}

	return PodList, nil
}

func (service K8SService) UpdateAdminConsole(ac v1alpha1.AdminConsole) (*v1alpha1.AdminConsole, error) {
	instance, err := service.edpClient.Update(&ac)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (service *K8SService) Init(config *rest.Config, scheme *runtime.Scheme) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	edpClient, err := client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "EDP Client initialization failed!")
	}

	service.edpClient = *edpClient
	service.coreClient = *coreClient
	service.scheme = scheme
	return nil
}
