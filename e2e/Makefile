KIND_VERSION = 0.14.0
KUBERNETES_VERSION = 1.24.2
CERT_MANAGER_VERSION = 1.8.0

BINDIR = $(dir $(shell pwd))bin
KIND = $(BINDIR)/kind
KUBECTL = $(BINDIR)/kubectl
KUSTOMIZE = $(BINDIR)/kustomize
KIND_CLUSTER_NAME = t4s
KIND_CONFIG = kind-config.yaml
KUBECONFIG = $(shell pwd)/.kubeconfig
export KUBECTL KUBECONFIG

.PHONY: start
start: $(KIND) $(KUBECTL)
	$(KIND) create cluster --name=$(KIND_CLUSTER_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(KUBERNETES_VERSION) --wait 1m
	$(MAKE) -C ../ docker-build
	kind load docker-image t4s-controller:dev --name=$(KIND_CLUSTER_NAME)
	kind load docker-image t4s-app:dev --name=$(KIND_CLUSTER_NAME)
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -k ../config/e2e
	$(KUBECTL) -n t4s-system wait --for=condition=available --timeout=180s --all deployments

.PHONY: test
test:
	env RUN_E2E=1 \
		go test -v -race . -ginkgo.progress -ginkgo.v -ginkgo.fail-fast

.PHONY: stop
stop: 
	$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)
	-docker image rm t4s-controller:dev
	-docker image rm t4s-app:dev
	-docker image prune -f

$(KIND):
	mkdir -p ../bin
	curl -sfL -o $@ https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-linux-amd64
	chmod a+x $@

$(KUBECTL):
	mkdir -p ../bin
	curl -sfL -o $@ https://dl.k8s.io/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $@
