<p align="center">
  <img height="300" src="https://sr.ht/2mc0.png">
</p>

<h1 align="center">tor-controller</h1>

This project started as an exercise to update `kragniz`'s https://github.com/kragniz/tor-controller version

***Important!!*** This project is not backward compatible with kragniz's OnionService definitions. You will need to update your OnionService manifests

# Changes

- Go updated to `1.17`
- Code ported to kubebuilder version `3`
- Domain updated moved from `tor.k8s.io` (protected) to `k8s.torproject.org` (see https://github.com/kubernetes/enhancements/pull/1111)
- Added `OnionBalancedService` type
- New OnionService version v1alpha2
- Migrate clientset code to controller-runtime
- Helm chart

# Roadmap / TODO list

- Implement `OnionBalancedService` resource (HA Onion Services)
- Metrics exporters
- TOR daemon management via socket (e.g: config reload)

# TOR

Tor is an anonymity network that provides:

- privacy
- enhanced tamperproofing
- freedom from network surveillance
- NAT traversal

tor-controller allows you to create `OnionService` resources in kubernetes.
These services are used similarly to standard kubernetes services, but they
only serve traffic on the tor network (available on `.onion` addresses).

See [this page](https://www.torproject.org/docs/onion-services.html.en) for
more information about onion services.

tor-controller creates the following resources for each OnionService:

- a service, which is used to send traffic to application pods
- tor pod, which contains a tor daemon to serve incoming traffic from the tor
  network, and a management process that watches the kubernetes API and
  generates tor config, signaling the tor daemon when it changes
- rbac rules

<p align="center">
  <img src="https://sr.ht/6WbX.png">
</p>

Install
-------

Using helm (recommended):

    $ helm upgrade --install \
      --create-namespace --namespace tor-controller \
      tor-controller ./helm/tor-controller

Install tor-controller directly using the manifest:

    $ kubectl apply -f hack/install.yaml

Quickstart with random address
------------------------------

Create some deployment to test against, in this example we'll deploy an echoserver. You can find the definition at [hack/sample/echoserver.yaml](hack/sample/echoserver.yaml):

Apply it:

    $ kubectl apply -f hack/sample/echoserver.yaml

For a fixed address, we need a private key. This should be kept safe, since
someone can impersonate your onion service if it is leaked. Tor-Controller will generate an Onion v3 key-pair for you (stored as a secret), unless it already exists

Create an onion service, [hack/sample/onionservice.yaml](hack/sample/onionservice.yaml), referencing an existing private key is optional:

```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  version: 3
  rules:
    - port:
        number: 80
      backend:
        service:
          name: http-app
          port:
            number: 8080
```

Apply it:

    $ kubectl apply -f hack/sample/onionservice.yaml

List active OnionServices:

```bash
$ kubectl get onionservices
NAME                    HOSTNAME                                                         TARGETCLUSTERIP   AGE
example-onion-service   cfoj4552cvq7fbge6k22qmkun3jl37oz273hndr7ktvoahnqg5kdnzqd.onion   10.43.252.41      1m
```

This service should now be accessable from any tor client,
for example [Tor Browser](https://www.torproject.org/projects/torbrowser.html.en):

Random service names
--------------------

If `spec.privateKeySecret` is not specified, tor-controller will start a service with a random name. The key-pair is stored in the same namespace as the tor-daemon, with the name `ONIONSERVICENAME-tor-secret`

Onion service versions
----------------------

The `spec.version` field specifies which onion protocol to use.
Only v3 is supported. 

tor-controller defaults to using v3 if `spec.version` is not specified.


Using with nginx-ingress
------------------------

tor-controller on its own simply directs TCP traffic to a backend service.
If you want to serve HTTP stuff, you'll probably want to pair it with
nginx-ingress or some other ingress controller.

To do this, first install nginx-ingress normally. Then point an onion service
at the nginx-ingress-controller, for example:

```yaml
apiVersion: tor.k8s.torproject.org/v1alpha2
kind: OnionService
metadata:
  name: example-onion-service
spec:
  version: 3
  rules:
    - port:
        number: 80
      backend:
        service:
          name: http-app
          port:
            number: 8080
  privateKeySecret:
    name: nginx-onion-key
    key: private_key
```

This can then be used in the same way any other ingress is. You can find a full
example, with a default backend at [hack/sample/full-example.yaml](hack/sample/full-example.yaml)

# Utils
- Helm docs updated with https://github.com/norwoodj/helm-docs


# Other projects
- https://github.com/rdkr/oniongen-go
- https://github.com/ajvb/awesome-tor

# References:
- https://tor.void.gr/docs/tor-manual.html.en
