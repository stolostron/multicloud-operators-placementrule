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

package argocdcluster

import (
	"context"

	agentv1 "github.com/open-cluster-management/klusterlet-addon-controller/pkg/apis/agent/v1"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileSecret) hubReconcile(acmClusterSecret *v1.Secret) error {
	newArgocdSecret := ArgocdCluster{}
	newArgocdSecret.SecretName = acmClusterSecret.Name
	newArgocdSecret.ArgocdSecretNamespace = ArgocdServerNamespace
	newArgocdSecret.ACMSecretNamespace = acmClusterSecret.Namespace
	newArgocdSecret.ClusterConfig = string(acmClusterSecret.Data["config"])
	newArgocdSecret.ClusterName = string(acmClusterSecret.Data["name"])

	newClusterServer := string(acmClusterSecret.Data["server"])

	argocdClusterSecretKey := types.NamespacedName{Name: acmClusterSecret.Name, Namespace: ArgocdServerNamespace}

	if ArgocdClusterSecret := r.GetClusterSecret(argocdClusterSecretKey); ArgocdClusterSecret != nil {
		if newArgocdSecret.ClusterConfig == string(ArgocdClusterSecret.Data["config"]) &&
			newClusterServer == string(ArgocdClusterSecret.Data["server"]) &&
			newArgocdSecret.ClusterName == string(ArgocdClusterSecret.Data["name"]) {
			klog.Infof("Skiping the same argocd cluster secret, ArgoCD Cluster Secret: %v", argocdClusterSecretKey.String())
			return nil
		}

		klog.Infof("Updating argocd cluster secret, ArgoCD Cluster Secret: %v", argocdClusterSecretKey.String())

		r.lock.Lock()
		if err := r.DeleteClusterSecret(argocdClusterSecretKey); err != nil {
			return err
		}

		err := r.CreateClusterSecret(newClusterServer, newArgocdSecret)
		r.lock.Unlock()

		return err
	}

	klog.Infof("Creating new argocd cluster secret, ArgoCD Cluster Secret: %v", argocdClusterSecretKey.String())

	return r.CreateClusterSecret(newClusterServer, newArgocdSecret)
}

// GetClusterSecret get ArgoCD/ACM cluster secet by given secret name and secret namespace
func (r *ReconcileSecret) GetClusterSecret(existingSecretKey types.NamespacedName) *v1.Secret {
	existingSecret := &v1.Secret{}

	err := r.Get(context.TODO(), existingSecretKey, existingSecret)
	if err == nil {
		return existingSecret
	}

	klog.Warningf("Cluster Secret not found: key: %v", existingSecretKey.String())

	return nil
}

// DeleteClusterSecret delete k8s secret
func (r *ReconcileSecret) DeleteClusterSecret(existingSecretKey types.NamespacedName) error {
	existingSecret := r.GetClusterSecret(existingSecretKey)

	if existingSecret != nil {
		err := r.Delete(context.TODO(), existingSecret)
		if err != nil {
			klog.Errorf("Error in deleting existing argocd cluster secret, key: %v, err: %v ", existingSecretKey.String(), err)
			return err
		}
	}

	return nil
}

func (r *ReconcileSecret) CreateClusterSecret(newClusterServer string, newArgocdSecret ArgocdCluster) error {
	// create the new cluster secret in the argocd server namespace
	newSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      newArgocdSecret.SecretName,
			Namespace: ArgocdServerNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type":              "cluster",
				"apps.open-cluster-management.io/acm-cluster": "true",
			},
		},
		Type: "Opaque",
		StringData: map[string]string{
			"config": newArgocdSecret.ClusterConfig,
			"name":   newArgocdSecret.ClusterName,
			"server": newClusterServer,
		},
	}

	klog.Infof("Creating new ArgoCD cluster secret: %v/%v", ArgocdServerNamespace, newArgocdSecret.SecretName)

	err := r.Create(context.TODO(), newSecret)

	if err != nil {
		klog.Errorf("Failed to create new ArgoCD cluster secret. name: %v/%v, error: %v", ArgocdServerNamespace, newArgocdSecret.SecretName, err)
		return err
	}

	return nil
}

//DeleteAllArgocdClusterSecrets delete all ArgoCD ACM cluster secrets from ArgoCD namespace
func (r *ReconcileSecret) DeleteAllArgocdClusterSecrets() error {
	if ArgocdServerNamespace == "" {
		klog.Info("empty argocd Server Namespace, do nothing...")
		return nil
	}

	ArgoCDSecretList := &v1.SecretList{}
	listopts := &client.ListOptions{Namespace: ArgocdServerNamespace}

	ArgocdClusterSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"argocd.argoproj.io/secret-type":              "cluster",
			"apps.open-cluster-management.io/acm-cluster": "true",
		},
	}

	ArgocdClusterLabel, err := utils.ConvertLabels(ArgocdClusterSelector)
	if err != nil {
		klog.Error("Failed to convert ArgoCD Cluster Selector, err:", err)
		return err
	}

	listopts.LabelSelector = ArgocdClusterLabel
	err = r.List(context.TODO(), ArgoCDSecretList, listopts)

	if err != nil {
		klog.Error("Failed to list ArgoCD cluster secrets, err:", err)
		return err
	}

	for _, sl := range ArgoCDSecretList.Items {
		curSecret := sl.DeepCopy()
		err := r.Delete(context.TODO(), curSecret)

		if err != nil {
			klog.Errorf("Error in deleting existing argocd cluster secret. ArgoCD server namespace: %v, key: %v/%v, err: %v",
				ArgocdServerNamespace, sl.Namespace, sl.Name, err)
		}
	}

	return nil
}

// loadClusterSecret load cluster secrets to map
// 1. all ACM cluster secrets in all managed cluster namespaces
// 2. all ArgoCD cluster secrets in ArgoCD server namespace
func (r *ReconcileSecret) loadClusterSecret(clusterSecrets ArgocdClusters, secretType string,
	matchLabel map[string]string) error {
	secretList := &v1.SecretList{}

	listopts := &client.ListOptions{}
	if secretType == "ArgoCD" {
		listopts = &client.ListOptions{Namespace: ArgocdServerNamespace}
	}

	clusterSelector := &metav1.LabelSelector{
		MatchLabels: matchLabel,
	}

	clusterLabel, err := utils.ConvertLabels(clusterSelector)
	if err != nil {
		klog.Error("Failed to convert Cluster Selector, err:", err)
		return err
	}

	listopts.LabelSelector = clusterLabel
	err = r.List(context.TODO(), secretList, listopts)

	if err != nil {
		klog.Error("Failed to list cluster secrets, err:", err)
		return err
	}

	for _, sl := range secretList.Items {
		newArgocdSecret := ArgocdCluster{}
		newArgocdSecret.SecretName = sl.Name
		newArgocdSecret.ArgocdSecretNamespace = ArgocdServerNamespace

		if secretType == "ArgoCD" {
			newArgocdSecret.ACMSecretNamespace = utils.GetACMClusterNamespace(sl.Name)
		} else if secretType == "ACM" {
			newArgocdSecret.ACMSecretNamespace = sl.Namespace
		} else {
			newArgocdSecret.ACMSecretNamespace = ""
			klog.Warningf("ACM cluster secret namespace not found, secret: %v", sl)
		}

		newArgocdSecret.ClusterConfig = string(sl.Data["config"])
		newArgocdSecret.ClusterName = string(sl.Data["name"])

		newClusterServer := string(sl.Data["server"])

		clusterSecrets[newClusterServer] = newArgocdSecret
	}

	return nil
}

func (r *ReconcileSecret) loadArgocdClusterSettings(allArgocdClusterSettings AddonargocdClusterSettings) error {
	addonConfigList := &agentv1.KlusterletAddonConfigList{}

	listopts := &client.ListOptions{}
	err := r.List(context.TODO(), addonConfigList, listopts)

	if err != nil {
		klog.Error("Failed to list KlusterletAddonConfig, err:", err)
		return err
	}

	for _, ac := range addonConfigList.Items {
		if !ac.Spec.ApplicationManagerConfig.Enabled {
			allArgocdClusterSettings[ac.Namespace] = false
			continue
		}

		allArgocdClusterSettings[ac.Namespace] = ac.Spec.ApplicationManagerConfig.ArgoCDCluster
	}

	return nil
}

func (r *ReconcileSecret) SyncArgocdClusterSecrets() error {
	if ArgocdServerNamespace == "" {
		klog.Info("empty argocd Server Namespace, do nothing...")
		return nil
	}

	//initialize ACM cluster secrets map
	allACMClusterSecrets := make(ArgocdClusters)
	AcmClusterLabel := map[string]string{
		"apps.open-cluster-management.io/secret-type": "acm-cluster",
	}

	if err := r.loadClusterSecret(allACMClusterSecrets, "ACM", AcmClusterLabel); err != nil {
		return err
	}

	//initialize ArgoCD cluster secrets map
	allArgocdClusterSecrets := make(ArgocdClusters)
	ArgocdClusterLabel := map[string]string{
		"argocd.argoproj.io/secret-type":              "cluster",
		"apps.open-cluster-management.io/acm-cluster": "true",
	}

	if err := r.loadClusterSecret(allArgocdClusterSecrets, "ArgoCD", ArgocdClusterLabel); err != nil {
		return err
	}

	//initialize klusterletAddonConfig map for all managed clusters
	allArgocdClusterSettings := make(AddonargocdClusterSettings)

	if err := r.loadArgocdClusterSettings(allArgocdClusterSettings); err != nil {
		return err
	}

	klog.Infof("Start Syncing ACM clusters to ArgoCD. ACM clusters: %v, ArgoCD clusters: %v, ArgoCD Namespace: %v",
		len(allACMClusterSecrets), len(allArgocdClusterSecrets), ArgocdServerNamespace)

	for acmServer, acmClusterSecret := range allACMClusterSecrets {
		if !allArgocdClusterSettings[acmClusterSecret.ACMSecretNamespace] {
			// The ACM cluster secret will be deleted from argocd during the following orphan argocd cluster secrets clear-up
			klog.Infof("The argocd cluster collection for this ACM cluster is disabled, cluster: %v",
				acmClusterSecret.ACMSecretNamespace)
			continue
		}

		if argocdClusterSecret, ok := allArgocdClusterSecrets[acmServer]; ok {
			if argocdClusterSecret.ClusterConfig == acmClusterSecret.ClusterConfig &&
				argocdClusterSecret.ClusterName == acmClusterSecret.ClusterName &&
				argocdClusterSecret.SecretName == acmClusterSecret.SecretName {
				// same argocd cluster secret found, no need to update
				delete(allArgocdClusterSecrets, acmServer)
				continue
			} else {
				// different argocd cluster secret found, need to sync the acm cluster secret to argocd
				argocdSecretKey := types.NamespacedName{Name: argocdClusterSecret.SecretName, Namespace: argocdClusterSecret.ArgocdSecretNamespace}

				r.lock.Lock()
				if r.DeleteClusterSecret(argocdSecretKey) == nil {
					_ = r.CreateClusterSecret(acmServer, acmClusterSecret)
					delete(allArgocdClusterSecrets, acmServer)
				}
				r.lock.Unlock()
			}
		} else {
			// new acm cluster found.
			_ = r.CreateClusterSecret(acmServer, acmClusterSecret)
		}
	}

	// clear up orphan argocd cluster secrets
	for _, argocdClusterSecret := range allArgocdClusterSecrets {
		argocdSecretKey := types.NamespacedName{Name: argocdClusterSecret.SecretName, Namespace: argocdClusterSecret.ArgocdSecretNamespace}
		_ = r.DeleteClusterSecret(argocdSecretKey)
	}

	return nil
}
