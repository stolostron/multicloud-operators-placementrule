/*
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope="Cluster"

// MultiClusterImport is the Schema for the multiclusterimports API.
type MultiClusterImport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterImportSpec   `json:"spec,omitempty"`
	Status MultiClusterImportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterImportSpec defines the desired state of MultiClusterImport.
type MultiClusterImportSpec struct {
	ArgoServer   ArgoServerSpec          `json:"ArgoServer,omitempty"`
	PlacementRef *corev1.ObjectReference `json:"placementRef,omitempty"`
}

// ArgoServerSpec defines a argo server installed in a managed cluster.
type ArgoServerSpec struct {
	Cluster       string `json:"cluster,omitempty"`
	ArgoNamespace string `json:"argoNamespace,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterImportStatus defines the observed state of MultiClusterImport.
type MultiClusterImportStatus struct {
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	Message        string      `json:"message,omitempty"`
	Phase          string      `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true

// MultiClusterImportList contains a list of MultiClusterImport.
type MultiClusterImportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterImport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiClusterImport{}, &MultiClusterImportList{})
}
