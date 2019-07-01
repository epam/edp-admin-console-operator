package service

import (
	"admin-console-operator/pkg/apis/edp/v1alpha1"
	"errors"
	"fmt"
	appsV1Api "github.com/openshift/api/apps/v1"
	authV1Api "github.com/openshift/api/authorization/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	securityV1Api "github.com/openshift/api/security/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityV1Client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templateV1Client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PgPort           = "5432"
	AdminConsolePort = 8080
	MemoryRequest    = "500Mi"
	DbName           = "edp-install-wizard-db"
)

type OpenshiftService struct {
	K8SService

	authClient     authV1Client.AuthorizationV1Client
	templateClient templateV1Client.TemplateV1Client
	projectClient  projectV1Client.ProjectV1Client
	securityClient securityV1Client.SecurityV1Client
	appClient      appsV1client.AppsV1Client
	routeClient    routeV1Client.RouteV1Client
}

func (service OpenshiftService) CreateDeployConf(ac v1alpha1.AdminConsole) error {
	dbEnabled := "false"
	keycloakEnabled := "false"

	host, err := service.getRouteUrl(ac)
	//TODO(Serhii_Shydlovskyi): We can determine the Namespace referenced by the current context in the kubeconfig file.
	// e.g. namespace, _, err := kubeconfig.Namespace()

	displayName,err := service.getDisplayName(ac)
	if err != nil{
		return logErrorAndReturn(err)
	}

	labels := generateLabels(ac.Name)
	consoleDcObject := &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRolling,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					Containers: []coreV1Api.Container{
						{
							Name:            ac.Name,
							Image:           ac.Spec.Image + ":" + ac.Spec.Version,
							ImagePullPolicy: coreV1Api.PullAlways,
							Env: []coreV1Api.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &coreV1Api.EnvVarSource{
										FieldRef: &coreV1Api.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "HOST",
									Value: host,
								},
								{
									Name:  "EDP_ADMIN_CONSOLE_VERSION",
									Value: ac.Spec.Version,
								},
								{
									Name:  "EDP_DEPLOY_PROJECT",
									Value: fmt.Sprintf(displayName + "-deploy-project"),
								},
								{
									Name:  "DB_ENABLED",
									Value: dbEnabled,
								},
								{
									Name:  "EDP_VERSION",
									Value: ac.Spec.EdpSpec.EdpVersion,
								},
								{
									Name:  "AUTH_KEYCLOAK_ENABLED",
									Value: keycloakEnabled,
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									ContainerPort: AdminConsolePort,
								},
							},
							LivenessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 180,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(AdminConsolePort),
									},
								},
							},
							ReadinessProbe: &coreV1Api.Probe{
								FailureThreshold:    5,
								InitialDelaySeconds: 60,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								Handler: coreV1Api.Handler{
									TCPSocket: &coreV1Api.TCPSocketAction{
										Port: intstr.FromInt(AdminConsolePort),
									},
								},
							},
							TerminationMessagePath: "/dev/termination-log",
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(MemoryRequest),
								},
							},
						},
					},
					ServiceAccountName: ac.Name,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(&ac, consoleDcObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleDc, err := service.appClient.DeploymentConfigs(consoleDcObject.Namespace).Get(consoleDcObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new DeploymentConfig %s/%s for Admin Console %s", consoleDcObject.Namespace, consoleDcObject.Name, ac.Name)

		consoleDc, err = service.appClient.DeploymentConfigs(consoleDcObject.Namespace).Create(consoleDcObject)
		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("DeploymentConfig %s/%s has been created", consoleDc.Namespace, consoleDc.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service OpenshiftService) CreateUserRole(ac v1alpha1.AdminConsole) error {
	consoleRoleObject := &authV1Api.Role{
		ObjectMeta: metav1.ObjectMeta{
			// !!!This little typo fix might kill A LOT of things in automation!!!
			Name:      "edp-resources-admin",
			Namespace: ac.Namespace,
		},
		Rules: []authV1Api.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"codebases", "applicationbranches", "codebasebranches", "cdpipelines", "stages"},
				Verbs:     []string{"get", "create"},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleRoleObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleRole, err := service.authClient.Roles(consoleRoleObject.Namespace).Get(consoleRoleObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating Role %s for Admin Console %s", consoleRoleObject.Name, consoleRoleObject.Name)

		consoleRole, err = service.authClient.Roles(consoleRoleObject.Namespace).Create(consoleRoleObject)
		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Role %s for Admin Console %s created", consoleRole.Name, ac.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service OpenshiftService) CreateUserRoleBinding(ac v1alpha1.AdminConsole, name string) error {
	acBindingObject := &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("edp-%s", name),
			Namespace: ac.Namespace,
		},
		RoleRef: coreV1Api.ObjectReference{
			APIVersion: "rbac.authorization.k8s.io",
			Kind:       "Role",
			Name:       name,
			Namespace:  ac.Namespace,
		},
		Subjects: []coreV1Api.ObjectReference{
			{
				Kind: "ServiceAccount",
				Name: "admin-console",
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, acBindingObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	acBinding, err := service.authClient.RoleBindings(acBindingObject.Namespace).Get(acBindingObject.Name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new RoleBinding %s/%s for Admin Console %s", acBindingObject.Namespace, acBindingObject.Name, ac.Name)
		acBinding, err = service.authClient.RoleBindings(acBindingObject.Namespace).Create(acBindingObject)
		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("RoleBinding %s/%s has been created", acBinding.Namespace, acBinding.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service OpenshiftService) CreateExternalEndpoint(ac v1alpha1.AdminConsole) error {

	labels := generateLabels(ac.Name)

	consoleRouteObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: ac.Name,
				Kind: "Service",
			},
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleRouteObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleRoute, err := service.routeClient.Routes(consoleRouteObject.Namespace).Get(consoleRouteObject.Name, metav1.GetOptions{})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new Route %s/%s for Admin Console %s", consoleRouteObject.Namespace, consoleRouteObject.Name, ac.Name)
		consoleRoute, err = service.routeClient.Routes(consoleRouteObject.Namespace).Create(consoleRouteObject)

		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Route %s/%s has been created", consoleRoute.Namespace, consoleRoute.Name)
	} else if err != nil {
		return logErrorAndReturn(err)
	}

	return nil
}

func (service OpenshiftService) CreateSecurityContext(ac v1alpha1.AdminConsole, sa *coreV1Api.ServiceAccount) error {

	labels := generateLabels(ac.Name)
	priority := int32(1)


	displayName,err := service.getDisplayName(ac)
	if err != nil{
		return logErrorAndReturn(err)
	}

	consoleSCCObject := &securityV1Api.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", ac.Name, displayName),
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Volumes: []securityV1Api.FSType{
			securityV1Api.FSTypeSecret,
			securityV1Api.FSTypeDownwardAPI,
			securityV1Api.FSTypeEmptyDir,
			securityV1Api.FSTypePersistentVolumeClaim,
			securityV1Api.FSProjected,
			securityV1Api.FSTypeConfigMap,
		},
		AllowHostDirVolumePlugin: false,
		AllowHostIPC:             true,
		AllowHostNetwork:         false,
		AllowHostPID:             false,
		AllowHostPorts:           false,
		AllowPrivilegedContainer: false,
		AllowedCapabilities:      []coreV1Api.Capability{},
		AllowedFlexVolumes:       []securityV1Api.AllowedFlexVolume{},
		DefaultAddCapabilities:   []coreV1Api.Capability{},
		FSGroup: securityV1Api.FSGroupStrategyOptions{
			Type:   securityV1Api.FSGroupStrategyRunAsAny,
			Ranges: []securityV1Api.IDRange{},
		},
		Groups:                 []string{},
		Priority:               &priority,
		ReadOnlyRootFilesystem: false,
		RunAsUser: securityV1Api.RunAsUserStrategyOptions{
			Type:        securityV1Api.RunAsUserStrategyRunAsAny,
			UID:         nil,
			UIDRangeMin: nil,
			UIDRangeMax: nil,
		},
		SELinuxContext: securityV1Api.SELinuxContextStrategyOptions{
			Type:           securityV1Api.SELinuxStrategyMustRunAs,
			SELinuxOptions: nil,
		},
		SupplementalGroups: securityV1Api.SupplementalGroupsStrategyOptions{
			Type:   securityV1Api.SupplementalGroupsStrategyRunAsAny,
			Ranges: nil,
		},
		Users: []string{
			"system:serviceaccount:" + ac.Namespace + ":admin-console",
		},
	}

	if err := controllerutil.SetControllerReference(&ac, consoleSCCObject, service.scheme); err != nil {
		return logErrorAndReturn(err)
	}

	consoleSCC, err := service.securityClient.SecurityContextConstraints().Get(consoleSCCObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.Printf("Creating a new Security Context Constraint %s for Admin Console %s", consoleSCCObject.Name, ac.Name)

		consoleSCC, err = service.securityClient.SecurityContextConstraints().Create(consoleSCCObject)

		if err != nil {
			return logErrorAndReturn(err)
		}

		log.Printf("Security Context Constraint %s has been created", consoleSCC.Name)
	} else if err != nil {
		return logErrorAndReturn(err)

	} else {
		// TODO(Serhii Shydlovskyi): Reflect reports that present users and currently stored in object are different for some reason.
		if !reflect.DeepEqual(consoleSCC.Users, consoleSCCObject.Users) {

			consoleSCC, err = service.securityClient.SecurityContextConstraints().Update(consoleSCCObject)

			if err != nil {
				return logErrorAndReturn(err)
			}

			log.Printf("Security Context Constraint %s has been updated", consoleSCC.Name)
		}
	}

	return nil
}

func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme) error {

	err := service.K8SService.Init(config, scheme)
	if err != nil {
		return logErrorAndReturn(err)
	}

	templateClient, err := templateV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.templateClient = *templateClient
	projectClient, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.projectClient = *projectClient
	securityClient, err := securityV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.securityClient = *securityClient
	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}

	service.appClient = *appClient
	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}
	service.routeClient = *routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return logErrorAndReturn(err)
	}
	service.authClient = *authClient

	return nil
}

func (service OpenshiftService) getDisplayName(ac v1alpha1.AdminConsole) (string, error) {
	project, err := service.projectClient.Projects().Get(ac.Namespace, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return "", errors.New(fmt.Sprintf("Unable to retrieve project %s", ac.Namespace))
	}

	displayName := project.GetObjectMeta().GetAnnotations()["openshift.io/display-name"]
	if displayName == "" {
		return "", errors.New(fmt.Sprintf("Project display name does not set"))
	}
	return displayName, nil
}

func (service OpenshiftService) getRouteUrl(ac v1alpha1.AdminConsole) (string, error) {

	route, err := service.routeClient.Routes(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		return  "", err
	}

	Url := route.Spec.Host
	return Url, nil
}
