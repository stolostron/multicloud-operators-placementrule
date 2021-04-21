package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Placement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the attributes of Placement.
	// +kubebuilder:validation:Required
	// +required
	Spec PlacementSpec `json:"spec"`

	// Status represents the current status of the Placement
	// +optional
	Status PlacementStatus `json:"status,omitempty"`
}

// PlacementSpec defines the attributes of Placement.
// An empty PlacementSpec selects all ManagedClusters from the ManagedClusterSets bound to
// the placement namespace. The containing fields are ANDed.
type PlacementSpec struct {
	// ClusterSets represent the ManagedClusterSets from which the ManagedClusters are selected.
	// If the slice is empty, ManagedClusters will be selected from the ManagedClusterSets bound to the placement
	// namespace, otherwise ManagedClusters will be selected from the intersection of this slice and the
	// ManagedClusterSets bound to the placement namespace.
	// +optional
	ClusterSets []string `json:"clusterSets,omitempty"`

	// NumberOfClusters represents the number of ManagedClusters to be selected which meet the
	// requirements.
	// 1) If not specified, it indicates all matched ManagedClusters will be selected;
	// 2) Otherwise if the nubmer of matched ManagedClusters is larger than NumberOfClusters, a random subset
	//    with expected number of ManagedClusters will be selected;
	// 3) If the nubmer of matched ManagedClusters is equal to NumberOfClusters, all matched ManagedClusters
	//    will be selected;
	// 4) If the nubmer of matched ManagedClusters is less than NumberOfClusters, all matched ManagedClusters
	//    will be selected, and the status of condition `PlacementConditionSatisfied` will be set to false;
	// +optional
	NumberOfClusters *int32 `json:"numberOfClusters,omitempty"`

	// Predicates represent a slice of predicates to select ManagedClusters. The predicates are ORed.
	// +optional
	Predicates []ClusterPredicate `json:"predicates,omitempty"`

	// Affinity represents the scheduling constraints. It will be applied to the final selection of the
	// ManagedClusters.
	// +optional
	Affinity Affinity `json:"affinity,omitempty"`
}

// ClusterPredicate represents a predicate to select ManagedClusters.
type ClusterPredicate struct {
	// RequiredClusterSelector represents a selector of ManagedClusters by label and claim. If specified,
	// 1) Any ManagedCluster, which does not match the selector, should not be selected by this ClusterPredicate;
	// 2) If a selected ManagedCluster (of this ClusterPredicate) ceases to match the selector (e.g. due to
	//    an update), it will be eventually removed from the placement decisions;
	// 3) If a ManagedCluster (not match the selector previously) starts to match the selector, it will either
	//    be selected or at least has chance to be selected (when NumberOfClusters is specified);
	// +optional
	RequiredClusterSelector ClusterSelector `json:"requiredClusterSelector,omitempty"`
}

// ClusterSelector represents the AND of the containing selectors. An empty cluster selector matches all objects.
// A null cluster selector matches no objects.
type ClusterSelector struct {
	// LabelSelector represents a selector of ManagedClusters by label
	// +optional
	LabelSelector metav1.LabelSelector `json:"labelSelector,omitempty"`

	// ClaimSelector represents a selector of ManagedClusters by clusterClaims in status
	// +optional
	ClaimSelector ClusterClaimSelector `json:"claimSelector,omitempty"`
}

// ClusterClaimSelector is a claim query over a set of ManagedClusters. An empty cluster claim
// selector matches all objects. A null cluster claim selector matches no objects.
type ClusterClaimSelector struct {
	// matchExpressions is a list of cluster claim selector requirements. The requirements are ANDed.
	// +optional
	MatchExpressions []metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// Affinity is a group of affinity scheduling rules.
type Affinity struct {
	// ClusterAntiAffinity describes cluster affinity scheduling rules for the placement.
	// +optional
	ClusterAntiAffinity ClusterAntiAffinity `json:"clusterAntiAffinity,omitempty"`
}

// ClusterAntiAffinity is a group of cluster affinity scheduling rules. An empty cluster anti affinity
// selects all objects. A null cluster anti affinity selects no objects.
type ClusterAntiAffinity struct {
	// RequiredClusterAffinityTerms contains a slice of ClusterAffinityTerms. If specified,
	// 1) Any ManagedCluster, which does not meet the requirements, should not be selected;
	// 2) If a selected ManagedCluster ceases to meet the anti-affinity requirements (e.g.
	//    due to an update), it or the ones conflict with this ManagedCluster will be eventually
	//    removed from the placement decisions;
	// 3) If a ManagedCluster (not meet the requirements previously) starts to meet the
	//    requirements, it will either be selected or at least has chance to be selected;
	// When there are multiple elements, the lists of ManagedClusters corresponding to each
	// ClusterAffinityTerm are intersected, i.e. all terms must be satisfied.
	//
	// +optional
	RequiredClusterAffinityTerms []ClusterAffinityTerm `json:"requiredClusterAffinityTerms,omitempty"`
}

// ClusterAffinityTerm defines a constraint on a set of ManagedClusters. Any ManagedCluster in this set
// should be co-located (affinity) or not co-located (anti-affinity) with each other, where co-located
// is defined as the value of the label/claim with name <topologyKey> on ManagedCluster is the same.
type ClusterAffinityTerm struct {
	// TopologyKey is either a label key or a cluster claim name of ManagedClusters
	// +kubebuilder:validation:Required
	// +required
	TopologyKey string `json:"topologyKey"`

	// TopologyKeyType indicates the type of TopologyKey. It could be Label or Claim.
	// +kubebuilder:validation:Required
	// +required
	TopologyKeyType string `json:"topologyKeyType"`
}

const (
	TopologyKeyTypeLabel string = "Label"
	TopologyKeyTypeClaim string = "Claim"
)

type PlacementStatus struct {
	// NumberOfSelectedClusters represents the number of selected ManagedClusters
	// +optional
	NumberOfSelectedClusters int32 `json:"numberOfSelectedClusters"`

	// Conditions contains the different condition statuses for this Placement.
	// +optional
	Conditions []metav1.Condition `json:"conditions"`
}

const (
	// PlacementConditionSatisfied means Placement is satisfied.
	// A placement is not satisfied only if
	// 1) NumberOfClusters is specified;
	// 2) And the number of matched ManagedClusters is less than NumberOfClusters;
	PlacementConditionSatisfied string = "PlacementSatisfied"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlacementList is a collection of Placements.
type PlacementList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of Placements.
	Items []Placement `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope="Namespaced"

// PlacementDecision indicates a decision from a placement
// PlacementDecision should has a label placement.open-cluster-management.io={placement name}
// to reference a certain placement.
//
// If a placement has NumberOfClusters specified, the total number of decisions contained in its PlacementDecisions
// should always be NumberOfClusters. Some of them might be empty when there are no enough ManagedClusters matched.
type PlacementDecision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Decisions is a slice of decisions according to a placement
	// The number of decisions should not be larger than 100
	// +kubebuilder:validation:Required
	// +required
	Decisions []ClusterDecision `json:"decisions"`
}

// ClusterDecision represents a decision from a placement
// An empty ClusterDecision indicates it is not scheduled yet.
type ClusterDecision struct {
	// ClusterName is the name of the ManagedCluster
	// +kubebuilder:validation:Required
	// +required
	ClusterName string `json:"clusterName"`

	// Reason represents the reason why the ManagedCluster is selected.
	// +kubebuilder:validation:Required
	// +required
	Reason string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterDecisionList is a collection of PlacementDecision.
type PlacementDecisionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of PlacementDecision.
	Items []PlacementDecision `json:"items"`
}
