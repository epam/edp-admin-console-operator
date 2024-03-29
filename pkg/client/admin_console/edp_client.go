package admin_console

import (
	"context"

	_ "github.com/lib/pq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
)

//var k8sConfig clientcmd.ClientConfig
var SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}

type EdpV1Client struct {
	crClient *rest.RESTClient
}

func NewForConfig(config *rest.Config) (*EdpV1Client, error) {
	if err := createCrdClient(config); err != nil {
		return nil, err
	}
	crClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	return &EdpV1Client{crClient: crClient}, nil
}

func (c *EdpV1Client) Get(name string, namespace string, options metav1.GetOptions) (result *adminConsoleApi.AdminConsole, err error) {
	result = &adminConsoleApi.AdminConsole{}
	err = c.crClient.Get().
		Namespace(namespace).
		Resource("adminconsoles").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(context.TODO()).
		Into(result)
	return
}

func (c *EdpV1Client) Update(ac *adminConsoleApi.AdminConsole) (result *adminConsoleApi.AdminConsole, err error) {
	result = &adminConsoleApi.AdminConsole{}
	err = c.crClient.Put().
		Namespace(ac.Namespace).
		Resource("adminconsoles").
		Name(ac.Name).
		Body(ac).
		Do(context.TODO()).
		Into(result)
	return
}

func createCrdClient(cfg *rest.Config) error {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return err
	}
	config := cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)

	return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&adminConsoleApi.AdminConsole{},
		&adminConsoleApi.AdminConsoleList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
