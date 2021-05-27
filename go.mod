module github.com/open-cluster-management/multicloud-operators-placementrule

go 1.16

require (
	github.com/IBM/controller-filtered-cache v0.2.2
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-openapi/spec v0.19.4
	github.com/onsi/gomega v1.10.1
	github.com/open-cluster-management/api v0.0.0-20210513122330-d76f10481f05
	github.com/open-cluster-management/klusterlet-addon-controller v0.0.0-20210303215539-1d12cebe6f19
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	k8s.io/api v0.20.0
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
)
