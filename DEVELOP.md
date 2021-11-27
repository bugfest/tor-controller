# requirements

    brew install kubebuilder

# init

    # boilerplates
    kubebuilder init --domain k8s.torproject.org --project-name tor-controller --repo example.com/null/tor-controller

    # We might need to support multiple groups
    kubebuilder edit --multigroup=true

    # v1alpha1 (to convert from original's project tor-controller)
    kubebuilder create api --group tor --version v1alpha1 --kind OnionService --controller --namespaced --resource
    kubebuilder create api --group tor --version v1alpha1 --kind OnionBalancedService --controller --namespaced --resource

    # v1alpha2 (to implement new OnionService and OnionBalancedService)
    kubebuilder create api --group tor --version v1alpha2 --kind OnionService --controller --namespaced --resource
    kubebuilder create api --group tor --version v1alpha2 --kind OnionBalancedService --controller --namespaced --resource
    kubebuilder create webhook --group tor --version v1alpha2 --kind OnionService --conversion

    # edit 
    # apis/tor/v1alpha1/onionservice_types.go
    # apis/tor/v1alpha2/onionservice_types.go
    # apis/tor/v1alpha2/onionbalancedservice_types.go

    # generate manifests
    make manifests

    # start k8s
    # cd docker; docker-compose -f docker-compose.k3s.yaml up -d; export KUBECONFIG=$(pwd)/kubeconfig.yaml

    # install CRDs
    make install

    # install sample manifest(s)
    # kubectl apply -f hack/samples/onionservice.yaml

    # ron controller against the current cluster
    make run ENABLE_WEBHOOKS=false

# changes

Changes vs https://github.com/kragniz/tor-controller version

- Go updated to `1.16`
- Code ported to kubebuilder version `3`
- Domain updated moved from protected `tor.k8s.io` to `k8s.torproject.org` (see https://github.com/kubernetes/enhancements/pull/1111)
- Added `OnionBalancedService` type

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
