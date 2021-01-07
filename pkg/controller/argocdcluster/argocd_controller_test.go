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
	"testing"
	"time"

	agentv1 "github.com/open-cluster-management/endpoint-operator/pkg/apis/agent/v1"
	"github.com/open-cluster-management/multicloud-operators-placementrule/pkg/utils"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

const (
	timeout = time.Second * 5
)

var (
	cluster1Namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster1",
		},
	}

	argocdServerNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "argocd",
		},
	}

	newArgocdServerNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-argocd",
		},
	}

	argocdServerPodKey = types.NamespacedName{
		Name:      "argocd-server",
		Namespace: "argocd",
	}

	argocdServerRequestKey = types.NamespacedName{
		Name:      "argocd-server--batch-sync-flag",
		Namespace: "argocd",
	}

	argocdServerExpectedRequest = reconcile.Request{NamespacedName: argocdServerRequestKey}

	newArgocdServerRequestKey = types.NamespacedName{
		Name:      "argocd-server--batch-sync-flag",
		Namespace: "new-argocd",
	}

	newArgocdServerExpectedRequest = reconcile.Request{NamespacedName: newArgocdServerRequestKey}

	argocdServerPod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-server",
			Namespace: "argocd",
			Labels: map[string]string{
				"app.kubernetes.io/name": "argocd-server",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	argocdSecret1Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: "argocd",
	}

	acmSecret1Key = types.NamespacedName{
		Name:      "cluster1-cluster-secret",
		Namespace: "cluster1",
	}

	acmSecret1 = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1-cluster-secret",
			Namespace: "cluster1",
			Labels: map[string]string{
				"apps.open-cluster-management.io/secret-type": "acm-cluster",
			},
		},
		StringData: map[string]string{
			"name":   "cluster1",
			"server": "https://api.cluster1.dev06.red-chesterfield.com:6443",
			"config": "test-bearer-token-1",
		},
	}

	acmSecret1ExpectedRequest = reconcile.Request{NamespacedName: acmSecret1Key}

	klusterletAddonConfig = &agentv1.KlusterletAddonConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1",
			Namespace: "cluster1",
		},
		Spec: agentv1.KlusterletAddonConfigSpec{
			ApplicationManagerConfig: agentv1.KlusterletAddonConfigApplicationManagerSpec{
				Enabled:       true,
				ArgoCDCluster: true,
			},
			ClusterLabels: map[string]string{
				"cloud": "AWS",
			},
			ClusterName:      "cluster1",
			ClusterNamespace: "cluster1",
		},
	}
)

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create argocd server namespace
	g.Expect(c.Create(context.TODO(), argocdServerNamespace)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), argocdServerNamespace)

	// Create argocd server Pod in the argocd namespace
	existingArgocdServerPod := argocdServerPod.DeepCopy()

	g.Expect(c.Create(context.TODO(), existingArgocdServerPod)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), existingArgocdServerPod)

	// Create the cluster namespace.
	g.Expect(c.Create(context.TODO(), cluster1Namespace)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), cluster1Namespace)

	// Create the ACM cluster secret object and expect the Reconcile
	err = c.Create(context.TODO(), acmSecret1)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), acmSecret1)

	// test1: the argocdcluster controller is reconciled
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(acmSecret1ExpectedRequest)))

	// test2: enable KlusterletAddonConfig argocdCluter = true, the argocd cluster secret is created
	g.Expect(c.Create(context.TODO(), klusterletAddonConfig)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), klusterletAddonConfig)

	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(acmSecret1ExpectedRequest)))

	time.Sleep(1 * time.Second)

	argocdSecretlist := &corev1.SecretList{}
	listopts := &client.ListOptions{Namespace: "argocd"}

	argocdSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"argocd.argoproj.io/secret-type":              "cluster",
			"apps.open-cluster-management.io/acm-cluster": "true",
		},
	}

	argocdLabel, _ := utils.ConvertLabels(argocdSelector)

	listopts.LabelSelector = argocdLabel
	err = c.List(context.TODO(), argocdSecretlist, listopts)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(len(argocdSecretlist.Items)).To(gomega.Equal(1))

	// test3: delete the argocd cluster secret, see it is back
	argocdSecret1 := &corev1.Secret{}
	err = c.Get(context.TODO(), argocdSecret1Key, argocdSecret1)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = c.Delete(context.TODO(), argocdSecret1)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(acmSecret1ExpectedRequest)))
	time.Sleep(1 * time.Second)

	argocdSecretlist = &corev1.SecretList{}
	err = c.List(context.TODO(), argocdSecretlist, listopts)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(len(argocdSecretlist.Items)).To(gomega.Equal(1))

	// test4: delete the argocd server pod, check its argocd cluster secret is removed from the argocd namespace
	existingArgocdServerPod = &corev1.Pod{}
	err = c.Get(context.TODO(), argocdServerPodKey, existingArgocdServerPod)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	err = c.Delete(context.TODO(), existingArgocdServerPod)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	ArgocdServerPod := &corev1.Pod{}

	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)

		err = c.Get(context.TODO(), argocdServerPodKey, ArgocdServerPod)

		if err != nil && errors.IsNotFound(err) {
			// make sure the argocd server pod is gone before checking argocd cluster secrets
			g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(argocdServerExpectedRequest)))

			argocdSecretlist = &corev1.SecretList{}
			err = c.List(context.TODO(), argocdSecretlist, listopts)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(len(argocdSecretlist.Items)).To(gomega.Equal(0))

			break
		}
	}

	// test5: create argocd server pod in the new namespace new-argocd, check its argocd cluster secret is synced to the new namespace
	g.Expect(c.Create(context.TODO(), newArgocdServerNamespace)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), newArgocdServerNamespace)

	newArgocdServerPod := argocdServerPod.DeepCopy()
	newArgocdServerPod.SetNamespace("new-argocd")

	g.Expect(c.Create(context.TODO(), newArgocdServerPod)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), newArgocdServerPod)

	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(newArgocdServerExpectedRequest)))
	time.Sleep(5 * time.Second)

	argocdSecretlist = &corev1.SecretList{}
	newListopts := listopts
	newListopts.Namespace = "new-argocd"
	err = c.List(context.TODO(), argocdSecretlist, newListopts)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(len(argocdSecretlist.Items)).To(gomega.Equal(1))
}
