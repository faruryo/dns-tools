![Docker](https://github.com/faruryo/dns-tools/workflows/Docker/badge.svg)


## Developing

### skaffold

```shell
skaffold dev --port-forward
```

### Knative Eventing setup

install
```
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.18.0/eventing-crds.yaml
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.18.0/eventing-core.yaml

kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.18.0/in-memory-channel.yaml
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.18.0/mt-channel-broker.yaml
```

check
```
kubectl get pods -n knative-eventing
```

[Installing Knative | Knative](https://knative.dev/docs/install/any-kubernetes-cluster/)
