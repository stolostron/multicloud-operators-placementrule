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

package gitopscluster

import (
	"context"
	"errors"
	"fmt"

	gitopsclusterV1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ReconcileGitOpsCluster reconciles a GitOpsCluster object.
type ReconcileGitOpsCluster struct {
	client.Client
}

// Add creates a new argocd cluster Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

var _ reconcile.Reconciler = &ReconcileGitOpsCluster{}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	authCfg := mgr.GetConfig()
	klog.Infof("Host: %v, BearerToken: %v", authCfg.Host, authCfg.BearerToken)

	dsRS := &ReconcileGitOpsCluster{
		Client: mgr.GetClient(),
	}

	return dsRS
}

type argocdSecretRuleMapper struct {
	client.Client
}

func (mapper *argocdSecretRuleMapper) Map(obj handler.MapObject) []reconcile.Request {
	// if managed cluster secret is updated, reconcile its secret in argo namespace
	var requests []reconcile.Request

	managedClusterSecretKey := types.NamespacedName{
		Name:      obj.Meta.GetName(),
		Namespace: utils.GetManagedClusterNamespace(obj.Meta.GetName())}

	requests = append(requests, reconcile.Request{NamespacedName: managedClusterSecretKey})

	return requests
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gitopscluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if utils.IsReadyACMClusterRegistry(mgr.GetAPIReader()) {
		// Watch gitopscluster changes
		err = c.Watch(
			&source.Kind{Type: &gitopsclusterV1alpha1.GitOpsCluster{}},
			&handler.EnqueueRequestForObject{},
			utils.GitOpsClusterPredicateFunc)
		if err != nil {
			return err
		}

		// Watch for managed cluster secret changes in managed cluster namespaces
		err = c.Watch(
			&source.Kind{Type: &v1.Secret{}},
			&handler.EnqueueRequestForObject{},
			utils.AcmClusterSecretPredicateFunc)
		if err != nil {
			return err
		}

		// Watch for managed cluster secret changes in argo namespaces
		err = c.Watch(
			&source.Kind{Type: &v1.Secret{}},
			&handler.EnqueueRequestForObject{},
			utils.ArgocdClusterSecretPredicateFunc)
		if err != nil {
			return err
		}
		// TODO: watch placement and placementdecision changes
	}

	return nil
}

func (r *ReconcileGitOpsCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	// What could have changed?
	//  - GitOpsCluster
	//       - Create/update all managed cluster secrets into the argo namespace
	//       - If it was updated, how about stale secrets?
	//  - Placement/PlacementDecision
	//       - Find all GitOpsCluster with placementRef
	//       - Create/update all managed cluster secrets into the argo namespace
	//       - How about stale secrets?
	//  - Managed cluster secret in managed cluster namespace
	//       - Find all secrets with label apps.open-cluster-management.io/cluster-name: <clusterName> and create/update or delete
	//  - Managed cluster secret in other namespace

	// Just delete all Managed cluster secret in other namespaces
	// AND loop through all GitOpsCluster CRs
	//    Create all managed cluster secrets into the argo namespace

	klog.Info("Reconciling GitOpsClusters for watched resource change: ", request.NamespacedName)

	// Get all existing managed cluster secrets from Argo namespaces
	// Delete all existing managed cluster secrets from Argo namespaces
	// Get all GitOpsCluster CRs
	// Verify the argocd namespace
	// For each,
	//   Get all managed clusters
	//   For each managed cluster, copy the secret over

	managedClusterSecretsInArgoList := map[types.NamespacedName]string{}

	managedClusterSecretsInArgo, err := r.GetAllManagedClusterSecretsInArgo()

	if err != nil {
		return reconcile.Result{Requeue: false}, nil
	}

	for _, secret := range managedClusterSecretsInArgo.Items {
		klog.Info("Delete secret: " + secret.Namespace + "/" + secret.Name)
		managedClusterSecretsInArgoList[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}] = secret.Namespace + "/" + secret.Name
	}

	gitOpsClusters, err := r.GetAllGitOpsClusters()

	if err != nil {
		return reconcile.Result{Requeue: false}, nil
	}

	for _, gitOpsCluster := range gitOpsClusters.Items {
		klog.Info("Process GitOpsCluster: " + gitOpsCluster.Namespace + "/" + gitOpsCluster.Name)

		instance := &gitopsclusterV1alpha1.GitOpsCluster{}

		err := r.Get(context.TODO(), types.NamespacedName{Name: gitOpsCluster.Name, Namespace: gitOpsCluster.Namespace}, instance)

		if err != nil && k8errors.IsNotFound(err) {
			klog.Infof("GitOpsCluster %s/%s deleted", gitOpsCluster.Namespace, gitOpsCluster.Name)

			return reconcile.Result{}, nil
		}

		// 1. Verify that spec.argoServer.argoNamespace is a valid ArgoCD namespace
		if !r.VerifyArgocdNamespace(gitOpsCluster.Spec.ArgoServer.ArgoNamespace) {
			klog.Info("invalid argocd namespace")
			instance.Status.LastUpdateTime = metav1.Now()
			instance.Status.Phase = "failed"
			instance.Status.Message = "invalid argocd namespace"
			r.Client.Status().Update(context.TODO(), instance)
		}

		// 2. Get the list of managed clusters
		managedClusters, err := r.GetManagedClusters(*instance.Spec.PlacementRef)
		// 2a. Get the placement decision
		// 2b. Get the managed cluster names from the placement decision
		if err != nil {
			klog.Info("failed to get managed cluster list")
			instance.Status.LastUpdateTime = metav1.Now()
			instance.Status.Phase = "failed"
			instance.Status.Message = err.Error()
			r.Client.Status().Update(context.TODO(), instance)
		}

		klog.Infof("argo namespace: %s", instance.Spec.ArgoServer.ArgoNamespace)
		klog.Infof("managed cluster list: %v", managedClusters)

		// 3. Copy secret contents from the managed cluster namespaces and create the secret in spec.argoServer.argoNamespace
		err = r.AddManagedClustersToArgo(instance.Spec.ArgoServer.ArgoNamespace, managedClusters, managedClusterSecretsInArgoList)

		if err != nil {
			klog.Info("failed to add managed clusters to argo")
			instance.Status.LastUpdateTime = metav1.Now()
			instance.Status.Phase = "failed"
			instance.Status.Message = err.Error()
			r.Client.Status().Update(context.TODO(), instance)
		}

		instance.Status.LastUpdateTime = metav1.Now()
		instance.Status.Phase = "successful"
		instance.Status.Message = fmt.Sprintf("Imported managed clusters: %v", managedClusters)
		r.Client.Status().Update(context.TODO(), instance)
	}

	for key, secretName := range managedClusterSecretsInArgoList {
		secretToDelete := &v1.Secret{}

		err := r.Get(context.TODO(), key, secretToDelete)

		if err == nil {
			klog.Infof("Deleting orphan managed cluster secret %s", secretName)

			err = r.Delete(context.TODO(), secretToDelete)

			if err != nil {
				klog.Errorf("failed to delete orphan managed cluster secret %s, err: ", key, err.Error())
			}
		}
	}

	return reconcile.Result{}, err
}

func (r *ReconcileGitOpsCluster) GetAllManagedClusterSecretsInArgo() (v1.SecretList, error) {
	klog.Info("Getting all managed cluster secrets from argo namespaces")

	secretList := &v1.SecretList{}
	listopts := &client.ListOptions{}

	secretSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"apps.open-cluster-management.io/acm-cluster": "true",
			"argocd.argoproj.io/secret-type":              "cluster",
		},
	}

	secretSelectionLabel, err := utils.ConvertLabels(secretSelector)
	if err != nil {
		klog.Error("Failed to convert managed cluster secret selector, err:", err)
		return *secretList, err
	}

	listopts.LabelSelector = secretSelectionLabel
	err = r.List(context.TODO(), secretList, listopts)

	if err != nil {
		klog.Error("Failed to list managed cluster secrets in argo, err:", err)
		return *secretList, err
	}

	return *secretList, nil
}

func (r *ReconcileGitOpsCluster) GetAllGitOpsClusters() (gitopsclusterV1alpha1.GitOpsClusterList, error) {
	klog.Info("Getting all GitOpsCluster resources")

	gitOpsClusterList := &gitopsclusterV1alpha1.GitOpsClusterList{}

	err := r.List(context.TODO(), gitOpsClusterList)

	if err != nil {
		klog.Error("Failed to list GitOpsCluster resources, err:", err)
		return *gitOpsClusterList, err
	}

	return *gitOpsClusterList, nil
}

func (r *ReconcileGitOpsCluster) VerifyArgocdNamespace(argoNamespace string) bool {
	// Check if argo server pod exists in the namespace
	isArgoCDNamespace := r.FindPodsWithLabelsAndNamespace(argoNamespace, map[string]string{"app.kubernetes.io/name": "argocd-server"})

	// Check if opernshift gitops argo server pod exists in the namespace
	isGitOpsNamespace := r.FindPodsWithLabelsAndNamespace(argoNamespace, map[string]string{"app.kubernetes.io/name": "openshift-gitops-server"})

	return isArgoCDNamespace || isGitOpsNamespace
}

func (r *ReconcileGitOpsCluster) FindPodsWithLabelsAndNamespace(namespace string, labels map[string]string) bool {
	podList := &v1.PodList{}
	listopts := &client.ListOptions{}

	podSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	podLabels, err := utils.ConvertLabels(podSelector)
	if err != nil {
		klog.Error("Failed to convert label selector, err:", err)
		return false
	}

	listopts.LabelSelector = podLabels
	listopts.Namespace = namespace
	err = r.List(context.TODO(), podList, listopts)

	if err != nil {
		klog.Error("Failed to list pods, err:", err)
		return false
	}

	if len(podList.Items) == 0 {
		klog.Errorf("No pod with labels %v found", labels)
		return false
	}

	for _, pod := range podList.Items {
		klog.Info("Found pod ", pod.GetName(), " in namespace ", pod.GetNamespace())
	}

	return true
}

func (r *ReconcileGitOpsCluster) GetManagedClusters(placementref v1.ObjectReference) ([]string, error) {
	if placementref.Kind != "Placement" ||
		placementref.APIVersion != "cluster.open-cluster-management.io/v1alpha1" {
		return nil, errors.New("invalid placementref kind: " + placementref.Kind + " apiVersion: " + placementref.APIVersion)
	}

	// TODO: Find the placementdecision with placementref.Name and placementref.Namespace
	//       and get the list of managed clusters

	return []string{"local-cluster"}, nil
}

func (r *ReconcileGitOpsCluster) AddManagedClustersToArgo(argoNamespace string, managedClusters []string, allSecretsList map[types.NamespacedName]string) error {
	for _, managedCluster := range managedClusters {
		klog.Infof("adding managed cluster %s to argo", managedCluster)

		secretName := managedCluster + "-cluster-secret"
		managedClusterSecretKey := types.NamespacedName{Name: secretName, Namespace: managedCluster}

		managedClusterSecret := &v1.Secret{}
		err := r.Get(context.TODO(), managedClusterSecretKey, managedClusterSecret)

		if err != nil {
			klog.Error("failed to get managed cluster secret. err: ", err.Error())
			return err
		}

		err = r.CreateManagedClusterSecretInArgo(argoNamespace, *managedClusterSecret)

		if err != nil {
			klog.Error("failed to create managed cluster secret. err: ", err.Error())
			return err
		}

		argoManagedClusterSecretKey := types.NamespacedName{Name: secretName, Namespace: argoNamespace}
		delete(allSecretsList, argoManagedClusterSecretKey)
	}

	return nil
}

func (r *ReconcileGitOpsCluster) CreateManagedClusterSecretInArgo(argoNamespace string, managedClusterSecret v1.Secret) error {
	labels := managedClusterSecret.GetLabels()

	// create the new cluster secret in the argocd server namespace
	newSecret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      managedClusterSecret.Name,
			Namespace: argoNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type":                 "cluster",
				"apps.open-cluster-management.io/acm-cluster":    "true",
				"apps.open-cluster-management.io/cluster-name":   labels["apps.open-cluster-management.io/cluster-name"],
				"apps.open-cluster-management.io/cluster-server": labels["apps.open-cluster-management.io/cluster-server"],
			},
		},
		Type: "Opaque",
		StringData: map[string]string{
			"config": string(managedClusterSecret.Data["config"]),
			"name":   string(managedClusterSecret.Data["name"]),
			"server": string(managedClusterSecret.Data["server"]),
		},
	}

	existingManagedClusterSecret := &v1.Secret{}

	err := r.Get(context.TODO(), types.NamespacedName{Name: managedClusterSecret.Name, Namespace: argoNamespace}, existingManagedClusterSecret)

	if err != nil && k8errors.IsNotFound(err) {
		klog.Infof("creating managed cluster secret in argo namespace: %v/%v", argoNamespace, managedClusterSecret.Name)

		err := r.Create(context.TODO(), newSecret)

		if err != nil {
			klog.Errorf("failed to create managed cluster secret. name: %v/%v, error: %v", argoNamespace, managedClusterSecret.Name, err)
			return err
		}
	} else {
		klog.Infof("updating managed cluster secret in argo namespace: %v/%v", argoNamespace, managedClusterSecret.Name)

		err := r.Update(context.TODO(), newSecret)

		if err != nil {
			klog.Errorf("failed to update managed cluster secret. name: %v/%v, error: %v", argoNamespace, managedClusterSecret.Name, err)
			return err
		}
	}

	return nil
}
