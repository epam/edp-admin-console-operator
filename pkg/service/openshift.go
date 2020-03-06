package service

import (
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/admin-console-operator/v2/pkg/apis/edp/v1alpha1"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"

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
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AdminConsolePort        = 8080
	MemoryRequest           = "500Mi"
	ClusterRole      string = "clusterrole"
	Role             string = "role"
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

func (service OpenshiftService) AddServiceAccToSecurityContext(scc string, ac v1alpha1.AdminConsole) error {
	saName := fmt.Sprintf("system:serviceaccount:%s:%s", ac.Namespace, ac.Name)
	consoleSCC, err := service.securityClient.SecurityContextConstraints().Get(scc, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !stringInSlice(saName, consoleSCC.Users) {
		log.Printf("Adding Service Account to anyuid Security Context")
		consoleSCC.Users = append(consoleSCC.Users, saName)
		_, err = service.securityClient.SecurityContextConstraints().Update(consoleSCC)
		if err != nil {
			return err
		}
		log.Printf("anyuid Security Context updated succesfully.")
	}

	return nil
}

func (service OpenshiftService) CreateDeployConf(ac v1alpha1.AdminConsole) error {
	openshiftClusterURL, err := service.getClusterURL()
	if err != nil {
		return errors.Wrap(err, "Unable to build an OpenshiftClusterURL value")
	}

	dbEnabled := "false"
	keycloakEnabled := "false"

	host, err := service.getRouteUrl(ac)
	secureHost := fmt.Sprintf("https://%s", host)

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
					ImagePullSecrets: ac.Spec.ImagePullSecrets,
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
									Value: secureHost,
								},
								{
									Name:  "EDP_ADMIN_CONSOLE_VERSION",
									Value: ac.Spec.Version,
								},
								{
									Name:  "DB_ENABLED",
									Value: dbEnabled,
								},
								{
									Name:  "EDP_VERSION",
									Value: ac.Spec.EdpSpec.Version,
								},
								{
									Name:  "AUTH_KEYCLOAK_ENABLED",
									Value: keycloakEnabled,
								},
								{
									Name:  "DNS_WILDCARD",
									Value: ac.Spec.EdpSpec.DnsWildcard,
								},
								{
									Name:  "OPENSHIFT_CLUSTER_URL",
									Value: openshiftClusterURL,
								},
								{
									Name: "PG_USER",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "admin-console-db",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "PG_PASSWORD",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: "admin-console-db",
											},
											Key: "password",
										},
									},
								},
								{
									Name:  "INTEGRATION_STRATEGIES",
									Value: ac.Spec.EdpSpec.IntegrationStrategies,
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

	displayName, err := service.GetDisplayName(ac)
	if err != nil {
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
			fmt.Sprintf("system:serviceaccount:%s:%s", ac.Namespace, ac.Name),
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
				Verbs:     []string{"get", "create", "update"},
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

func (service OpenshiftService) CreateUserRoleBinding(ac v1alpha1.AdminConsole, name string, binding string, kind string) error {

	acBindingObject, err := getNewRoleObject(ac, name, binding, kind)
	if err != nil {
		return err
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

func (service OpenshiftService) GetDisplayName(ac v1alpha1.AdminConsole) (string, error) {
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

func (service OpenshiftService) GetDeployConf(ac v1alpha1.AdminConsole) (*appsV1Api.DeploymentConfig, error) {

	result, err := service.appClient.DeploymentConfigs(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		return &appsV1Api.DeploymentConfig{}, err
	}

	return result, nil
}

func (service OpenshiftService) GenerateDbSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {
	var out []coreV1Api.EnvVar

	if ac.Spec.DbSpec.Enabled {
		log.Printf("Generating DB settings for Admin Console %s", ac.Name)
		if containsEmpty(ac.Spec.DbSpec.Name, ac.Spec.DbSpec.Hostname, ac.Spec.DbSpec.Port) {
			return nil, errors.New("One or many DB settings field are empty!")
		}

		out = []coreV1Api.EnvVar{
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
		}

		return out, nil
	}

	log.Printf("DB_ENABLED flag in %s spec is false. Settings will not be created.", ac.Name)
	return out, nil
}

func (service OpenshiftService) GenerateKeycloakSettings(ac v1alpha1.AdminConsole) ([]coreV1Api.EnvVar, error) {
	var out []coreV1Api.EnvVar

	if ac.Spec.KeycloakSpec.Enabled {
		log.Printf("Generating Keycloak settings for Admin Console %s", ac.Name)
		if ac.Spec.KeycloakSpec.Url == "" {
			return nil, errors.New(fmt.Sprintf("Keycloak URL field in %s is empty and integration enabled!", ac.Name))
		}

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
				Value: ac.Spec.KeycloakSpec.Url,
			},
			{
				Name:  "AUTH_KEYCLOAK_ENABLED",
				Value: strconv.FormatBool(ac.Spec.KeycloakSpec.Enabled),
			},
		}
		return out, nil
	}

	log.Printf("KEYCLOAK_ENABLED flag in %s spec is false. Settings will not be created.", ac.Name)
	return out, nil
}

func (service OpenshiftService) PatchDeployConfEnv(ac v1alpha1.AdminConsole, dc *appsV1Api.DeploymentConfig, env []coreV1Api.EnvVar) error {

	if len(env) == 0 {
		return nil
	}

	container, err := selectContainer(dc.Spec.Template.Spec.Containers, ac.Name)
	if err != nil {
		return err
	}

	container.Env = updateEnv(container.Env, env)

	dc.Spec.Template.Spec.Containers = append(dc.Spec.Template.Spec.Containers, container)

	jsonDc, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	_, err = service.appClient.DeploymentConfigs(dc.Namespace).Patch(dc.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
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

func (service OpenshiftService) getRouteUrl(ac v1alpha1.AdminConsole) (string, error) {

	route, err := service.routeClient.Routes(ac.Namespace).Get(ac.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	Url := route.Spec.Host
	return Url, nil
}

// getClusterURL extracts Openshift's Cluster URL from openshift-web-console/webconsole-config ConfigMap object
func (service OpenshiftService) getClusterURL() (string, error) {
	namespace := "openshift-web-console"
	name := "webconsole-config"
	filename := "webconsole-config.yaml"
	cm, err := service.coreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "Unable to get a ConfigMap - %s/%s", namespace, name)
	}
	data, ok := cm.Data[filename]
	if !ok {
		return "", errors.Wrapf(err, "ConfigMap %s/%s has no required data, % is missing", namespace, name, filename)
	}
	config, err := parseWebConsoleConfig(data)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse WebConsoleConfiguration")
	}
	clusterURL, err := stripClusterURL(config.ClusterInfo.ConsolePublicURL)
	if err != nil {
		return "", errors.Wrap(err, "Unable to strip a relative path from a URL")
	}
	// Success
	return clusterURL, nil
}

// webConsoleConfiguration defines required properties of a data structure used by YAML-formatted payload
// of the openshift-web-console/webconsole-config ConfigMap object
type webConsoleConfiguration struct {
	ClusterInfo struct {
		ConsolePublicURL string `yaml:"consolePublicURL"`
	} `yaml:"clusterInfo"`
}

// parseWebConsoleConfig unmarshals YAML-formatted data into webConsoleConfiguration object
func parseWebConsoleConfig(data string) (*webConsoleConfiguration, error) {
	config := &webConsoleConfiguration{}
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal webConsoleConfiguration data")
	}
	// Success
	return config, nil
}

// stripClusterURL returns ClusterURL as url parameter value without relative path
func stripClusterURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse a URL string")
	}
	// Success
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func selectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	out := coreV1Api.Container{}
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}

	return out, errors.New("No matching container in spec found!")
}

func updateEnv(existing []coreV1Api.EnvVar, env []coreV1Api.EnvVar) []coreV1Api.EnvVar {
	var out []coreV1Api.EnvVar
	var covered []string

	for _, e := range existing {
		newer, ok := findEnv(env, e.Name)
		if ok {
			covered = append(covered, e.Name)
			out = append(out, newer)
			continue
		}
		out = append(out, e)
	}
	for _, e := range env {
		if stringInSlice(e.Name, covered) {
			continue
		}
		covered = append(covered, e.Name)
		out = append(out, e)
	}
	return out
}

func findEnv(env []coreV1Api.EnvVar, name string) (coreV1Api.EnvVar, bool) {
	for _, e := range env {
		if e.Name == name {
			return e, true
		}
	}
	return coreV1Api.EnvVar{}, false
}

func getNewRoleObject(ac v1alpha1.AdminConsole, name string, binding string, kind string) (*authV1Api.RoleBinding, error) {
	switch strings.ToLower(kind) {
	case ClusterRole:
		return newCluseterRoleObject(ac, name, binding), nil
	case Role:
		return newRoleObject(ac, name, binding), nil
	default:
		return &authV1Api.RoleBinding{}, errors.New(fmt.Sprintf("Wrong role kind %s! Cant't create rolebinding", kind))

	}

}

func newCluseterRoleObject(ac v1alpha1.AdminConsole, name string, binding string) *authV1Api.RoleBinding {
	return &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
		},
		RoleRef: coreV1Api.ObjectReference{
			APIVersion: "rbac.authorization.k8s.io",
			Kind:       "ClusterRole",
			Name:       binding,
		},
		Subjects: []coreV1Api.ObjectReference{
			{
				Kind: "ServiceAccount",
				Name: ac.Name,
			},
		},
	}
}

func newRoleObject(ac v1alpha1.AdminConsole, name string, binding string) *authV1Api.RoleBinding {

	return &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ac.Namespace,
		},
		RoleRef: coreV1Api.ObjectReference{
			APIVersion: "rbac.authorization.k8s.io",
			Kind:       "Role",
			Name:       binding,
			Namespace:  ac.Namespace,
		},
		Subjects: []coreV1Api.ObjectReference{
			{
				Kind: "ServiceAccount",
				Name: ac.Name,
			},
		},
	}

}

func containsEmpty(ss ...string) bool {
	for _, s := range ss {
		if s == "" {
			return true
		}
	}
	return false
}
