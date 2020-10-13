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

package utils

import (
	"reflect"

	"k8s.io/klog"

	spokeClusterV1 "github.com/open-cluster-management/api/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ClusterPredicateFunc defines predicate function for cluster related watch, main purpose is to ignore heartbeat without change
var ClusterPredicateFunc = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldcl := e.ObjectOld.(*spokeClusterV1.ManagedCluster)
		newcl := e.ObjectNew.(*spokeClusterV1.ManagedCluster)

		//if managed cluster is being deleted
		if !reflect.DeepEqual(oldcl.DeletionTimestamp, newcl.DeletionTimestamp) {
			return true
		}

		if !reflect.DeepEqual(oldcl.Labels, newcl.Labels) {
			return true
		}

		oldcondMap := make(map[string]metav1.ConditionStatus)
		for _, cond := range oldcl.Status.Conditions {
			oldcondMap[cond.Type] = cond.Status
		}
		for _, cond := range newcl.Status.Conditions {
			oldcondst, ok := oldcondMap[cond.Type]
			if !ok || oldcondst != cond.Status {
				return true
			}
			delete(oldcondMap, cond.Type)
		}

		if len(oldcondMap) > 0 {
			return true
		}

		klog.V(1).Info("Out Cluster Predicate Func ", oldcl.Name, " with false possitive")
		return false
	},
}
