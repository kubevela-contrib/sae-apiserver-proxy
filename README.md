# SAE APIServer Proxy

[![ImageBuild](https://github.com/kubevela-contrib/sae-apiserver-proxy/actions/workflows/image-build.yml/badge.svg)](https://github.com/kubevela-contrib/sae-apiserver-proxy/actions/workflows/image-build.yml/badge.svg)
[![Releases](https://img.shields.io/github/release/kubevela-contrib/sae-apiserver-proxy/all.svg)](https://github.com/kubevela-contrib/sae-apiserver-proxy/releases)
[![LICENSE](https://img.shields.io/github/license/kubevela-contrib/sae-apiserver-proxy.svg)](/LICENSE)

[DockerHub](https://hub.docker.com/r/kubevelacontrib/sae-apiserver-proxy/tags) | [SAE](https://www.aliyun.com/product/sae) | [KubeVela](https://github.com/kubevela/kubevela/)

This is a proxy server that implements the Kubernetes aggregated APIServer and provide proxy access to SAE APIServer.

It can be used natively as a cluster in ClusterGateway. Therefore, it wraps the SAE APIServer into a Kubernetes cluster for ClusterGateway. It is possible for KubeVela related projects (including Core Controller, CommandLine Tools and Workflow Controller) to leverage the multi-cluster management capabilities, such as deploying KubeVela application to SAE.

> Notice that you need to install ClusterGateway >=v1.7.0 to make correct usage.

To install the SAE APIServer, you can pull the code repo and run

```shell
helm install ./charts
```

To register a SAE APIServer proxy, apply the following YAML.

```yaml
apiVersion: sae.alibaba-cloud.oam.dev/v1alpha1
kind: SAEAPIServer
metadata:
  name: sae-stage
spec:
  accessKeyId: <your aliyun accessKeyId>
  accessKeySecret: <your aliyun accessKeySecret>
  region: <the SAE APIServer region>
```

You can check it through running `kubectl get saeapiserver` and see
```shell
NAME          REGION        AK
sae-stage     cn-hangzhou   <your aliyun accessKeyId>
```

You can change the saeapiserver by `kubectl edit saeapiserver` if you want to update your AK/SK or delete it if expired.

Now in the KubeVela system, you can use `vela cluster list` to see your cluster
```shell
CLUSTER         ALIAS   TYPE                ENDPOINT                                              ACCEPTED        LABELS                                                
local                   Internal            -                                                     true                                                                  
sae-stage               ServiceAccountToken https://.../apis/sae.alibaba-cloud.oam.dev/v1al...    true            sae.alibaba-cloud.oam.dev/apiserver=true              
                                                                                                              sae.alibaba-cloud.oam.dev/apiserver-region=cn-hangzhou
```

Start your journey with KubeVela application!
```yaml
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: sae-app
  namespace: default
spec:
  components:
    - type: webservice
      name: sae-app
      properties:
        image: nginx
      traits:
        - type: expose
          properties:
            port: [80]
            type: LoadBalancer
        - type: scaler
          properties:
            replicas: 3
  policies:
    - type: topology
      name: sae-stage
      properties:
        clusters: ["sae-stage"]
```