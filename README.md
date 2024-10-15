# automq-operator

This [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) is made to easily deploy AutoMQ onto your Kubernetes cluster.

Goals:

- [x] Automatically deploy and manage a AutoMQ cluster
- [x] Ability to be managed by other Operators
- [x] Auto rolling upgrade and restart
- [ ] Grafana dashboard
- [ ] Pod Affinity and Anti-Affinity

## Description

AutoMQ is a message queue system that is designed to be easy to use and deploy. It is built on top of the [automq](https://www.automq.com) messaging system and provides a simple API for sending and receiving messages.

This operator is designed to make it easy to deploy AutoMQ onto your Kubernetes cluster. It will automatically deploy and manage a AutoMQ cluster, and can be managed by other Operators.

## Prerequisites

- Kubernetes 1.21+
- Helm 3.x.x
- StorageClass
- S3 storage ( minio, aws s3, etc )
- Cert-Manager


## Getting Started

### Install sealos binary

```shell
curl -sfL https://raw.githubusercontent.com/labring/sealos/v5.0.0/scripts/install.sh | sh -s v5.0.0  labring/sealos
```


### Installation Kubernetes

```shell
sealos run labring/kubernetes:v1.27.7 
````

### Installation dependencies

```shell
sealos run labring/helm:v3.9.4 labring/calico:v3.26.5
sealos run labring/openebs:v3.9.0
sealos run labring/cert-manager:v1.14.6
sealos run labring/minio:RELEASE.2024-01-11T07-46-16Z
sealos run labring/kube-prometheus-stack:v0.63.0 
sealos run labring/kafka-ui:v0.7.1
```

### Installation Operator 

#### For dev version

1.  Using sealos images install operator

    ```shell
    sealos run ghcr.io/cuisongliu/automq-operator-sealos:latest
    ```

2.  Using helm chart install operator

    ```shell
    git clone https://github.com/cuisongliu/automq-operator.git
    cd automq-operator
    IMG=ghcr.io/cuisongliu/automq-operator:latest make set-image
    cd deploy
    bash install.sh
    ```

#### For release version

1.  Using sealos images install operator

    ```shell
    sealos run ghcr.io/cuisongliu/automq-operator-sealos:v0.0.4
    ```

2.  Using helm chart install operator

    ```shell
    wget https://github.com/cuisongliu/automq-operator/releases/download/v0.0.4/automq-operator-v0.0.4-sealos.tgz
    mkdir -p automq-operator
    tar -zxvf automq-operator-v0.0.4-sealos.tgz -C automq-operator
    cd automq-operator/deploy
    bash install.sh
    ```

### Install AutoMQ

```shell

cat <<EOF | kubectl apply -f -
apiVersion: infra.cuisongliu.github.com/v1beta1
kind: AutoMQ
metadata:
  name: automq
spec:
  s3:
    endpoint: http://minio.minio.svc.cluster.local:9000
    region: cn-north-1
    accessKeyID: admin
    secretAccessKey: minio123
    bucket: automq
    enablePathStyle: true
  nodePort: 32009
  controller:
    replicas: 3
    jvmOptions:
      - -Xms1g
      - -Xmx1g
      - -XX:MetaspaceSize=96m
  clusterID: "rZdE0DjZSrqy96PXrMUZVw"
EOF

```

### Verify AutoMQ

```shell
kubectl get pods -n default
kubectl get svc -n default
```


### Uninstall Operator

```shell
kubectl get automq -A -o yaml | kubectl delete -f -
helm delete -n automq-operator automq-operator
```

## 👩‍💻 Contributing & Development

Have a look through [existing Issues](https://github.com/cuisongliu/automq-operator/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc) and [Pull Requests](https://github.com/cuisongliu/automq-operator/pulls?q=is%3Apr+is%3Aopen+sort%3Aupdated-desc) that you could help with. If you'd like to request a feature or report a bug, please [create a GitHub Issue](https://github.com/cuisongliu/automq-operator/issues/new/choose) using one of the templates provided.

📖 [See contribution guide →](./CONTRIBUTING.md)

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make generate && make manifests 
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024 cuisongliu.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

