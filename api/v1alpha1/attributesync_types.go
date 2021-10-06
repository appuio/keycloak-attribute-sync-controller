package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AttributeSyncSpec defines the desired state of AttributeSync
type AttributeSyncSpec struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the Keycloak server
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the Keycloak server
	// +kubebuilder:validation:Required
	CredentialsSecret SecretRef `json:"credentialsSecret"`

	// LoginRealm is the Keycloak realm to authenticate against
	// +kubebuilder:validation:Optional
	LoginRealm string `json:"loginRealm,omitempty"`

	// Realm is the realm containing the groups to synchronize against
	// +kubebuilder:validation:Required
	Realm string `json:"realm"`

	// URL is the location of the Keycloak server
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Attribute specifies the attribute to sync
	// +kubebuilder:validation:Required
	Attribute string `json:"attribute"`

	// TargetLabel specifies the label to sync the attribute to
	// +kubebuilder:validation:Optional
	TargetLabel string `json:"targetLabel,omitempty"`

	// TargetAnnotation specifies the label to sync the attribute to
	// +kubebuilder:validation:Optional
	TargetAnnotation string `json:"targetAnnotation,omitempty"`

	// Schedule represents a cron based configuration for synchronization
	// +kubebuilder:validation:Optional
	Schedule string `json:"schedule,omitempty"`
}

// AttributeSyncStatus defines the observed state of AttributeSync
type AttributeSyncStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AttributeSync is the Schema for the attributesyncs API
type AttributeSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AttributeSyncSpec   `json:"spec,omitempty"`
	Status AttributeSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AttributeSyncList contains a list of AttributeSync
type AttributeSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AttributeSync `json:"items"`
}

// SecretRef represents a reference to an item within a Secret
// +k8s:openapi-gen=true
type SecretRef struct {
	// Name represents the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace represents the namespace containing the secret
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

func (a *AttributeSync) GetCaSecret() *SecretRef {
	ref := a.Spec.CaSecret
	if ref == nil {
		return nil
	}
	ns := ref.Namespace
	if ns == "" {
		ns = a.ObjectMeta.Namespace
	}
	return &SecretRef{Name: ref.Name, Namespace: ns}
}

func (a *AttributeSync) GetCredentialsSecret() SecretRef {
	ref := a.Spec.CredentialsSecret
	ns := ref.Namespace
	if ns == "" {
		ns = a.ObjectMeta.Namespace
	}
	return SecretRef{Name: ref.Name, Namespace: ns}
}

func (a *AttributeSync) GetLoginRealm() string {
	if a.Spec.LoginRealm == "" {
		return "master"
	}
	return a.Spec.LoginRealm
}

func (a *AttributeSync) GetConditions() []metav1.Condition {
	return a.Status.Conditions
}

func (a *AttributeSync) SetConditions(conditions []metav1.Condition) {
	a.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&AttributeSync{}, &AttributeSyncList{})
}
