// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package placementrule

import (
	"context"

	"github.com/golang/glog"

	appv1alpha1 "github.com/IBM/multicloud-operators-placementrule/pkg/apis/app/v1alpha1"
	"github.com/IBM/multicloud-operators-placementrule/pkg/utils"

	//	mcmv1alpha1 "github.ibm.com/IBMPrivateCloud/hcm-api/pkg/apis/mcm/v1alpha1"
	//	policyv1alpha1 "github.ibm.com/IBMPrivateCloud/hcm-compliance/pkg/apis/policy/v1alpha1"

	"sort"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcilePlacementRule) hubReconcile(instance *appv1alpha1.PlacementRule) error {
	clmap, err := r.prepareClusterAndStatusMaps(instance)
	if err != nil {
		glog.Error("Error in preparing clusters by status:", err)
		return err
	}

	err = r.filteClustersByStatus(instance, clmap /* , clstatusmap */)
	if err != nil {
		glog.Error("Error in filtering clusters by status:", err)
		return err
	}

	err = r.filteClustersByPolicies(instance, clmap /* , clstatusmap */)
	if err != nil {
		glog.Error("Error in filtering clusters by policy:", err)
		return err
	}

	// go without mcm repositories, removed identity check

	clidx := r.sortClustersByResourceHint(instance, clmap /* , clstatusmap */)

	newpd := r.pickClustersByReplicas(instance, clmap, clidx)

	instance.Status.Decisions = newpd

	return nil
}

// Top priority: clusterNames, ignore selector
// Bottomline: Use label selector
func (r *ReconcilePlacementRule) prepareClusterAndStatusMaps(instance *appv1alpha1.PlacementRule) (map[string]*clusterv1alpha1.Cluster, error) {
	if instance == nil {
		return nil /* nil, */, nil
	}

	clmap := make(map[string]*clusterv1alpha1.Cluster)

	var labelSelector *metav1.LabelSelector

	if instance.Spec.ClusterNames != nil {
		namereq := metav1.LabelSelectorRequirement{}
		namereq.Key = "name"
		namereq.Operator = metav1.LabelSelectorOpIn

		namereq.Values = append(namereq.Values, instance.Spec.ClusterNames...)
		labelSelector = &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{namereq},
		}
	} else {
		labelSelector = instance.Spec.ClusterLabels
	}
	clSelector, err := utils.ConvertLabels(labelSelector)
	if err != nil {
		return nil, err
	}
	glog.V(10).Info("Using Cluster LabelSelector ", clSelector)
	cllist := &clusterv1alpha1.ClusterList{}
	err = r.List(context.TODO(), &client.ListOptions{LabelSelector: clSelector}, cllist)
	if err != nil && !errors.IsNotFound(err) {
		glog.Error("Listing clusters and found error: ", err)
		return nil, err
	}

	for _, cl := range cllist.Items {
		clmap[cl.Name] = cl.DeepCopy()
	}

	return clmap, nil
}

func (r *ReconcilePlacementRule) filteClustersByStatus(instance *appv1alpha1.PlacementRule, clmap map[string]*clusterv1alpha1.Cluster) error {
	if instance == nil || instance.Spec.ClusterConditions == nil || clmap == nil {
		return nil
	}

	for k, cl := range clmap {
		keep := true
		for _, cond := range instance.Spec.ClusterConditions {
			condMatched := false
			for _, clcond := range cl.Status.Conditions {
				if cond.Type == clcond.Type {
					condMatched = true
					break
				}
			}
			if !condMatched {
				keep = false
				break
			}
		}

		glog.V(10).Info("Cond Check ", cl.Name, cl.Status.Conditions, keep)
		if !keep {
			delete(clmap, k)
		}
	}

	return nil
}

type clusterInfo struct {
	Name      string
	Namespace string
	Metrics   resource.Quantity
}

func (cinfo clusterInfo) DeepCopyInto(newinfo *clusterInfo) {
	newinfo.Name = cinfo.Name
	newinfo.Namespace = cinfo.Namespace
	cinfo.Metrics.DeepCopyInto(&(newinfo.Metrics))
}

type clusterIndex struct {
	Ascedent bool
	Clusters []clusterInfo
}

func (ci clusterIndex) Len() int {
	return len(ci.Clusters)
}

func (ci clusterIndex) Less(x, y int) bool {
	less := (ci.Clusters[x].Metrics.Cmp(ci.Clusters[y].Metrics) == -1)

	if !ci.Ascedent {
		return !less
	}
	return less
}

func (ci clusterIndex) Swap(x, y int) {
	tmp := clusterInfo{}
	ci.Clusters[x].DeepCopyInto(&tmp)
	ci.Clusters[y].DeepCopyInto(&(ci.Clusters[x]))
	tmp.DeepCopyInto(&(ci.Clusters[y]))
}

func (r *ReconcilePlacementRule) sortClustersByResourceHint(instance *appv1alpha1.PlacementRule,
	clmap map[string]*clusterv1alpha1.Cluster) *clusterIndex {
	sortedcls := &clusterIndex{}

	if instance == nil || clmap == nil || instance.Spec.ResourceHint == nil {
		return nil
	}

	sortedcls.Ascedent = false
	if instance.Spec.ResourceHint.Order == appv1alpha1.SelectionOrderAsce {
		sortedcls.Ascedent = true
	}

	for _, cl := range clmap {
		newcli := clusterInfo{
			Name:      cl.Name,
			Namespace: cl.Namespace,
		}

		sortedcls.Clusters = append(sortedcls.Clusters, newcli)
	}

	sort.Sort(sortedcls)

	return sortedcls
}

func (r *ReconcilePlacementRule) pickClustersByReplicas(instance *appv1alpha1.PlacementRule,
	clmap map[string]*clusterv1alpha1.Cluster, clidx *clusterIndex) []appv1alpha1.PlacementDecision {
	newpd := []appv1alpha1.PlacementDecision{}
	total := len(clmap)
	if instance.Spec.ClusterReplicas != nil && total > int(*(instance.Spec.ClusterReplicas)) {
		total = int(*instance.Spec.ClusterReplicas)
	}
	picked := 0

	// no sort, pick existing decisions first, then clmap
	if clidx == nil {
		for _, cli := range instance.Status.Decisions {
			// check if still eligible
			if _, ok := clmap[cli.ClusterName]; !ok {
				continue
			}
			if picked < total {
				newpd = append(newpd, *cli.DeepCopy())
				delete(clmap, cli.ClusterName)
				picked++
			} else {
				break
			}
		}
		for _, cl := range clmap {
			if picked < total {
				pd := appv1alpha1.PlacementDecision{
					ClusterName:      cl.Name,
					ClusterNamespace: cl.Namespace,
				}
				newpd = append(newpd, pd)
				picked++
			} else {
				break
			}
		}
	} else {
		// sort by something
		for _, cli := range clidx.Clusters {
			if _, ok := clmap[cli.Name]; !ok {
				continue
			}
			if picked < total {
				pd := appv1alpha1.PlacementDecision{
					ClusterName:      cli.Name,
					ClusterNamespace: cli.Namespace,
				}
				newpd = append(newpd, pd)
				picked++
			} else {
				break
			}
		}
	}
	glog.V(10).Info("New decisions for ", instance.Name, ": ", newpd)
	return newpd
}
func (r *ReconcilePlacementRule) filteClustersByPolicies(instance *appv1alpha1.PlacementRule,
	clmap map[string]*clusterv1alpha1.Cluster /* , clstatusmap map[string]*mcmv1alpha1.ClusterStatus */) error {
	if instance == nil || instance.Spec.Policies == nil || clmap == nil {
		return nil
	}

	return nil
}
