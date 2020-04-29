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
	"k8s.io/klog"

	appv1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"

	"sort"

	"k8s.io/apimachinery/pkg/api/resource"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func (r *ReconcilePlacementRule) hubReconcile(instance *appv1alpha1.PlacementRule) error {
	clmap, err := utils.PlaceByGenericPlacmentFields(r.Client, instance.Spec.GenericPlacementFields, r.authClient, instance)
	if err != nil {
		klog.Error("Error in preparing clusters by status:", err)
		return err
	}

	err = r.filteClustersByStatus(instance, clmap /* , clstatusmap */)
	if err != nil {
		klog.Error("Error in filtering clusters by status:", err)
		return err
	}

	err = r.filteClustersByUser(instance, clmap)
	if err != nil {
		klog.Error("Error in filtering clusters by user Identity:", err)
		return err
	}

	err = r.filteClustersByPolicies(instance, clmap /* , clstatusmap */)
	if err != nil {
		klog.Error("Error in filtering clusters by policy:", err)
		return err
	}

	// go without mcm repositories, removed identity check

	clidx := r.sortClustersByResourceHint(instance, clmap /* , clstatusmap */)

	newpd := r.pickClustersByReplicas(instance, clmap, clidx)

	instance.Status.Decisions = newpd

	return nil
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

		klog.V(10).Info("Cond Check ", cl.Name, cl.Status.Conditions, keep)

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

	klog.V(10).Info("New decisions for ", instance.Name, ": ", newpd)

	return newpd
}

func (r *ReconcilePlacementRule) filteClustersByPolicies(instance *appv1alpha1.PlacementRule,
	clmap map[string]*clusterv1alpha1.Cluster /* , clstatusmap map[string]*mcmv1alpha1.ClusterStatus */) error {
	if instance == nil || instance.Spec.Policies == nil || clmap == nil {
		return nil
	}

	return nil
}

func (r *ReconcilePlacementRule) filteClustersByUser(instance *appv1alpha1.PlacementRule,
	clmap map[string]*clusterv1alpha1.Cluster) error {
	if instance == nil || clmap == nil {
		return nil
	}

	annotations := instance.GetAnnotations()
	if annotations == nil {
		return nil
	}

	if _, ok := annotations[appv1alpha1.UserIdentityAnnotation]; !ok {
		return nil
	}

	err := utils.FilteClustersByIdentity(r.authClient, instance, clmap)

	if err != nil {
		return err
	}

	return nil
}
