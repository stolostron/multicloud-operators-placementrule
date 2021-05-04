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
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	gitopsclusterV1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	c client.Client

	// Namespace where GitOpsCluster1 CR is
	testNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}

	testNamespace2 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test2",
		},
	}

	managedClusterNamespace1 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster1",
		},
	}

	managedClusterNamespace2 = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster2",
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

	managedClusterSecret1Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: "cluster1",
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

	gitOpsClusterSecret1 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1-cluster-secret",
			Namespace: "argocd1",
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

	gitOpsCluster11Key = types.NamespacedName{
		Name:      "git-ops-cluster-1",
		Namespace: testNamespace1.Name,
	}

	gitOpsCluster1 = &gitopsclusterV1alpha1.GitOpsCluster{
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
				Namespace:  testNamespace1.Name,
				Name:       "test-placement-1",
			},
		},
	}

	gitOpsCluster2Key = types.NamespacedName{
		Name:      "git-ops-cluster-2",
		Namespace: testNamespace1.Name,
	}

	gitOpsCluster2 = &gitopsclusterV1alpha1.GitOpsCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-ops-cluster-2",
			Namespace: testNamespace1.Name,
		},
		Spec: gitopsclusterV1alpha1.GitOpsClusterSpec{
			ArgoServer: gitopsclusterV1alpha1.ArgoServerSpec{
				Cluster:       "local-cluster",
				ArgoNamespace: "argocd2",
			},
			PlacementRef: &corev1.ObjectReference{
				Kind:       "Placement",
				APIVersion: "cluster.open-cluster-management.io/v1alpha1",
				Namespace:  testNamespace1.Name,
				Name:       "test-placement-2",
			},
		},
	}

	gitOpsCluster3 = &gitopsclusterV1alpha1.GitOpsCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git-ops-cluster-3",
			Namespace: testNamespace1.Name,
		},
		Spec: gitopsclusterV1alpha1.GitOpsClusterSpec{
			ArgoServer: gitopsclusterV1alpha1.ArgoServerSpec{
				Cluster:       "local-cluster",
				ArgoNamespace: gitopsServerNamespace1.Name,
			},
			PlacementRef: &corev1.ObjectReference{
				Kind:       "Placement",
				APIVersion: "cluster.open-cluster-management.io/v1alpha1",
				Namespace:  testNamespace1.Name,
				Name:       "test-placement-2",
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

	recFn, _ := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment

	// Managed cluster namespace
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create Argo namespace
	c.Create(context.TODO(), argocdServerNamespace1)
	g.Expect(c.Create(context.TODO(), argoServerPod.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), argoServerPod)

	// Create GitOpsCluster CR
	c.Create(context.TODO(), testNamespace1)
	g.Expect(c.Create(context.TODO(), gitOpsCluster1.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsCluster1)

	// Test that the managed cluster's secret is created in the Argo namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret1Key)).To(gomega.BeTrue())
}

func TestReconcileNoSecretInInvalidArgoNamespace(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn, _ := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment

	// Create managed cluster namespaces
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create invalid Argo namespaces where there is no argo server pod
	c.Create(context.TODO(), argocdServerNamespace2)

	// Create GitOpsCluster CR
	c.Create(context.TODO(), testNamespace1)
	g.Expect(c.Create(context.TODO(), gitOpsCluster2)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsCluster2)

	// Test that the managed cluster's secret is not created in argocd2
	// namespace because there is no valid argocd server pod in argocd2 namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret2Key)).To(gomega.BeFalse())
}

func TestReconcileCreateSecretInOpenshiftGitops(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn, _ := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment

	// Managed cluster namespace
	c.Create(context.TODO(), managedClusterNamespace1)
	g.Expect(c.Create(context.TODO(), managedClusterSecret1.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), managedClusterSecret1)

	// Create Openshift-gitops namespace
	c.Create(context.TODO(), gitopsServerNamespace1)
	g.Expect(c.Create(context.TODO(), openshiftGitopsServerPod)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), openshiftGitopsServerPod)

	// Create GitOpsCluster CR
	c.Create(context.TODO(), testNamespace1)
	g.Expect(c.Create(context.TODO(), gitOpsCluster3)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsCluster3)

	// Test that the managed cluster's secret is created in the Argo namespace
	g.Expect(expectedSecretCreated(c, gitOpsClusterSecret3Key)).To(gomega.BeTrue())
}

func expectedSecretCreated(c client.Client, expectedSecretKey types.NamespacedName) bool {
	timeout := 0
	for {
		fmt.Println("checking secret " + expectedSecretKey.String())
		secret := &corev1.Secret{}
		err := c.Get(context.TODO(), expectedSecretKey, secret)

		if err == nil {
			fmt.Println("FOUND")
			return true
		}
		fmt.Println("NOT FOUND")

		if timeout > 30 {
			return false
		}

		time.Sleep(time.Second * 3)

		timeout = timeout + 3
	}
}

func TestReconcileDeleteOrphanSecret(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn, _ := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Set up test environment

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
	c.Create(context.TODO(), testNamespace1)
	g.Expect(c.Create(context.TODO(), gitOpsCluster1.DeepCopy())).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsCluster1)

	// Test that the orphan managed cluster's secret is deleted from the Argo namespace
	g.Expect(checkOrphanSecretDeleted(c, gitOpsClusterSecret2Key)).To(gomega.BeTrue())
}

func checkOrphanSecretDeleted(c client.Client, expectedSecretKey types.NamespacedName) bool {
	timeout := 0
	for {
		fmt.Println("checking if orphan secret gone " + expectedSecretKey.String())
		secret := &corev1.Secret{}
		err := c.Get(context.TODO(), expectedSecretKey, secret)

		if err != nil {
			return true
		}

		if timeout > 30 {
			return false
		}

		time.Sleep(time.Second * 3)

		timeout = timeout + 3
	}
}

/*
func TestGetAllManagedClusterSecretsInArgo(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	r := &ReconcileGitOpsCluster{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		authClient: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		lock:       sync.Mutex{}}

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// No managed cluster secret yet
	secretList, err := r.GetAllManagedClusterSecretsInArgo()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(len(secretList.Items)).To(gomega.Equal(0))

	// Create one cluster secret that is not for Argo
	c.Create(context.TODO(), managedClusterNamespace1)
	c.Create(context.TODO(), argocdServerNamespace1)

	//g.Expect(c.Create(context.TODO(), managedClusterNamespace1)).NotTo(gomega.HaveOccurred())
	//defer c.Delete(context.TODO(), managedClusterNamespace1)

	g.Expect(c.Create(context.TODO(), managedClusterSecret1)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), managedClusterSecret1)

	secretList, err = r.GetAllManagedClusterSecretsInArgo()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(len(secretList.Items)).To(gomega.Equal(0))

	// Create one cluster secret in Argo
	//g.Expect(c.Create(context.TODO(), argocdServerNamespace1)).NotTo(gomega.HaveOccurred())
	//defer c.Delete(context.TODO(), argocdServerNamespace1)

	g.Expect(c.Create(context.TODO(), gitOpsClusterSecret1)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsClusterSecret1)

	secretList, err = r.GetAllManagedClusterSecretsInArgo()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(secretList.Items[0].Name).To(gomega.Equal("cluster1-cluster-secret"))

	// Give it some time to finish all deferred deletes
	time.Sleep(10 * time.Second)
}

func TestAddManagedClustersToArgo(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	r := &ReconcileGitOpsCluster{
		Client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		authClient: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		lock:       sync.Mutex{}}

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	cleanupResources()

	orphanGitOpsClusterSecretList := map[types.NamespacedName]string{}

	// Create one cluster secret for Argo
	c.Create(context.TODO(), argocdServerNamespace1)
	c.Create(context.TODO(), managedClusterNamespace1)
	c.Create(context.TODO(), argocdServerNamespace2)

	//g.Expect(c.Create(context.TODO(), argocdServerNamespace1)).NotTo(gomega.HaveOccurred())
	//defer c.Delete(context.TODO(), argocdServerNamespace1)

	//g.Expect(c.Create(context.TODO(), managedClusterNamespace1)).NotTo(gomega.HaveOccurred())
	//defer c.Delete(context.TODO(), managedClusterNamespace1)

	g.Expect(c.Create(context.TODO(), managedClusterSecret1)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), managedClusterSecret1)

	// This secret simulates an orphan GitOps managed cluster secret that needs to be removed.
	//g.Expect(c.Create(context.TODO(), argocdServerNamespace2)).NotTo(gomega.HaveOccurred())
	//defer c.Delete(context.TODO(), argocdServerNamespace2)

	g.Expect(c.Create(context.TODO(), gitOpsClusterSecret2)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), gitOpsClusterSecret2)

	secretList, err := r.GetAllManagedClusterSecretsInArgo()
	g.Expect(err).NotTo(gomega.HaveOccurred())

	for _, secret := range secretList.Items {
		orphanGitOpsClusterSecretList[types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}] = secret.Namespace + "/" + secret.Name
	}

	// Initially the orphan secret list starts with all GitOps managed cluster secrets
	g.Expect(len(orphanGitOpsClusterSecretList)).To(gomega.Equal(1))

	// Test if the managed cluster secret from namespace cluster1 gets copied and created into Argo namespace argocd1
	g.Expect(r.AddManagedClustersToArgo(argocdServerNamespace1.Name, []string{"cluster1"}, orphanGitOpsClusterSecretList)).NotTo(gomega.HaveOccurred())

	// Give it some time to finish creating the secret
	time.Sleep(3 * time.Second)

	expectedGitOpsClusterSecret1 := &corev1.Secret{}
	g.Expect(c.Get(context.TODO(), gitOpsClusterSecret1Key, expectedGitOpsClusterSecret1)).NotTo(gomega.HaveOccurred())

	// Since gitOpsClusterSecret2 is not generated by r.AddManagedClustersToArgo, it should remain in orphanGitOpsClusterSecretList.
	g.Expect(len(orphanGitOpsClusterSecretList)).To(gomega.Equal(1))

	defer c.Delete(context.TODO(), expectedGitOpsClusterSecret1)

	// Test if the managed cluster secret from namespace cluster1 gets copied and created into Argo namespace argocd1
	g.Expect(r.AddManagedClustersToArgo(argocdServerNamespace2.Name, []string{"cluster1"}, orphanGitOpsClusterSecretList)).NotTo(gomega.HaveOccurred())

	// Give it some time to finish creating the secret
	time.Sleep(3 * time.Second)

	expectedGitOpsClusterSecret2 := &corev1.Secret{}
	g.Expect(c.Get(context.TODO(), gitOpsClusterSecret1Key, expectedGitOpsClusterSecret2)).NotTo(gomega.HaveOccurred())

	// Since gitOpsClusterSecret2 is now generated by r.AddManagedClustersToArgo, it should be removed from orphanGitOpsClusterSecretList.
	g.Expect(len(orphanGitOpsClusterSecretList)).To(gomega.Equal(0))

}

func cleanupResources() {
	argocdServerNamespace1.ResourceVersion = ""
	argocdServerNamespace2.ResourceVersion = ""
	managedClusterNamespace1.ResourceVersion = ""
	managedClusterNamespace2.ResourceVersion = ""
	managedClusterSecret1.ResourceVersion = ""
	gitOpsClusterSecret2.ResourceVersion = ""
	gitOpsClusterSecret1.ResourceVersion = ""
}
*/
