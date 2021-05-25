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

package gitopscluster

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	gitopsclusterV1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	c client.Client

	// Test1 resources
	test1Ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}

	test1Pl = &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-1",
			Namespace: test1Ns.Name,
		},
		Spec: clusterv1alpha1.PlacementSpec{},
	}

	test1PlDc = &clusterv1alpha1.PlacementDecision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-decision-1",
			Namespace: test1Ns.Name,
			Labels: map[string]string{
				"cluster.open-cluster-management.io/placement": "test-placement-1",
			},
		},
	}

	placementDecisionStatus = &clusterv1alpha1.PlacementDecisionStatus{
		Decisions: []clusterv1alpha1.ClusterDecision{
			*clusterDecision1,
		},
	}

	clusterDecision1 = &clusterv1alpha1.ClusterDecision{
		ClusterName: "cluster1",
		Reason:      "OK",
	}

	// Test2 resources
	test2Ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test2",
		},
	}

	test2Pl = &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-2",
			Namespace: test2Ns.Name,
		},
		Spec: clusterv1alpha1.PlacementSpec{},
	}

	test2PlDc = &clusterv1alpha1.PlacementDecision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-decision-2",
			Namespace: test2Ns.Name,
			Labels: map[string]string{
				"cluster.open-cluster-management.io/placement": "test-placement-2",
			},
		},
	}

	// Test3 resources
	test3Ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test3",
		},
	}

	test3Pl = &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-3",
			Namespace: test3Ns.Name,
		},
		Spec: clusterv1alpha1.PlacementSpec{},
	}

	test3PlDc = &clusterv1alpha1.PlacementDecision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-decision-3",
			Namespace: test3Ns.Name,
			Labels: map[string]string{
				"cluster.open-cluster-management.io/placement": "test-placement-3",
			},
		},
	}

	// Test4 resources
	test4Ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test4",
		},
	}

	test4Pl = &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-4",
			Namespace: test4Ns.Name,
		},
		Spec: clusterv1alpha1.PlacementSpec{},
	}

	test4PlDc = &clusterv1alpha1.PlacementDecision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-placement-decision-4",
			Namespace: test4Ns.Name,
			Labels: map[string]string{
				"cluster.open-cluster-management.io/placement": "test-placement-4",
			},
		},
	}

	// Namespace where GitOpsCluster1 CR is
	testNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}

	managedClusterNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster1",
		},
	}

	argocdServerNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd1",
		},
	}

	argocdServerNamespace2 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd2",
		},
	}

	gitopsServerNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-gitops1",
		},
	}

	managedClusterSecret1 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1-cluster-secret",
			Namespace: "cluster1",
			Labels: map[string]string{
				"apps.open-cluster-management.io/secret-type": "acm-cluster",
			},
		},
		StringData: map[string]string{
			"name":   "cluster1",
			"server": "https://api.cluster1.com:6443",
			"config": "test-bearer-token-1",
		},
	}

	gitOpsClusterSecret1Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: "argocd1",
	}

	gitOpsClusterSecret2Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: "argocd2",
	}

	gitOpsClusterSecret2 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1-cluster-secret",
			Namespace: "argocd2",
			Labels: map[string]string{
				"apps.open-cluster-management.io/acm-cluster": "true",
				"argocd.argoproj.io/secret-type":              "cluster",
			},
		},
		StringData: map[string]string{
			"name":   "cluster1",
			"server": "https://api.cluster1.com:6443",
			"config": "test-bearer-token-1",
		},
	}

	gitOpsCluster = &gitopsclusterV1alpha1.GitOpsCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-ops-cluster-1",
			Namespace: testNamespace1.Name,
		},
		Spec: gitopsclusterV1alpha1.GitOpsClusterSpec{
			ArgoServer: gitopsclusterV1alpha1.ArgoServerSpec{
				Cluster:       "local-cluster",
				ArgoNamespace: "argocd1",
			},
			PlacementRef: &corev1.ObjectReference{
				Kind:       "Placement",
				APIVersion: "cluster.open-cluster-management.io/v1alpha1",
				Namespace:  test1Ns.Name,
				Name:       test1Pl.Name,
			},
		},
	}

	gitOpsClusterSecret3Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: gitopsServerNamespace1.Name,
	}

	argoServerPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argo-server",
			Namespace: "argocd1",
			Labels: map[string]string{
				"app.kubernetes.io/name": "argocd-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   "busybox",
					Command: []string{"sh", "-c", "echo \"Fake argo server\" && sleep 3600"},
				},
			},
		},
	}

	openshiftGitopsServerPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitops-server",
			Namespace: "openshift-gitops1",
			Labels: map[string]string{
				"app.kubernetes.io/name": "openshift-gitops-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   "busybox",
					Command: []string{"sh", "-c", "echo \"Fake argo server\" && sleep 3600"},
				},
			},
		},
	}
)

func TestReconcileCreateSecretInArgo(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment
	c.Create(context.TODO(), test1Ns)

	// Create placement
	g.Expect(c.Create(context.TODO(), test1Pl.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), test1Pl)

	// Create placement decision
	g.Expect(c.Create(context.TODO(), test1PlDc.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), test1PlDc)

	// Update placement decision status
	placementDecision1 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: test1PlDc.Namespace, Name: test1PlDc.Name},
		placementDecision1)).NotTo(gomega.HaveOccurred())

	newPlacementDecision1 := placementDecision1.DeepCopy()
	newPlacementDecision1.Status = *placementDecisionStatus

	g.Expect(c.Status().Update(context.TODO(), newPlacementDecision1)).NotTo(gomega.HaveOccurred())

	time.Sleep(time.Second * 3)

	placementDecisionAfterupdate := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: placementDecision1.Namespace, Name: placementDecision1.Name},
		placementDecisionAfterupdate)).NotTo(gomega.HaveOccurred())

	g.Expect(placementDecisionAfterupdate.Status.Decisions[0].ClusterName).To(gomega.Equal("cluster1"))

	// Managed cluster namespace
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create Argo namespace and fake argo server pod
	c.Create(context.TODO(), argocdServerNamespace1)
	g.Expect(c.Create(context.TODO(), argoServerPod.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), argoServerPod)

	// Create GitOpsCluster CR
	goc := gitOpsCluster.DeepCopy()
	goc.Spec.PlacementRef = &corev1.ObjectReference{
		Kind:       "Placement",
		APIVersion: "cluster.open-cluster-management.io/v1alpha1",
		Namespace:  test1Ns.Name,
		Name:       test1Pl.Name,
	}
	g.Expect(c.Create(context.TODO(), goc)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), goc)

	// Test that the managed cluster's secret is created in the Argo namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret1Key)).To(gomega.BeTrue())
}

func TestReconcileNoSecretInInvalidArgoNamespace(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment
	c.Create(context.TODO(), test2Ns)

	// Create placement
	g.Expect(c.Create(context.TODO(), test2Pl.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), test2Pl)

	// Create placement decision
	g.Expect(c.Create(context.TODO(), test2PlDc.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), test2PlDc)

	// Update placement decision status
	placementDecision2 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: test2PlDc.Namespace, Name: test2PlDc.Name},
		placementDecision2)).NotTo(gomega.HaveOccurred())

	newPlacementDecision2 := placementDecision2.DeepCopy()
	newPlacementDecision2.Status = *placementDecisionStatus

	g.Expect(c.Status().Update(context.TODO(), newPlacementDecision2)).NotTo(gomega.HaveOccurred())

	time.Sleep(time.Second * 3)

	placementDecisionAfterupdate2 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: placementDecision2.Namespace, Name: placementDecision2.Name},
		placementDecisionAfterupdate2)).NotTo(gomega.HaveOccurred())

	g.Expect(placementDecisionAfterupdate2.Status.Decisions[0].ClusterName).To(gomega.Equal("cluster1"))

	// Create managed cluster namespaces
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create invalid Argo namespaces where there is no argo server pod
	c.Create(context.TODO(), argocdServerNamespace2)

	// Create GitOpsCluster CR
	goc := gitOpsCluster.DeepCopy()
	goc.Namespace = test2Ns.Name
	goc.Spec.PlacementRef = &corev1.ObjectReference{
		Kind:       "Placement",
		APIVersion: "cluster.open-cluster-management.io/v1alpha1",
		Namespace:  test2Ns.Name,
		Name:       test2Pl.Name,
	}
	g.Expect(c.Create(context.TODO(), goc)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), goc)

	// Test that the managed cluster's secret is not created in argocd2
	// namespace because there is no valid argocd server pod in argocd2 namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret2Key)).To(gomega.BeFalse())
}

func TestReconcileCreateSecretInOpenshiftGitops(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment
	c.Create(context.TODO(), test3Ns)

	// Create placement
	g.Expect(c.Create(context.TODO(), test3Pl.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), test3Pl)

	// Create placement decision
	g.Expect(c.Create(context.TODO(), test3PlDc.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), test3PlDc)

	// Update placement decision status
	placementDecision3 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: test3PlDc.Namespace, Name: test3PlDc.Name},
		placementDecision3)).NotTo(gomega.HaveOccurred())

	newPlacementDecision3 := placementDecision3.DeepCopy()
	newPlacementDecision3.Status = *placementDecisionStatus

	g.Expect(c.Status().Update(context.TODO(), newPlacementDecision3)).NotTo(gomega.HaveOccurred())

	time.Sleep(time.Second * 3)

	placementDecisionAfterupdate3 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: placementDecision3.Namespace, Name: placementDecision3.Name},
		placementDecisionAfterupdate3)).NotTo(gomega.HaveOccurred())

	g.Expect(placementDecisionAfterupdate3.Status.Decisions[0].ClusterName).To(gomega.Equal("cluster1"))

	// Managed cluster namespace
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create Openshift-gitops namespace
	c.Create(context.TODO(), gitopsServerNamespace1)
	g.Expect(c.Create(context.TODO(), openshiftGitopsServerPod)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), openshiftGitopsServerPod)

	// Create GitOpsCluster CR
	goc := gitOpsCluster.DeepCopy()
	goc.Namespace = test3Ns.Name
	goc.Spec.ArgoServer.ArgoNamespace = gitopsServerNamespace1.Name
	goc.Spec.PlacementRef = &corev1.ObjectReference{
		Kind:       "Placement",
		APIVersion: "cluster.open-cluster-management.io/v1alpha1",
		Namespace:  test3Ns.Name,
		Name:       test3Pl.Name,
	}
	g.Expect(c.Create(context.TODO(), goc)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), goc)

	// Test that the managed cluster's secret is created in the Argo namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret3Key)).To(gomega.BeTrue())
}

func expectedSecretCreated(c client.Client, expectedSecretKey types.NamespacedName) bool {
	timeout := 0

	for {
		secret := &corev1.Secret{}
		err := c.Get(context.TODO(), expectedSecretKey, secret)

		if err == nil {
			return true
		}

		if timeout > 30 {
			return false
		}

		time.Sleep(time.Second * 3)

		timeout += 3
	}
}

func TestReconcileDeleteOrphanSecret(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment
	c.Create(context.TODO(), test4Ns)

	// Create placement
	g.Expect(c.Create(context.TODO(), test4Pl.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), test4Pl)

	// Create placement decision
	g.Expect(c.Create(context.TODO(), test4PlDc.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), test4PlDc)

	// Update placement decision status
	placementDecision4 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: test4PlDc.Namespace, Name: test4PlDc.Name},
		placementDecision4)).NotTo(gomega.HaveOccurred())

	newPlacementDecision4 := placementDecision4.DeepCopy()
	newPlacementDecision4.Status = *placementDecisionStatus

	g.Expect(c.Status().Update(context.TODO(), newPlacementDecision4)).NotTo(gomega.HaveOccurred())

	time.Sleep(time.Second * 3)

	placementDecisionAfterupdate4 := &clusterv1alpha1.PlacementDecision{}
	g.Expect(c.Get(context.TODO(),
		types.NamespacedName{Namespace: placementDecision4.Namespace,
			Name: placementDecision4.Name}, placementDecisionAfterupdate4)).NotTo(gomega.HaveOccurred())

	g.Expect(placementDecisionAfterupdate4.Status.Decisions[0].ClusterName).To(gomega.Equal("cluster1"))

	// Managed cluster namespace
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create Argo namespace
	c.Create(context.TODO(), argocdServerNamespace1)
	g.Expect(c.Create(context.TODO(), argoServerPod.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), argoServerPod)

	// Create invalid Argo namespaces where there is no argo server pod
	// And create a cluster secret to simulate an orphan cluster secret
	c.Create(context.TODO(), argocdServerNamespace2)
	g.Expect(c.Create(context.TODO(), gitOpsClusterSecret2.DeepCopy())).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), gitOpsClusterSecret2)

	// Create GitOpsCluster CR
	goc := gitOpsCluster.DeepCopy()
	goc.Namespace = test4Ns.Name
	goc.Spec.PlacementRef = &corev1.ObjectReference{
		Kind:       "Placement",
		APIVersion: "cluster.open-cluster-management.io/v1alpha1",
		Namespace:  test4Ns.Name,
		Name:       test4Pl.Name,
	}
	g.Expect(c.Create(context.TODO(), goc)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), goc)

	// Test that the orphan managed cluster's secret is deleted from the Argo namespace
	g.Expect(checkOrphanSecretDeleted(c, gitOpsClusterSecret2Key)).To(gomega.BeTrue())
}

func checkOrphanSecretDeleted(c client.Client, expectedSecretKey types.NamespacedName) bool {
	timeout := 0

	for {
		secret := &corev1.Secret{}
		err := c.Get(context.TODO(), expectedSecretKey, secret)

		if err != nil {
			return true
		}

		if timeout > 30 {
			return false
		}

		time.Sleep(time.Second * 3)

		timeout += 3
	}
}
