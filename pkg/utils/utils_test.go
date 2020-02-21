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
	"context"
	"testing"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	appv1alpha1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/app/v1alpha1"
)

func TestLocal(t *testing.T) {
	if ToPlaceLocal(nil) {
		t.Error("Failed to check local placement for nil placement")
	}

	pl := &appv1alpha1.Placement{}
	if ToPlaceLocal(pl) {
		t.Error("Failed to check local placement for nil local")
	}

	l := false
	pl.Local = &l

	if ToPlaceLocal(pl) {
		t.Error("Failed to check local placement for false local")
	}

	l = true

	if !ToPlaceLocal(pl) {
		t.Error("Failed to check local placement for true local")
	}
}

func TestLoadCRD(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(CheckAndInstallCRD(cfg, "../../deploy/crds/app.ibm.com_placementrules_crd.yaml")).NotTo(gomega.HaveOccurred())
	g.Expect(CheckAndInstallCRD(cfg, "../../deploy/crds/multicloud-apps.io_placementrules_crd.yaml")).NotTo(gomega.HaveOccurred())
}

func TestEventRecorder(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	rec, err := NewEventRecorder(cfg, scheme.Scheme)
	cfgmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
	g.Expect(c.Create(context.TODO(), cfgmap)).NotTo(gomega.HaveOccurred())

	defer c.Delete(context.TODO(), cfgmap)

	rec.RecordEvent(cfgmap, "no reason", "no message", err)
}
