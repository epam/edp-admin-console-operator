package v1alpha1

import (
	"time"

	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AdminConsoleSpec defines the desired state of AdminConsole
// +k8s:openapi-gen=true
type AdminConsoleSpec struct {
	Image            string                           `json:"image"`
	Version          string                           `json:"version"`
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	KeycloakSpec     KeycloakSpec                     `json:"keycloakSpec,omitempty"`
	EdpSpec          EdpSpec                          `json:"edpSpec"`
	DbSpec           AdminConsoleDbSettings           `json:"dbSpec,omitempty"`
	BasePath         string                           `json:"basePath, omitempty"`
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type EdpSpec struct {
	Version               string `json:"version"`
	Name                  string `json:"name, omitempty"`
	DnsWildcard           string `json:"dnsWildcard"`
	IntegrationStrategies string `json:"integrationStrategies,omitempty"`
}

type KeycloakSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type AdminConsoleDbSettings struct {
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     string `json:"port,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
}

// AdminConsoleStatus defines the observed state of AdminConsole
// +k8s:openapi-gen=true
type AdminConsoleStatus struct {
	Available       bool      `json:"available,omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated,omitempty"`
	Status          string    `json:"status,omitempty"`
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AdminConsole is the Schema for the adminconsoles API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type AdminConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdminConsoleSpec   `json:"spec,omitempty"`
	Status AdminConsoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AdminConsoleList contains a list of AdminConsole
type AdminConsoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdminConsole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdminConsole{}, &AdminConsoleList{})
}
