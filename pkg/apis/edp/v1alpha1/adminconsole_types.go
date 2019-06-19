package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AdminConsoleSpec defines the desired state of AdminConsole
// +k8s:openapi-gen=true
type AdminConsoleSpec struct {
	Image                 string                 `json:"image"`
	Version               string                 `json:"version"`
	KeycloakEnabled       string                 `json:"keycloakEnabled, omitempty"`
	KeycloakUrl           string                 `json:"keycloakUrl, omitempty"`
	EdpSpec               EdpSpec                `json:"edpSpec"`
	DbSettings            AdminConsoleDbSettings `json:"dbSettings, omitempty"`
	ExternalConfiguration ExternalConfiguration  `json:"externalConfiguration, omitempty"`
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

type EdpSpec struct {
	EdpVersion  string `json:"edpVersion"`
	EdpName     string `json:"edpName, omitempty"`
	DnsWildcard string `json:"dnsWildcard"`
}

type ExternalConfigurationItem struct {
	Name        string `json:"name, omitempty"`
	Kind        string `json:"kind, omitempty"`
	Description string `json:"description, omitempty"`
}

type AdminConsoleDbSettings struct {
	DatabaseName     string `json:"databaseName,omitempty"`
	DatabaseHostname string `json:"databaseHostname,omitempty"`
	DatabasePort     string `json:"databasePort,omitempty"`
	DatabaseEnabled  string `json:"databaseEnabled, omitempty"`
}

type ExternalConfiguration struct {
	DbUser       ExternalConfigurationItem `json:"DbUser,omitempty"`
	KeycloakUser ExternalConfigurationItem `json:"keycloackUser,omitempty"`
}

// AdminConsoleStatus defines the observed state of AdminConsole
// +k8s:openapi-gen=true
type AdminConsoleStatus struct {
	Available       bool      `json:"available, omitempty"`
	LastTimeUpdated time.Time `json:"lastTimeUpdated, omitempty"`
	Status          string    `json:"status, omitempty"`
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
