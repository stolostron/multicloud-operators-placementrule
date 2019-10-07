// IBM Confidential
// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

package v1alpha1

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cfg *rest.Config
var c client.Client

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "..", "deploy", "crds")},
	}

	err := SchemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		log.Fatal(err)
	}

	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	if c, err = client.New(cfg, client.Options{Scheme: scheme.Scheme}); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	t.Stop()
	os.Exit(code)
}
