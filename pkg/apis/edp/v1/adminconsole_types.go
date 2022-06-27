package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AdminConsoleSpec defines the desired state of AdminConsole
type AdminConsoleSpec struct {
	// +optional
	KeycloakSpec KeycloakSpec `json:"keycloakSpec,omitempty"`
	EdpSpec      EdpSpec      `json:"edpSpec"`
	// +optional
	DbSpec AdminConsoleDbSettings `json:"dbSpec,omitempty"`
	// +optional
	BasePath string `json:"basePath,omitempty"`
}

type EdpSpec struct {
	// +optional
	Name            string `json:"name,omitempty"`
	DnsWildcard     string `json:"dnsWildcard"`
	TestReportTools string `json:"testReportTools"`
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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// AdminConsole is the Schema for the adminconsoles API
type AdminConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdminConsoleSpec   `json:"spec,omitempty"`
	Status AdminConsoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AdminConsoleList contains a list of AdminConsole
type AdminConsoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AdminConsole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdminConsole{}, &AdminConsoleList{})
}
