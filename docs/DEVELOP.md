# requirements

    - go 1.17
    - kubebuilder

optional:

    - k3d

# Mac/Linux with brew

    brew install go@1.17
    brew link go@1.17
    export GOPATH=$HOME/go
    export GOROOT="$(brew --prefix go@1.17)/libexec"
    export PATH="${GOPATH}/bin:${GOROOT}/bin:$PATH"

# init

    # boilerplates
    kubebuilder init --domain k8s.torproject.org --project-name tor-controller --repo github.com/bugfest/tor-controller --component-config

    # We might need to support multiple groups
    kubebuilder edit --multigroup=true

    # Create ProjectConfig CRD
    kubebuilder create api --group config --version v2 --kind ProjectConfig --resource --controller=false --make=false

    # v1alpha1 (to convert from original's project tor-controller)
    kubebuilder create api --group tor --version v1alpha1 --kind OnionService --controller --namespaced --resource
    kubebuilder create api --group tor --version v1alpha1 --kind OnionBalancedService --controller --namespaced --resource

    # v1alpha2 (to implement new OnionService and OnionBalancedService)
    kubebuilder create api --group tor --version v1alpha2 --kind Tor --controller --namespaced --resource
    kubebuilder create api --group tor --version v1alpha2 --kind OnionService --controller --namespaced --resource
    kubebuilder create api --group tor --version v1alpha2 --kind OnionBalancedService --controller --namespaced --resource
    kubebuilder create webhook --group tor --version v1alpha2 --kind OnionService --conversion

    kubebuilder create config --name=tor --controller-image=quay.io/bugfest/tor-controller-manager:latest --output=hack/install.yaml
    
    # edit 
    # apis/tor/v1alpha1/onionservice_types.go
    # apis/tor/v1alpha2/onionservice_types.go
    # apis/tor/v1alpha2/onionbalancedservice_types.go

    # generate manifests
    make manifests
    
    # install CRDs
    make install

    # install sample manifest(s)
    # kubectl apply -f hack/samples/onionservice.yaml

    # run controller against the current cluster
    make run ENABLE_WEBHOOKS=false

To test tor-local-controller (agent)

    go run agents/tor/main.go -namespace default -name example-onion-service

To deploy in a test cluster

    echo "127.0.0.1 onions" | sudo tee -a /etc/hosts
    k3d cluster create onions --registry-create onions:5000

    export REGISTRY=onions:5000
    export IMG=$REGISTRY/tor-controller:latest
    export IMG_DAEMON=$REGISTRY/tor-daemon:latest
    export IMG_DAEMON_MANAGER=$REGISTRY/tor-daemon-manager:latest
    export IMG_ONIONBALANCE_MANAGER=$REGISTRY/tor-onionbalance-manager:latest

    make docker-build-all && make docker-push-all
    # make docker-build && make docker-push
    # make docker-build-daemon && make docker-push-daemon
    # make docker-build-daemon-manager && make docker-push-daemon-manager
    # make docker-build-onionbalance-manager && make docker-push-onionbalance-manager

    # make deploy
    make rundev ENABLE_WEBHOOKS=false

    # deploy some examples
    kubectl apply -f hack/sample/full-example.yaml
    kubectl apply -f hack/sample/onionservice.yaml

# Docker Buildx

    docker buildx build --platform=linux/amd64,linux/arm64,linux/arm -f Dockerfile --tag quay.io/bugfest/tor-controller:latest .
    docker buildx build --platform=linux/amd64,linux/arm64,linux/arm -f Dockerfile.tor-daemon-manager --tag quay.io/bugfest/tor-daemon-manager:latest .
    docker buildx build --platform=linux/amd64,linux/arm64,linux/arm -f Dockerfile.tor-onionbalance-manager --tag quay.io/bugfest/tor-onionbalance-manager:latest .
    
# Helm

    # Update CRDs
    make helm

    # Install local chart with latest images
    helm upgrade --install \
        --set image.repository=onions:5000/tor-controller \
        --set image.tag=latest \
        --set daemon.image.repository=onions:5000/tor-daemon \
        --set daemon.image.tag=latest \
        --set manager.image.repository=onions:5000/tor-daemon-manager \
        --set manager.image.tag=latest \
        --set onionbalance.image.repository=onions:5000/tor-onionbalance-manager \
        --set onionbalance.image.tag=latest \
        tor-controller ./charts/tor-controller

    # Update helm chart README
    docker run --rm --volume "$(pwd)/charts:/helm-docs" -u $(id -u) jnorwood/helm-docs:latest

# Namespaced deployment

1. Use controller's SA to impersonate its permissions

```shell
cat <<EOF | kubectl apply -f - 
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  name: tor-controller
  annotations:
    kubernetes.io/service-account.name: tor-controller
EOF

# your server name goes here
server=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
# the name of the secret containing the service account token goes here
name=tor-controller

ca=$(kubectl get secret/$name -o jsonpath='{.data.ca\.crt}')
token=$(kubectl get secret/$name -o jsonpath='{.data.token}' | base64 --decode)
namespace=$(kubectl get secret/$name -o jsonpath='{.data.namespace}' | base64 --decode)
mysa=$(mktemp)

echo "
apiVersion: v1
kind: Config
clusters:
- name: default-cluster
  cluster:
    certificate-authority-data: ${ca}
    server: ${server}
contexts:
- name: default-context
  context:
    cluster: default-cluster
    namespace: default
    user: default-user
current-context: default-context
users:
- name: default-user
  user:
    token: ${token}
" > $mysa

echo using KUBECONFIG=$mysa
export KUBECONFIG=$mysa
```

[Steps' source](https://stackoverflow.com/questions/47770676/how-to-create-a-kubectl-config-file-for-serviceaccount)

# Prometheus/Grafana

    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
    --set grafana.enabled=false \
    --set alertmanager.enabled=false \
    --set kubeApiServer.enabled=false \
    --set kubelet.enabled=false \
    --set kubeControllerManager.enabled=false \
    --set coreDns.enabled=false \
    --set kubeEtcd.enabled=false \
    --set kubeScheduler.enabled=false \
    --set kubeProxy.enabled=false \
    --set kubeStateMetrics.enabled=false \
    --set nodeExporter.enabled=false \
    --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

    kubectl port-forward svc/kube-prometheus-stack-prometheus 9090
    # browse http://localhost:9090/targets to check the metrics are scraped

# Changelog

    # Update changelog
    make changelog

# Arm64 emulation with QEMU

Check out [QEMU.md](QEMU.md)

# refs

https://book.kubebuilder.io/cronjob-tutorial/running.html
https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html
https://book.kubebuilder.io/reference/generating-crd.html

build tor from source: https://github.com/anthonybudd/tor-controller/blob/7f9c1f44cd415b6e64f12919cf0d6c6b9eca690a/Dockerfile.tor-daemon-manager

onion v3 generation: https://gitlab.torproject.org/djackson/stem/-/blob/master/stem/descriptor/hidden_service.py#L1115

onionbalance docs: https://onionbalance.readthedocs.io/en/latest/v3/tutorial-v3.html
onionbalance proposal: https://community.torproject.org/gsoc/onion-balance-v3/

k3s docker-compose: https://github.com/k3s-io/k3s/blob/master/docker-compose.yml

v1alpha2 draft onionservice:

    rules:
    - port:
        name: "http"
        number: 80
    backend:
        # resource: (mutually exclusive setting with "service")
        #   apiGroup:
        #   kind: (required)
        #   name: (required)
        service:
            name: "myservice"
            port:
                name: http
                # number: 80

Use controller-runtime instead clientsets
https://hackernoon.com/platforms-on-k8s-with-golang-watch-any-crd-0v2o3z1q (from https://github.com/kubernetes-sigs/kubebuilder/issues/1152)

controller-runtime update/create/... objects examples:
https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/example_test.go
