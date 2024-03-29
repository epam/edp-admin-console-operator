package v1alpha1

import (
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AdminConsoleSpec defines the desired state of AdminConsole
type AdminConsoleSpec struct {
	Image   string `json:"image"`
	Version string `json:"version"`
	// +nullable
	// +optional
	ImagePullSecrets []coreV1Api.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	KeycloakSpec KeycloakSpec `json:"keycloakSpec,omitempty"`
	EdpSpec      EdpSpec      `json:"edpSpec"`
	// +optional
	DbSpec AdminConsoleDbSettings `json:"dbSpec,omitempty"`
	// +optional
	BasePath string `json:"basePath,omitempty"`
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type EdpSpec struct {
	Version string `json:"version"`
	// +optional
	Name                  string `json:"name,omitempty"`
	DnsWildcard           string `json:"dnsWildcard"`
	IntegrationStrategies string `json:"integrationStrategies"`
	TestReportTools       string `json:"testReportTools"`
}

type KeycloakSpec struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

type AdminConsoleDbSettings struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// +optional
	Port string `json:"port,omitempty"`
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// AdminConsoleStatus defines the observed state of AdminConsole
type AdminConsoleStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`
	// +optional
	LastTimeUpdated metav1.Time `json:"lastTimeUpdated,omitempty"`
	// +optional
	Status string `json:"status,omitempty"`
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// AdminConsole is the Schema for the adminconsoles API
type AdminConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdminConsoleSpec   `json:"spec,omitempty"`
	Status AdminConsoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdminConsoleList contains a list of AdminConsole
type AdminConsoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AdminConsole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdminConsole{}, &AdminConsoleList{})
}
