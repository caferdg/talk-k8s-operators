##@ Demonstration

KIND_CLUSTER ?= demo


.PHONY: setup-cluster
setup-cluster: kind-create setup-vault-eso setup-cert-manager ## Set up the full demo cluster with secrets.
	kubectl create namespace demo-system
	kubectl config set-context --current --namespace=demo-system
	kubectl create secret generic gcp-sa-key \
		--namespace=external-secrets \
		--from-file=credentials.json=PATH-TO-YOUR-SERVICE-ACCOUNT-CREDENTIALS
	kubectl apply -f config/eso/cluster-secret-store.yaml

.PHONY: setup-cert-manager
setup-cert-manager: ## Install cert-manager in the cluster, required for admission webhooks.
	helm repo add jetstack https://charts.jetstack.io
	helm repo update jetstack
	helm install cert-manager jetstack/cert-manager \
		--namespace cert-manager --create-namespace \
		--set crds.enabled=true
	kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=120s
	
.PHONY: kind-create
kind-create: ## Create a Kind cluster.
	@command -v $(KIND) >/dev/null 2>&1 || { echo "Kind is not installed."; exit 1; }
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) echo "Kind cluster '$(KIND_CLUSTER)' already exists." ;; \
		*) $(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: setup-eso
setup-vault-eso: ## Install External Secrets Operator.
	helm repo add external-secrets https://charts.external-secrets.io
	helm repo update external-secrets
	helm install external-secrets external-secrets/external-secrets \
		--namespace external-secrets --create-namespace
	@echo "Waiting for pods to be scheduled..."
	@until kubectl get pod -l app.kubernetes.io/name=external-secrets -n external-secrets 2>/dev/null | grep -q external-secrets; do sleep 2; done
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=external-secrets -n external-secrets --timeout=120s
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=external-secrets-webhook -n external-secrets --timeout=120s

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller, deleting CRs first to let finalizers run.
	-$(KUBECTL) delete gitlabprojects,gitlabgroups,notifiers --all -A --timeout=60s
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

trigger-ci-success/%: ## Trigger a passing pipeline. Usage: make trigger-ci-success/<cr-name>
	@hack/push-ci.sh "$*" pass

trigger-ci-failure/%: ## Trigger a failing pipeline. Usage: make trigger-ci-failure/<cr-name>
	@hack/push-ci.sh "$*" fail

.PHONY: clean
clean: ## Delete the Kind cluster.
	$(KIND) delete cluster --name $(KIND_CLUSTER)
