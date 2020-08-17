module github.com/open-cluster-management/multicloud-operators-placementrule

go 1.15

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.4
	github.com/onsi/gomega v1.9.0
	github.com/open-cluster-management/api v0.0.0-20200610161514-939cead3902c
	github.com/operator-framework/operator-sdk v0.18.0
	github.com/prometheus/common v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20200421231249-e086a090c8fd
	k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver v0.18.2
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2
)
