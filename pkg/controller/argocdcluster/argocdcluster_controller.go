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
	"strings"
	"sync"
	"time"

	agentv1 "github.com/stolostron/endpoint-operator/pkg/apis/agent/v1"
	"github.com/stolostron/multicloud-operators-placementrule/pkg/utils"
	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	batchSyncFlag = "--batch-sync-flag"
)

// ArgocdCluster defines a argocd cluster secret
type ArgocdCluster struct {
	SecretName            string
	ArgocdSecretNamespace string
	ACMSecretNamespace    string
	ClusterConfig         string
	ClusterName           string
}

// ArgocdClusterSecrets defines all argocd cluster secrets , key is ArgoCD remote cluster server URL
type ArgocdClusters map[string]ArgocdCluster

// AddonargocdClusterSettings defines ArgoCDCluster settings for each managed cluster
// key: managed cluster name, value: true/false - ArgoCD cluster is on/off
type AddonargocdClusterSettings map[string]bool

// ArgocdServerNamespace defines ArgoCD Server Namespace. by default it is argocd, but it could be installed in any namespace
var ArgocdServerNamespace = "argocd"

// Add creates a new argocd cluster Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	authCfg := mgr.GetConfig()
	klog.Infof("Host: %v, BearerToken: %v", authCfg.Host, authCfg.BearerToken)
	kubeClient := kubernetes.NewForConfigOrDie(authCfg)

	dsRS := &ReconcileSecret{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		authClient: kubeClient,
		lock:       sync.Mutex{},
	}

	// add ReconcileSecret to manager for starting go routine synchronizer to sync all acm clusters to argocd
	// the daily synchronizer is disable for now.
	// _ = mgr.Add(dsRS)

	return dsRS
}

type argocdSecretRuleMapper struct {
	client.Client
}

func (mapper *argocdSecretRuleMapper) Map(obj handler.MapObject) []reconcile.Request {
	// if ArgoCD Cluster secret is changed, reconcile its relative acm cluster secret
	var requests []reconcile.Request

	acmClusterSecretKey := types.NamespacedName{
		Name:      obj.Meta.GetName(),
		Namespace: utils.GetACMClusterNamespace(obj.Meta.GetName())}

	requests = append(requests, reconcile.Request{NamespacedName: acmClusterSecretKey})

	return requests
}

type argocdAddonConfigRuleMapper struct {
	client.Client
}

func (mapper *argocdAddonConfigRuleMapper) Map(obj handler.MapObject) []reconcile.Request {
	// if ArgoCDCluster addon config is changed, reconcile its relative acm cluster secret
	var requests []reconcile.Request

	acmClusterSecretKey := types.NamespacedName{
		Name:      obj.Meta.GetNamespace() + "-cluster-secret",
		Namespace: obj.Meta.GetNamespace()}

	requests = append(requests, reconcile.Request{NamespacedName: acmClusterSecretKey})

	return requests
}

type argocdServerRuleMapper struct {
	client.Client
}

func (mapper *argocdServerRuleMapper) Map(obj handler.MapObject) []reconcile.Request {
	// if ArgoCDCluster addon config is changed, reconcile its relative acm cluster secret
	var requests []reconcile.Request

	acmClusterSecretKey := types.NamespacedName{
		Name:      obj.Meta.GetName() + batchSyncFlag,
		Namespace: obj.Meta.GetNamespace()}

	requests = append(requests, reconcile.Request{NamespacedName: acmClusterSecretKey})

	return requests
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("argocdcluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if utils.IsReadyACMClusterRegistry(mgr.GetAPIReader()) {
		// Watch for ACM cluster secret changes
		err = c.Watch(
			&source.Kind{Type: &v1.Secret{}}, &handler.EnqueueRequestForObject{}, utils.AcmClusterSecretPredicateFunc,
		)
		if err != nil {
			return err
		}

		// Watch for ArgoCD cluster secret changes
		err = c.Watch(
			&source.Kind{Type: &v1.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &argocdSecretRuleMapper{mgr.GetClient()}},
			utils.ArgocdClusterSecretPredicateFunc)
		if err != nil {
			return err
		}

		// Watch for KlusterletAddonConfig argocdCluster setting changes
		err = c.Watch(
			&source.Kind{Type: &agentv1.KlusterletAddonConfig{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &argocdAddonConfigRuleMapper{mgr.GetClient()}})
		if err != nil {
			return err
		}

		// Watch for argocd server changes
		err = c.Watch(
			&source.Kind{Type: &v1.Service{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &argocdServerRuleMapper{mgr.GetClient()}},
			utils.ArgocdServerPredicateFunc)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSecret{}

// ReconcileSecret reconciles a Secret object
type ReconcileSecret struct {
	client.Client
	authClient kubernetes.Interface
	scheme     *runtime.Scheme
	lock       sync.Mutex
}

// Start the discovery and start caches
func (r *ReconcileSecret) Start(s <-chan struct{}) error {
	go wait.Until(func() {
		// If argocd is installed, batch sync all acm clusters to argocd every 24 hours.
		ArgocdServerNamespace = r.GetArgocdServerNamespace()
		if ArgocdServerNamespace > "" {
			err := r.SyncArgocdClusterSecrets()
			klog.Infof("Finished syncing ACM clusters to ArgoCD. ArgoCD namespace: %v, err: %v", ArgocdServerNamespace, err)
		} else {
			err := r.DeleteAllArgocdClusterSecrets()
			klog.Infof("ArgoCD Server not found, Delete all argocd cluster secrets. ArgoCD Namespace: %v, err: %v", ArgocdServerNamespace, err)
		}
	}, time.Duration(24)*time.Hour, s)

	<-s

	return nil
}

// Reconcile reads that state of the ACM cluster Secret objects and makes changes based on the state read
func (r *ReconcileSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the ACM cluster secret instance
	instance := &v1.Secret{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)

	klog.Info("Reconciling ACM cluster secret:", request.NamespacedName, " with Get err:", err)

	if strings.HasSuffix(request.Name, batchSyncFlag) {
		argocdServerName := strings.TrimSuffix(request.Name, batchSyncFlag)
		ArgocdServerNamespace = request.Namespace

		argocdServerKey := types.NamespacedName{Name: argocdServerName, Namespace: ArgocdServerNamespace}

		argocdServerService := &v1.Service{}
		err := r.Get(context.TODO(), argocdServerKey, argocdServerService)

		if (err != nil && errors.IsNotFound(err)) ||
			(argocdServerService.DeletionTimestamp != nil && argocdServerService.DeletionTimestamp.IsZero()) {
			// no argocd server service found, batch delete all argocd cluster secrets from the argocd server namespace
			err = r.DeleteAllArgocdClusterSecrets()
			klog.Infof("ArgoCD Server not found, Delete all argocd cluster secrets. ArgoCD Namespace: %v, err: %v", ArgocdServerNamespace, err)

			return reconcile.Result{}, err
		}

		err = r.SyncArgocdClusterSecrets()
		klog.Infof("ArgoCD Server namespace updated. Sync all argocd cluster secrets. ArgoCD Namespace: %v, err: %v", ArgocdServerNamespace, err)

		return reconcile.Result{}, err
	}

	ArgocdServerNamespace = r.GetArgocdServerNamespace()
	if ArgocdServerNamespace == "" {
		klog.Infof("No ArgoCD server found: %v", ArgocdServerNamespace)
		return reconcile.Result{}, nil
	}

	klog.Infof("ArgoCD server namespace: %v", ArgocdServerNamespace)

	if !r.IfSyncArgocdCluster(request.Namespace) {
		klog.Infof("The argocd cluster collection for this ACM cluster is disabled, cluster: %v", request.Namespace)

		argocdClusterSecretKey := types.NamespacedName{Name: request.Name, Namespace: ArgocdServerNamespace}
		err = r.DeleteClusterSecret(argocdClusterSecretKey)

		return reconcile.Result{}, err
	}

	if err != nil {
		if errors.IsNotFound(err) {
			// ACM cluster secret is gone, delete its ArgoCD cluster secret
			argocdClusterSecretKey := types.NamespacedName{Name: request.Name, Namespace: ArgocdServerNamespace}

			err = r.DeleteClusterSecret(argocdClusterSecretKey)

			klog.Info("Delete ArgoCD cluster secret:", argocdClusterSecretKey, " with Get err:", err)

			return reconcile.Result{}, err
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// do nothing if has finalizer
	if len(instance.GetObjectMeta().GetFinalizers()) != 0 {
		return reconcile.Result{}, nil
	}

	err = r.hubReconcile(instance)
	if err != nil {
		klog.Infof("ACM cluster secret failed. instance: %v/%v, err: %v", instance.Namespace, instance.Name, err)

		return reconcile.Result{}, err
	}

	klog.V(1).Info("Reconciling - finished.", request.NamespacedName, " with Get err:", err)

	return reconcile.Result{}, nil
}

// GetArgocdServerNamespace get ArgoCD server namespace.
// We assume that only one ArgoCD server is running on ACM hub cluster
func (r *ReconcileSecret) GetArgocdServerNamespace() string {
	ArgocdServerServiceList := &v1.ServiceList{}
	listopts := &client.ListOptions{}

	ArgocdServerSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/component": "server",
			"app.kubernetes.io/part-of":   "argocd",
		},
	}

	ArgocdServerLabel, err := utils.ConvertLabels(ArgocdServerSelector)
	if err != nil {
		klog.Error("Failed to convert ArgoCD Cluster Selector, err:", err)
		return ""
	}

	listopts.LabelSelector = ArgocdServerLabel
	err = r.List(context.TODO(), ArgocdServerServiceList, listopts)

	if err != nil {
		klog.Error("Failed to list ArgoCD cluster secrets, err:", err)
		return ""
	}

	if len(ArgocdServerServiceList.Items) == 0 {
		klog.Error("No ArgoCD server service found")
		return ""
	}

	return ArgocdServerServiceList.Items[0].Namespace
}

func (r *ReconcileSecret) IfSyncArgocdCluster(clusterName string) bool {
	klusterletAddonConfigKey := types.NamespacedName{Name: clusterName, Namespace: clusterName}

	klusterletAddonConfig := &agentv1.KlusterletAddonConfig{}

	err := r.Get(context.TODO(), klusterletAddonConfigKey, klusterletAddonConfig)
	if err == nil {
		if !klusterletAddonConfig.Spec.ApplicationManagerConfig.Enabled {
			return false
		}

		return klusterletAddonConfig.Spec.ApplicationManagerConfig.ArgoCDCluster
	}

	return false
}
