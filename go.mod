module github.com/open-cluster-management/multicloud-operators-placementrule

go 1.15

require (
	cloud.google.com/go v0.49.0 // indirect
	github.com/Azure/go-autorest/autorest v0.9.3-0.20191028180845-3492b2aff503 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1-0.20191028180845-3492b2aff503 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/spec v0.19.4
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/gophercloud/gophercloud v0.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0
	github.com/open-cluster-management/api v0.0.0-20200610161514-939cead3902c
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904 // indirect
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/net v0.0.0-20200421231249-e086a090c8fd
	golang.org/x/sys v0.0.0-20200817155316-9781c653f443 // indirect
	golang.org/x/tools v0.0.0-20200403190813-44a64ad78b9b // indirect
	google.golang.org/appengine v1.6.5 // indirect
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
	github.com/coreos/etcd => go.etcd.io/etcd v3.3.22+incompatible
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	k8s.io/client-go => k8s.io/client-go v0.18.2
)
