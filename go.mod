module github.com/stolostron/multicloud-operators-placementrule

go 1.15

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-openapi/spec v0.19.4
	github.com/onsi/gomega v1.10.1
	github.com/open-cluster-management/api v0.0.0-20201007180356-41d07eee4294
	github.com/spf13/pflag v1.0.5
	github.com/stolostron/endpoint-operator v1.2.2-0-20220114-0ddd7f9
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	github.com/open-cluster-management/api => open-cluster-management.io/api v0.0.0-20201007180356-41d07eee4294
	k8s.io/client-go => k8s.io/client-go v0.19.3
)
