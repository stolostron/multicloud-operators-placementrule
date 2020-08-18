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

package exec

import (
	"fmt"
	"os"

	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/controller"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost             = "0.0.0.0"
	metricsPort         int = 8383
	operatorMetricsPort int = 8686
)

// RunManager starts the actual manager
func RunManager() {
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		Port:                    operatorMetricsPort,
		LeaderElection:          true,
		LeaderElectionID:        "multicloud-operators-placementrule-leader.open-cluster-management.io",
		LeaderElectionNamespace: "kube-system",
	})

	if err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	klog.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		klog.Error(err, "")
		os.Exit(1)
	}

	sig := signals.SetupSignalHandler()

	klog.Info("Detecting ACM cluster API service...")
	utils.DetectClusterRegistry(mgr.GetAPIReader(), sig)

	klog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(sig); err != nil {
		klog.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
