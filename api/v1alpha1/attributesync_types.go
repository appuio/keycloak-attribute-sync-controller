/*
Copyright 2021 APPUiO.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	CredentialsSecret *SecretRef `json:"credentialsSecret"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Keycloak
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

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
	TargetLabel string `json:"target_label,omitempty"`

	// TargetAnnotation specifies the label to sync the attribute to
	// +kubebuilder:validation:Optional
	TargetAnnotation string `json:"target_annotation,omitempty"`

	// Schedule represents a cron based configuration for synchronization
	// +kubebuilder:validation:Optional
	Schedule string `json:"schedule,omitempty"`
}

// AttributeSyncStatus defines the observed state of AttributeSync
type AttributeSyncStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LastSyncSuccessTime represents the time last synchronization completed successfully
	// +kubebuilder:validation:Optional
	LastSyncSuccessTime *metav1.Time `json:"lastSyncSuccessTime,omitempty"`
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
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Key represents the specific key to reference from the secret
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
}

func init() {
	SchemeBuilder.Register(&AttributeSync{}, &AttributeSyncList{})
}
