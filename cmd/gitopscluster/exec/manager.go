// Copyright 2021 The Kubernetes Authors.
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

package exec

import (
	"fmt"
	"os"

	"github.com/IBM/controller-filtered-cache/filteredcache"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	endpointapis "github.com/open-cluster-management/klusterlet-addon-controller/pkg/apis"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/controller"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/rest"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost             = "0.0.0.0"
	metricsPort         int = 8388
	operatorMetricsPort int = 8688
)

// RunManager starts the actual manager
func RunManager() {
	enableLeaderElection := false

	if _, err := rest.InClusterConfig(); err == nil {
		klog.Info("LeaderElection enabled as running in a cluster")

		enableLeaderElection = true
	} else {
		klog.Info("LeaderElection disabled as not running in a cluster")
	}

	// Cache only the managed cluster secrets
	filteredSecretMap := map[schema.GroupVersionKind]filteredcache.Selector{
		v1.SchemeGroupVersion.WithKind("Secret"): {
			LabelSelector: "apps.open-cluster-management.io/cluster-name,argocd.argoproj.io/secret-type==cluster",
		},
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		Port:                    operatorMetricsPort,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "multicloud-operators-gitopscluster-leader.open-cluster-management.io",
		LeaderElectionNamespace: "kube-system",
		NewCache:                filteredcache.NewFilteredCacheBuilder(filteredSecretMap),
	})

	if err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	klog.Info("Registering GitOpsCluster components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	if err := endpointapis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	if err := clusterv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddGitOpsClusterToManager(mgr); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	sig := signals.SetupSignalHandler()

	klog.Info("Detecting ACM cluster API service...")
	utils.DetectClusterRegistry(sig, mgr.GetAPIReader())

	klog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(sig); err != nil {
		klog.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
