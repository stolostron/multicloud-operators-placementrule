module github.com/open-cluster-management/multicloud-operators-placementrule

go 1.16

require (
	github.com/IBM/controller-filtered-cache v0.3.2
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.5
	github.com/onsi/gomega v1.13.0
	github.com/open-cluster-management/api v0.0.0-20210513122330-d76f10481f05
	github.com/open-cluster-management/klusterlet-addon-controller v0.0.0-20210303215539-1d12cebe6f19
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/controller-runtime v0.9.1
)

replace k8s.io/client-go => k8s.io/client-go v0.21.3
