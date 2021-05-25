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
	"os"
	"sync"
	"time"

	spokeClusterV1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	agentv1 "github.com/open-cluster-management/klusterlet-addon-controller/pkg/apis/agent/v1"
	gitopsclusterV1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	authClient kubernetes.Interface
	scheme     *runtime.Scheme
	lock       sync.Mutex
}

// Add creates a new argocd cluster Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

var _ reconcile.Reconciler = &ReconcileGitOpsCluster{}

var errInvalidPlacementRef = errors.New("invalid placement reference")

var migrationDone = false

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	authCfg := mgr.GetConfig()
	klog.Infof("Host: %v, BearerToken: %v", authCfg.Host, authCfg.BearerToken)
	kubeClient := kubernetes.NewForConfigOrDie(authCfg)

	dsRS := &ReconcileGitOpsCluster{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		authClient: kubeClient,
		lock:       sync.Mutex{},
	}

	return dsRS
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

		// Watch for managed cluster secret changes in argo or managed cluster namespaces
		// The manager started with cache that filters all other secrets so no predicate needed
		err = c.Watch(
			&source.Kind{Type: &v1.Secret{}},
			&handler.EnqueueRequestForObject{},
			utils.ManagedClusterSecretPredicateFunc)
		if err != nil {
			return err
		}

		// Watch cluster list changes in placement decision
		err = c.Watch(
			&source.Kind{Type: &clusterv1alpha1.PlacementDecision{}},
			&handler.EnqueueRequestForObject{},
			utils.PlacementDecisionPredicateFunc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileGitOpsCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling GitOpsClusters for watched resource change: ", request.NamespacedName)

	// Get all existing GitOps managed cluster secrets, not the ones from the managed cluster namespaces
	managedClusterSecretsInArgo, err := r.GetAllManagedClusterSecretsInArgo()

	if err != nil {
		klog.Error("failed to get all existing managed cluster secrets for ArgoCD, ", err)
		return reconcile.Result{Requeue: false}, nil
	}

	if !migrationDone {
		argoNamespace := ""
		if len(managedClusterSecretsInArgo.Items) > 0 {
			// If migration is required, use this namespace as target ArgoCD namespace because we supported only one instance of
			// ArgoCD server previously
			argoNamespace = managedClusterSecretsInArgo.Items[0].Namespace
		}

		err := r.MigrateManagedClusterSecrets(argoNamespace)

		if err != nil {
			klog.Info("failed to migrate managed cluster secrets, err: ", err)
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(2) * time.Minute}, err
		}
	} else {
		// If there is no managed cluster secret in any ArgoCD namespace, migration is not necessary
		migrationDone = true
	}

	// Then save it in a map. As we create/update GitOps managed cluster secrets while
	// reconciling each GitOpsCluster resource, remove the secret from this list.
	// After reconciling all GitOpsCluster resources, the secrets left in this list are
	// orphan secrets to be removed.
	orphanGitOpsClusterSecretList := map[types.NamespacedName]string{}

	for _, secret := range managedClusterSecretsInArgo.Items {
		orphanGitOpsClusterSecretList[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}] = secret.Namespace + "/" + secret.Name
	}

	// Get all GitOpsCluster resources
	gitOpsClusters, err := r.GetAllGitOpsClusters()

	if err != nil {
		return reconcile.Result{Requeue: false}, nil
	}

	// For any watched resource change, process all GitOpsCluster CRs to create new secrets or update existing secrets.
	for _, gitOpsCluster := range gitOpsClusters.Items {
		klog.Info("Process GitOpsCluster: " + gitOpsCluster.Namespace + "/" + gitOpsCluster.Name)

		instance := &gitopsclusterV1alpha1.GitOpsCluster{}

		err := r.Get(context.TODO(), types.NamespacedName{Name: gitOpsCluster.Name, Namespace: gitOpsCluster.Namespace}, instance)

		if err != nil && k8errors.IsNotFound(err) {
			klog.Infof("GitOpsCluster %s/%s deleted", gitOpsCluster.Namespace, gitOpsCluster.Name)
			// deleted? just skip to the next GitOpsCluster resource
			continue
		}

		// reconcile one GitOpsCluster resource
		requeueInterval, err := r.reconcileGitOpsCluster(*instance, orphanGitOpsClusterSecretList)

		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(requeueInterval) * time.Minute}, err
		}
	}

	// Remove all invalid/orphan GitOps cluster secrets
	if !r.cleanupOrphanSecrets(orphanGitOpsClusterSecretList) {
		// If it failed to delete orphan GitOps managed cluster secrets, reconile again in 10 minutes.
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(10) * time.Minute}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGitOpsCluster) cleanupOrphanSecrets(orphanGitOpsClusterSecretList map[types.NamespacedName]string) bool {
	cleanupSuccessful := true

	// 4. Delete all orphan GitOps managed cluster secrets
	for key, secretName := range orphanGitOpsClusterSecretList {
		secretToDelete := &v1.Secret{}

		err := r.Get(context.TODO(), key, secretToDelete)

		if err == nil {
			klog.Infof("Deleting orphan GitOps managed cluster secret %s", secretName)

			err = r.Delete(context.TODO(), secretToDelete)

			if err != nil {
				klog.Errorf("failed to delete orphan managed cluster secret %s, err: %s", key, err.Error())

				cleanupSuccessful = false

				continue
			}
		}
	}

	return cleanupSuccessful
}

func (r *ReconcileGitOpsCluster) reconcileGitOpsCluster(
	gitOpsCluster gitopsclusterV1alpha1.GitOpsCluster,
	orphanSecretsList map[types.NamespacedName]string) (int, error) {
	instance := gitOpsCluster.DeepCopy()

	// 1. Verify that spec.argoServer.argoNamespace is a valid ArgoCD namespace
	if !r.VerifyArgocdNamespace(gitOpsCluster.Spec.ArgoServer.ArgoNamespace) {
		klog.Info("invalid argocd namespace because argo server pod was not found")

		instance.Status.LastUpdateTime = metav1.Now()
		instance.Status.Phase = "failed"
		instance.Status.Message = "invalid gitops namespace because argo server pod was not found"

		err := r.Client.Status().Update(context.TODO(), instance)

		if err != nil {
			klog.Errorf("failed to update GitOpsCluster %s status, will try again in 3 minutes: %s", instance.Namespace+"/"+instance.Name, err)
			return 3, err
		}

		return 0, nil
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

		err := r.Client.Status().Update(context.TODO(), instance)

		if err != nil {
			klog.Errorf("failed to update GitOpsCluster %s status, will try again in 3 minutes: %s", instance.Namespace+"/"+instance.Name, err)
			return 3, err
		}
	}

	klog.Infof("adding managed clusters %v into argo namespace %s", managedClusters, instance.Spec.ArgoServer.ArgoNamespace)

	// 3. Copy secret contents from the managed cluster namespaces and create the secret in spec.argoServer.argoNamespace
	err = r.AddManagedClustersToArgo(instance.Spec.ArgoServer.ArgoNamespace, managedClusters, orphanSecretsList)

	if err != nil {
		klog.Info("failed to add managed clusters to argo")

		instance.Status.LastUpdateTime = metav1.Now()
		instance.Status.Phase = "failed"
		instance.Status.Message = err.Error()

		err := r.Client.Status().Update(context.TODO(), instance)

		if err != nil {
			klog.Errorf("failed to update GitOpsCluster %s status, will try again in 3 minutes: %s", instance.Namespace+"/"+instance.Name, err)
			return 3, err
		}

		// it passed all vaidations but simply failed to create or update secrets. Reconile again in 1 minute.
		return 1, err
	}

	instance.Status.LastUpdateTime = metav1.Now()
	instance.Status.Phase = "successful"
	instance.Status.Message = fmt.Sprintf("Added managed clusters %v to gitops namespace %s", managedClusters, instance.Spec.ArgoServer.ArgoNamespace)

	err = r.Client.Status().Update(context.TODO(), instance)

	if err != nil {
		klog.Errorf("failed to update GitOpsCluster %s status, will try again in 3 minutes: %s", instance.Namespace+"/"+instance.Name, err)
		return 3, err
	}

	return 0, nil
}

// GetAllManagedClusterSecretsInArgo returns list of secrets from all GitOps managed cluster
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

// GetAllGitOpsClusters returns all GitOpsCluster CRs
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

// VerifyArgocdNamespace verifies that the given argoNamespace is a valid namspace by verifying that ArgoCD is actually
// installed in that namespace
func (r *ReconcileGitOpsCluster) VerifyArgocdNamespace(argoNamespace string) bool {
	// Check if argo server pod exists in the namespace. This is the community ArgoCD installation.
	isArgoCDNamespace := r.FindPodsWithLabelsAndNamespace(argoNamespace, map[string]string{"app.kubernetes.io/name": "argocd-server"})

	// Check if opernshift gitops argo server pod exists in the namespace. This is the RH OpenShift GitOps operator installation.
	isGitOpsNamespace := r.FindPodsWithLabelsAndNamespace(argoNamespace, map[string]string{"app.kubernetes.io/name": "openshift-gitops-server"})

	return isArgoCDNamespace || isGitOpsNamespace
}

// FindPodsWithLabelsAndNamespace finds a list of pods with provided labels from the specified namespace
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
		klog.Infof("No pod with labels %v found", labels)
		return false
	}

	for _, pod := range podList.Items {
		klog.Info("Found pod ", pod.GetName(), " in namespace ", pod.GetNamespace())
	}

	return true
}

// GetManagedClusters retrieves managed cluster names from placement decision
func (r *ReconcileGitOpsCluster) GetManagedClusters(placementref v1.ObjectReference) ([]string, error) {
	if placementref.Kind != "Placement" ||
		placementref.APIVersion != "cluster.open-cluster-management.io/v1alpha1" {
		return nil, errInvalidPlacementRef
	}

	placement := &clusterv1alpha1.Placement{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: placementref.Namespace, Name: placementref.Name}, placement)

	if err != nil {
		klog.Error("failed to get placement. err: ", err.Error())
		return nil, err
	}

	klog.Infof("looking for placement decisions for placement %s", placementref.Name)

	placementDecisions := &clusterv1alpha1.PlacementDecisionList{}

	listopts := &client.ListOptions{}

	secretSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"cluster.open-cluster-management.io/placement": placementref.Name,
		},
	}

	placementDecisionSelectionLabel, err := utils.ConvertLabels(secretSelector)
	if err != nil {
		klog.Error("Failed to convert placement decision selector, err:", err)
		return nil, err
	}

	listopts.LabelSelector = placementDecisionSelectionLabel
	err = r.List(context.TODO(), placementDecisions, listopts)

	if err != nil {
		klog.Error("Failed to list placement decisions, err:", err)
		return nil, err
	}

	if len(placementDecisions.Items) < 1 {
		klog.Info("no placement decision found for placement: " + placementref.Name)
		return nil, errors.New("no placement decision found for placement: " + placementref.Name)
	}

	clusters := make([]string, 0)

	for _, placementdecision := range placementDecisions.Items {
		klog.Info("getting cluster names from placement decision " + placementdecision.Name)

		for _, clusterDecision := range placementdecision.Status.Decisions {
			klog.Info("cluster name: " + clusterDecision.ClusterName)
			clusters = append(clusters, clusterDecision.ClusterName)
		}
	}

	return clusters, nil
}

// AddManagedClustersToArgo copies a managed cluster secret from the managed cluster namespace to ArgoCD namespace
func (r *ReconcileGitOpsCluster) AddManagedClustersToArgo(
	argoNamespace string,
	managedClusters []string,
	orphanSecretsList map[types.NamespacedName]string) error {
	for _, managedCluster := range managedClusters {
		klog.Infof("adding managed cluster %s to gitops namespace %s", managedCluster, argoNamespace)

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

		// Since thie secret is now added to Argo, it is not orphan.
		argoManagedClusterSecretKey := types.NamespacedName{Name: secretName, Namespace: argoNamespace}
		delete(orphanSecretsList, argoManagedClusterSecretKey)
	}

	return nil
}

// CreateManagedClusterSecretInArgo creates a managed cluster secret with specific metadata in Argo namespace
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

func (r *ReconcileGitOpsCluster) MigrateManagedClusterSecrets(argoNamespace string) error {
	klog.Info("trying to migrate managed cluster secrets")

	allManagedClusters := &spokeClusterV1.ManagedClusterList{}

	err := r.List(context.TODO(), allManagedClusters)

	if err != nil {
		klog.Errorf("failed to migrate managed cluster secrets, err: %v", err.Error())
		return err
	}

	clustersToMigrate := make([]string, 0)

	// Find managed clusters with ArgoCDCluster enabled but do not disable it yet
	for _, managedCluster := range allManagedClusters.Items {
		klog.Infof("checking for migrating secret for managed cluster: %s ", managedCluster.Name)
		wasEnabled, err := r.CheckAndDisableInKlusterletAddonConfig(managedCluster.Name, true)

		if err == nil {
			if wasEnabled {
				klog.Infof("klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster is enabled for managed cluster %s", managedCluster.Name)
				clustersToMigrate = append(clustersToMigrate, managedCluster.Name)
			}
		} else {
			return err
		}
	}

	// Do the migration only if there is at least one managed cluster with ArgoCDCluster enabled
	if len(clustersToMigrate) > 0 {
		// Create a placement rule with those managed clusters
		klog.Infof("migrating secret for managed cluster: %v, ", clustersToMigrate)

		// Create a placement rule for migration
		err = r.CreateMigrationPlacement(clustersToMigrate)

		if err != nil {
			return err
		}

		klog.Info("placement rule CR was successfully created for migration")

		// Now create GitOpsCluster CR with the placement rule in POD_NAMESPACE
		err = r.CreateMigrationGitOpsCluster(argoNamespace, clustersToMigrate)

		if err != nil {
			return err
		}

		klog.Info("gitOpsCluster CR was successfully created for migration")

		// After the GitOpsCluster CR is successfully created, disable ArgoCDCluster in klusterletAddonConfig
		for _, cluster := range clustersToMigrate {
			klog.Infof("disabling ArgoCDCluster in klusterletAddonConfig for managed cluster: %s ", cluster)
			_, err := r.CheckAndDisableInKlusterletAddonConfig(cluster, false)

			if err == nil {
				klog.Infof("klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster is now disabled for managed cluster %s", cluster)
			} else {
				return err
			}
		}

		klog.Info("argocdcluster was successfully disabled in managed clusters")
	} else {
		klog.Infof("no managed cluster secret to migrate")
	}

	migrationDone = true

	klog.Info("argocdcluster migration was done successfully")

	return nil
}

func (r *ReconcileGitOpsCluster) CheckAndDisableInKlusterletAddonConfig(clusterName string, checkOnly bool) (bool, error) {
	klusterletAddonConfigKey := types.NamespacedName{Name: clusterName, Namespace: clusterName}

	klusterletAddonConfig := &agentv1.KlusterletAddonConfig{}

	err := r.Get(context.TODO(), klusterletAddonConfigKey, klusterletAddonConfig)

	if err != nil {
		return false, err
	}

	if klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster {
		if !checkOnly {
			klog.Info("disabling klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster")

			klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster = false
			err = r.Update(context.TODO(), klusterletAddonConfig)

			if err != nil {
				time.Sleep(time.Second * 2)

				err = r.Update(context.TODO(), klusterletAddonConfig)

				if err != nil {
					klog.Errorf("failed to disable ApplicationManagerConfig in klusterletAddonConfig %s, error: %v", klusterletAddonConfigKey, err.Error())
					return false, err
				}

				klog.Info("successfully disabled klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster")

				return true, nil
			}

			klog.Info("successfully disabled klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster")

			return true, nil
		}

		return true, nil
	}

	return false, nil
}

func (r *ReconcileGitOpsCluster) CreateMigrationPlacement(clusterNames []string) error {
	namespace := os.Getenv("POD_NAMESPACE")

	if namespace == "" {
		return errors.New("environment variable POD_NAMESPACE is empty")
	}

	migrationPlacement := &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-cluster-secret-migration",
			Namespace: namespace,
		},
		Spec: clusterv1alpha1.PlacementSpec{
			Predicates: []clusterv1alpha1.ClusterPredicate{
				{
					RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
						ClaimSelector: clusterv1alpha1.ClusterClaimSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "id.k8s.io",
									Operator: "In",
									Values:   clusterNames,
								},
							},
						},
					},
				},
			},
		},
	}

	err := r.Create(context.TODO(), migrationPlacement)

	if err != nil {
		klog.Error("failed to create migration placement CR. will try again. ", err.Error())

		time.Sleep(time.Second * 2)

		err := r.Create(context.TODO(), migrationPlacement)

		if err != nil {
			klog.Error("failed to create migration placement CR again. ", err.Error())
			return err
		}
	}

	return nil
}

func (r *ReconcileGitOpsCluster) CreateMigrationGitOpsCluster(argoNamespace string, clusterNames []string) error {
	namespace := os.Getenv("POD_NAMESPACE")

	if namespace == "" {
		return errors.New("environment variable POD_NAMESPACE is empty")
	}

	newGitOpsClusterCR := &gitopsclusterV1alpha1.GitOpsCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-cluster-secret-migration",
			Namespace: namespace,
		},
		Spec: gitopsclusterV1alpha1.GitOpsClusterSpec{
			ArgoServer: gitopsclusterV1alpha1.ArgoServerSpec{
				Cluster:       "local-cluster",
				ArgoNamespace: argoNamespace,
			},
			PlacementRef: &v1.ObjectReference{
				Kind:       "Placement",
				APIVersion: "cluster.open-cluster-management.io/v1alpha1",
				Namespace:  namespace,
				Name:       "managed-cluster-secret-migration",
			},
		},
	}

	err := r.Create(context.TODO(), newGitOpsClusterCR)

	if err != nil && !k8errors.IsAlreadyExists(err) {
		if k8errors.IsAlreadyExists(err) {
			klog.Info("managed-cluster-secret-migration GitOpsCluster CR already exists. Skip migration.")
		} else {
			klog.Error("failed to create migration GitOpsCluster CR. will try again. ", err.Error())

			time.Sleep(time.Second * 2)

			err := r.Create(context.TODO(), newGitOpsClusterCR)

			if err != nil {
				klog.Error("failed to create migration GitOpsCluster CR again. ", err.Error())
				return err
			}
		}
	}

	return nil
}
