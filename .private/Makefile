DOMAIN ?=mk100120.com

initProject:
	kubebuilder init --plugins go/v4 --domain github.com --owner "kenny"

createApi:
	kubebuilder create api --group configuration --version v1 --kind Application
