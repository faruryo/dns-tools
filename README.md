![Docker](https://github.com/faruryo/dns-tools/workflows/Docker/badge.svg)


## Developing

### Knative Eventing setup

install
```
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.23.1/eventing-crds.yaml
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.23.1/eventing-core.yaml

kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.23.1/in-memory-channel.yaml
kubectl apply --filename https://github.com/knative/eventing/releases/download/v0.23.1/mt-channel-broker.yaml
```

check
```
kubectl get pods -n knative-eventing
```

[Installing Knative | Knative](https://knative.dev/docs/install/any-kubernetes-cluster/)

### Setup secret

Cloudflare tokens can be created at User Profile > API Tokens > API Tokens. The following settings are recommended:

- Permissions:
  - Zone - DNS - Edit
  - Zone - Zone - Read
- Zone Resources:
  - Include - All Zones

Set environment variables; we recommend using [direnv](https://github.com/direnv/direnv).

```shell
export CLOUDFLARE_API_TOKEN="hogehoge"
```

Create a secret using an environment variable.

```shell
kubectl create secret generic dns-tools \
    --from-literal=CLOUDFLARE_API_TOKEN=$CLOUDFLARE_API_TOKEN
```

### skaffold

```shell
skaffold dev --port-forward
```
