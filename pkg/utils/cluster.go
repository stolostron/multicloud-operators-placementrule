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
	"encoding/base64"
	"reflect"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	spokeClusterV1 "github.com/open-cluster-management/api/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// #nosec G101
	ACMClusterSecretLabel = "apps.open-cluster-management.io/secret-type"
	// #nosec G101
	ArgocdClusterSecretLabel = "apps.open-cluster-management.io/acm-cluster"
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

// AcmClusterSecretPredicateFunc defines predicate function for ACM cluster secrets watch
var AcmClusterSecretPredicateFunc = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldSecret, ok := e.ObjectOld.(*v1.Secret)
		if !ok {
			return false
		}

		newSecret, nok := e.ObjectNew.(*v1.Secret)
		if !nok {
			return false
		}

		oldSecretType, ok := e.MetaOld.GetLabels()[ACMClusterSecretLabel]
		newSecretType, nok := e.MetaNew.GetLabels()[ACMClusterSecretLabel]

		if ok && oldSecretType == "acm-cluster" {
			klog.Infof("Update a old ACM cluster secret, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
			return true
		}

		if nok && newSecretType == "acm-cluster" {
			klog.Infof("Update a new ACM cluster secret, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
			return true
		}

		klog.Infof("Not a ACM cluster secret update, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
		return false
	},
	CreateFunc: func(e event.CreateEvent) bool {
		SecretType, ok := e.Meta.GetLabels()[ACMClusterSecretLabel]

		if !ok {
			return false
		} else if SecretType != "acm-cluster" {
			return false
		}

		klog.Infof("Create a ACM cluster secret: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		SecretType, ok := e.Meta.GetLabels()[ACMClusterSecretLabel]

		if !ok {
			return false
		} else if SecretType != "acm-cluster" {
			return false
		}

		klog.Infof("Delete a ACM cluster secret: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
}

// ArgocdClusterSecretPredicateFunc defines predicate function for ArgoCD cluster secrets watch
var ArgocdClusterSecretPredicateFunc = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldSecret, ok := e.ObjectOld.(*v1.Secret)
		if !ok {
			return false
		}

		newSecret, nok := e.ObjectNew.(*v1.Secret)
		if !nok {
			return false
		}

		oldSecretType, ok := e.MetaOld.GetLabels()[ArgocdClusterSecretLabel]
		newSecretType, nok := e.MetaNew.GetLabels()[ArgocdClusterSecretLabel]

		if ok && oldSecretType == "true" {
			klog.Infof("Update a old ArgoCD cluster secret, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
			return true
		}

		if nok && newSecretType == "true" {
			klog.Infof("Update a new Argocd cluster secret, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
			return true
		}

		klog.Infof("Not a ArgoCD cluster secret update, old: %v/%v, new: %v/%v", oldSecret.Namespace, oldSecret.Name, newSecret.Namespace, newSecret.Name)
		return false
	},
	CreateFunc: func(e event.CreateEvent) bool {
		SecretType, ok := e.Meta.GetLabels()[ArgocdClusterSecretLabel]

		if !ok {
			return false
		} else if SecretType != "true" {
			return false
		}

		klog.Infof("Create a ArgoCD cluster secret: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		SecretType, ok := e.Meta.GetLabels()[ArgocdClusterSecretLabel]

		if !ok {
			return false
		} else if SecretType != "true" {
			return false
		}

		klog.Infof("Delete a ArgoCD cluster secret: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
}

// ArgocdServerPredicateFunc defines predicate function for cluster related watch
var ArgocdServerPredicateFunc = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldPod, ok := e.ObjectOld.(*v1.Pod)
		if !ok {
			return false
		}

		newPod, nok := e.ObjectNew.(*v1.Pod)
		if !nok {
			return false
		}

		oldArgocdServerLabel, ok := e.MetaOld.GetLabels()["app.kubernetes.io/name"]
		newArgocdServerLabel, nok := e.MetaNew.GetLabels()["app.kubernetes.io/name"]

		if ok && oldArgocdServerLabel == "argocd-server" {
			klog.Infof("Update a old ArgoCD Server Pod, old: %v/%v, new: %v/%v", oldPod.Namespace, oldPod.Name, newPod.Namespace, newPod.Name)
			return true
		}

		if nok && newArgocdServerLabel == "argocd-server" {
			klog.Infof("Update a new ArgoCD Server Pod, old: %v/%v, new: %v/%v", oldPod.Namespace, oldPod.Name, newPod.Namespace, newPod.Name)
			return true
		}

		klog.Infof("Not a ArgoCD Server Pod, old: %v/%v, new: %v/%v", oldPod.Namespace, oldPod.Name, newPod.Namespace, newPod.Name)
		return false
	},
	CreateFunc: func(e event.CreateEvent) bool {
		ArgocdServerLabel, ok := e.Meta.GetLabels()["app.kubernetes.io/name"]

		if !ok {
			return false
		} else if ArgocdServerLabel != "argocd-server" {
			return false
		}

		klog.Infof("Create a ArgoCD Server Pod: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		ArgocdServerLabel, ok := e.Meta.GetLabels()["app.kubernetes.io/name"]

		if !ok {
			return false
		} else if ArgocdServerLabel != "argocd-server" {
			return false
		}

		klog.Infof("Delete a ArgoCD Server Pod: %v/%v", e.Meta.GetNamespace(), e.Meta.GetName())
		return true
	},
}

// Base64StringDecode decode a base64 string
func Base64StringDecode(encodedStr string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		klog.Errorf("Failed to base64 decode, err: %v", err)
		return "", err
	}

	return string(decodedBytes), nil
}

// GetACMClusterNamespace return ACM secret namespace accoding to its secret name
func GetACMClusterNamespace(secretName string) string {
	if secretName == "" {
		return ""
	}

	if strings.HasSuffix(secretName, "-cluster-secret") {
		return strings.TrimSuffix(secretName, "-cluster-secret")
	}

	klog.Errorf("invalid ACM cluster secret name, secretName: %v", secretName)

	return ""
}
