# 2. oomkilled — OOMKilled

- Run: 2026-11-25 00:16 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `oom_killed`, agent said `oom_killed`
- Must name Deployment `report-generator`: named
- Agent time: 13s, tool calls: 6, tokens: 15114

## Symptom given to the agent

> Reports are not being generated in namespace scn-reports. Find the root cause.

## Final answer

Based on my investigation, I found the root cause:

**What's broken:** The `report-generator` deployment pods are being killed repeatedly because they exceed their memory limit. The pod `report-generator-864d75f9-s5bn5` shows status `OOMKilled` with restart count of 1.

**Evidence:**
- Pod `report-generator-864d75f9-s5bn5` has state: Terminated, Reason: OOMKilled, Exit Code: 137
- The container command is `tail /dev/zero` which continuously writes to zero, causing unbounded memory consumption
- Memory limit is set to only 32Mi while the container requests 16Mi
- Deployment shows 0 available replicas and 1 unavailable

**Root cause:** The container is being killed for using more memory than its limit. The `tail /dev/zero` command continuously allocates memory without bound, exceeding the 32Mi memory limit set in the deployment spec.

```json
{
  "root_cause_category": "oom_killed",
  "resources": [
    {"kind": "Pod", "name": "report-generator-864d75f9-s5bn5"},
    {"kind": "Deployment", "name": "report-generator", "namespace": "scn-reports"}
  ],
  "summary": "The report-generator container is OOMKilled because tail /dev/zero consumes unbounded memory exceeding the 32Mi limit"
}
```

## Full transcript

### 1. user

Reports are not being generated in namespace scn-reports. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"all_namespaces":false,"namespace":"scn-reports","output":"wide","resource_type":"pod"}
```

### 3. tool call: `k8s_get_resources`

```json
{"all_namespaces":false,"namespace":"scn-reports","output":"wide","resource_type":"deployment"}
```

### 4. tool call: `k8s_get_resources`

```json
{"all_namespaces":false,"namespace":"scn-reports","output":"wide","resource_type":"service"}
```

### 5. tool result: `k8s_get_resources`

```text
NAME                              READY   STATUS      RESTARTS     AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
report-api-997dbfb6f-trxf8        1/1     Running     0            5s    10.244.0.27   airgap-sre-control-plane   <none>           <none>
report-generator-864d75f9-s5bn5   0/1     OOMKilled   1 (5s ago)   5s    10.244.0.26   airgap-sre-control-plane   <none>           <none>
```

### 6. tool result: `k8s_get_resources`

```text
NAME               READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS   IMAGES                             SELECTOR
report-api         1/1     1            1           5s    api          docker.io/library/busybox:1.37.0   app=report-api
report-generator   0/1     1            0           5s    generator    docker.io/library/busybox:1.37.0   app=report-generator
```

### 7. tool result: `k8s_get_resources`

```text
No resources found in scn-reports namespace.
```

### 8. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-reports","resource_name":"report-generator-864d75f9-s5bn5","resource_type":"pod"}
```

### 9. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-reports","resource_name":"report-generator","resource_type":"deployment"}
```

### 10. tool call: `k8s_get_events`

```json
{"namespace":"scn-reports"}
```

### 11. tool result: `k8s_describe_resource`

```text
Name:             report-generator-864d75f9-s5bn5
Namespace:        scn-reports
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:19:07 +0000
Labels:           app=report-generator
                  pod-template-hash=864d75f9
Annotations:      <none>
Status:           Running
IP:               10.244.0.26
IPs:
  IP:           10.244.0.26
Controlled By:  ReplicaSet/report-generator-864d75f9
Containers:
  generator:
    Container ID:  containerd://39613745e1d61a273bf6f969afa4c0e94ed9d7dbad1f0ee33db1a467e4b1d295
    Image:         docker.io/library/busybox:1.37.0
    Image ID:      docker.io/library/busybox@sha256:66a6306db78bf2dbf3487f293aa8d6990d8e506fdffab9cc43fe422becf886e4
    Port:          <none>
    Host Port:     <none>
    Command:
      tail
      /dev/zero
    State:          Terminated
      Reason:       OOMKilled
      Exit Code:    137
      Started:      Tue, 24 Nov 2026 22:19:08 +0000
      Finished:     Tue, 24 Nov 2026 22:19:08 +0000
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
      Started:      Tue, 24 Nov 2026 22:19:07 +0000
      Finished:     Tue, 24 Nov 2026 22:19:07 +0000
    Ready:          False
    Restart Count:  1
    Limits:
      memory:  32Mi
    Requests:
      cpu:        10m
      memory:     16Mi
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-f8dlf (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-f8dlf:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    Optional:                false
    DownwardAPI:             true
QoS Class:                   Burstable
Node-Selectors:              <none>
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type     Reason     Age              From               Message
  ----     ------     ----             ----               -------
  Normal   Scheduled  8s               default-scheduler  Successfully assigned scn-reports/report-generator-864d75f9-s5bn5 to airgap-sre-control-plane
  Normal   Pulled     7s (x2 over 8s)  kubelet            spec.containers{generator}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal   Created    7s (x2 over 8s)  kubelet            spec.containers{generator}: Container created
  Normal   Started    7s (x2 over 8s)  kubelet            spec.containers{generator}: Container started
  Warning  BackOff    6s               kubelet            spec.containers{generator}: Back-off restarting failed container generator in pod report-generator-864d75f9-s5bn5_scn-reports(1431c896-cd1b-4e12-9408-e6e751f1786e)
```

### 12. tool result: `k8s_describe_resource`

```text
Name:                   report-generator
Namespace:              scn-reports
CreationTimestamp:      Tue, 24 Nov 2026 22:19:07 +0000
Labels:                 app=report-generator
Annotations:            deployment.kubernetes.io/revision: 1
Selector:               app=report-generator
Replicas:               1 desired | 1 updated | 1 total | 0 available | 1 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=report-generator
  Containers:
   generator:
    Image:      docker.io/library/busybox:1.37.0
    Port:       <none>
    Host Port:  <none>
    Command:
      tail
      /dev/zero
    Limits:
      memory:  32Mi
    Requests:
      cpu:         10m
      memory:      16Mi
    Environment:   <none>
    Mounts:        <none>
  Volumes:         <none>
  Node-Selectors:  <none>
  Tolerations:     <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Progressing    True    NewReplicaSetAvailable
  Available      False   MinimumReplicasUnavailable
OldReplicaSets:  <none>
NewReplicaSet:   report-generator-864d75f9 (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  8s    deployment-controller  Scaled up replica set report-generator-864d75f9 from 0 to 1
```

### 13. tool result: `k8s_get_events`

```text
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-trxf8",
                "namespace": "scn-reports",
                "resourceVersion": "22459",
                "uid": "b9a3e080-76b5-4636-bf8c-5b5c4d961f79"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Successfully assigned scn-reports/report-api-997dbfb6f-trxf8 to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api-997dbfb6f-trxf8.18eb1b2ad3911f73",
                "namespace": "scn-reports",
                "resourceVersion": "22467",
                "uid": "bf0dd132-4df2-45cf-b27a-87b1fc52f019"
            },
            "reason": "Scheduled",
            "reportingComponent": "default-scheduler",
            "reportingInstance": "",
            "source": {
                "component": "default-scheduler"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-trxf8",
                "namespace": "scn-reports",
                "resourceVersion": "22463",
                "uid": "b9a3e080-76b5-4636-bf8c-5b5c4d961f79"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api-997dbfb6f-trxf8.18eb1b2aeff64a64",
                "namespace": "scn-reports",
                "resourceVersion": "22474",
                "uid": "a107881c-4cbc-4936-80a9-aed4cde804a5"
            },
            "reason": "Pulled",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-trxf8",
                "namespace": "scn-reports",
                "resourceVersion": "22463",
                "uid": "b9a3e080-76b5-4636-bf8c-5b5c4d961f79"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api-997dbfb6f-trxf8.18eb1b2af42806fa",
                "namespace": "scn-reports",
                "resourceVersion": "22477",
                "uid": "98daf968-5c0f-42de-9bfb-abc3d21d4b9c"
            },
            "reason": "Created",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-trxf8",
                "namespace": "scn-reports",
                "resourceVersion": "22463",
                "uid": "b9a3e080-76b5-4636-bf8c-5b5c4d961f79"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api-997dbfb6f-trxf8.18eb1b2af7ed9fae",
                "namespace": "scn-reports",
                "resourceVersion": "22479",
                "uid": "26e23572-b626-4573-957a-1e912e5fcbe5"
            },
            "reason": "Started",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "report-api-997dbfb6f",
                "namespace": "scn-reports",
                "resourceVersion": "22452",
                "uid": "29700d05-f405-4d21-bc4b-4037bf6be108"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Created pod: report-api-997dbfb6f-trxf8",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api-997dbfb6f.18eb1b2ad2d08aa3",
                "namespace": "scn-reports",
                "resourceVersion": "22462",
                "uid": "24811b87-4727-4ed1-87c6-561e9e69d646"
            },
            "reason": "SuccessfulCreate",
            "reportingComponent": "replicaset-controller",
            "reportingInstance": "",
            "source": {
                "component": "replicaset-controller"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "report-api",
                "namespace": "scn-reports",
                "resourceVersion": "22450",
                "uid": "cb7b6ecc-6155-44ae-bab2-6719cbb37b08"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Scaled up replica set report-api-997dbfb6f from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-api.18eb1b2ad22b0009",
                "namespace": "scn-reports",
                "resourceVersion": "22457",
                "uid": "4c48e77e-639b-4bab-a819-8e2718090f9b"
            },
            "reason": "ScalingReplicaSet",
            "reportingComponent": "deployment-controller",
            "reportingInstance": "",
            "source": {
                "component": "deployment-controller"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "report-generator-864d75f9-s5bn5",
                "namespace": "scn-reports",
                "resourceVersion": "22451",
                "uid": "1431c896-cd1b-4e12-9408-e6e751f1786e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Successfully assigned scn-reports/report-generator-864d75f9-s5bn5 to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator-864d75f9-s5bn5.18eb1b2ad27f69a1",
                "namespace": "scn-reports",
                "resourceVersion": "22460",
                "uid": "09b80573-11b2-4d8e-9e0f-f12687a0bb3a"
            },
            "reason": "Scheduled",
            "reportingComponent": "default-scheduler",
            "reportingInstance": "",
            "source": {
                "component": "default-scheduler"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 2,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-s5bn5",
                "namespace": "scn-reports",
                "resourceVersion": "22456",
                "uid": "1431c896-cd1b-4e12-9408-e6e751f1786e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:08Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator-864d75f9-s5bn5.18eb1b2aef079925",
                "namespace": "scn-reports",
                "resourceVersion": "22486",
                "uid": "6e9ee936-9ad3-4176-bc28-6a18e73ef8ca"
            },
            "reason": "Pulled",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 2,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-s5bn5",
                "namespace": "scn-reports",
                "resourceVersion": "22456",
                "uid": "1431c896-cd1b-4e12-9408-e6e751f1786e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:08Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator-864d75f9-s5bn5.18eb1b2af3786574",
                "namespace": "scn-reports",
                "resourceVersion": "22490",
                "uid": "4365ef66-c3c3-463b-a506-d07f53f5fd53"
            },
            "reason": "Created",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 2,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-s5bn5",
                "namespace": "scn-reports",
                "resourceVersion": "22456",
                "uid": "1431c896-cd1b-4e12-9408-e6e751f1786e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:08Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator-864d75f9-s5bn5.18eb1b2af7ed9fce",
                "namespace": "scn-reports",
                "resourceVersion": "22491",
                "uid": "1adf12a5-12ad-4362-bda7-2df2850e1d95"
            },
            "reason": "Started",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:09Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-s5bn5",
                "namespace": "scn-reports",
                "resourceVersion": "22456",
                "uid": "1431c896-cd1b-4e12-9408-e6e751f1786e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:09Z",
            "message": "Back-off restarting failed container generator in pod report-generator-864d75f9-s5bn5_scn-reports(1431c896-cd1b-4e12-9408-e6e751f1786e)",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:09Z",
                "name": "report-generator-864d75f9-s5bn5.18eb1b2b56e4041f",
                "namespace": "scn-reports",
                "resourceVersion": "22497",
                "uid": "0da18fd3-58f5-4a3e-8720-7222a5d8cebf"
            },
            "reason": "BackOff",
            "reportingComponent": "kubelet",
            "reportingInstance": "airgap-sre-control-plane",
            "source": {
                "component": "kubelet",
                "host": "airgap-sre-control-plane"
            },
            "type": "Warning"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "report-generator-864d75f9",
                "namespace": "scn-reports",
                "resourceVersion": "22447",
                "uid": "e619d515-1a2e-49d4-88dc-8674c4a68673"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Created pod: report-generator-864d75f9-s5bn5",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator-864d75f9.18eb1b2ad1d5501b",
                "namespace": "scn-reports",
                "resourceVersion": "22453",
                "uid": "83d16ed4-20fa-4208-af39-e9610890132c"
            },
            "reason": "SuccessfulCreate",
            "reportingComponent": "replicaset-controller",
            "reportingInstance": "",
            "source": {
                "component": "replicaset-controller"
            },
            "type": "Normal"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:07Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "report-generator",
                "namespace": "scn-reports",
                "resourceVersion": "22446",
                "uid": "3bfa091c-95f4-46e3-a0ad-5d78b7198df1"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:07Z",
            "message": "Scaled up replica set report-generator-864d75f9 from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:07Z",
                "name": "report-generator.18eb1b2ad105d587",
                "namespace": "scn-reports",
                "resourceVersion": "22448",
                "uid": "db5b5915-b76f-4c5b-b11e-21c8b3333d97"
            },
            "reason": "ScalingReplicaSet",
            "reportingComponent": "deployment-controller",
            "reportingInstance": "",
            "source": {
                "component": "deployment-controller"
            },
            "type": "Normal"
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": ""
    }
}
```

### 14. agent

Based on my investigation, I found the root cause:

**What's broken:** The `report-generator` deployment pods are being killed repeatedly because they exceed their memory limit. The pod `report-generator-864d75f9-s5bn5` shows status `OOMKilled` with restart count of 1.

**Evidence:**
- Pod `report-generator-864d75f9-s5bn5` has state: Terminated, Reason: OOMKilled, Exit Code: 137
- The container command is `tail /dev/zero` which continuously writes to zero, causing unbounded memory consumption
- Memory limit is set to only 32Mi while the container requests 16Mi
- Deployment shows 0 available replicas and 1 unavailable

**Root cause:** The container is being killed for using more memory than its limit. The `tail /dev/zero` command continuously allocates memory without bound, exceeding the 32Mi memory limit set in the deployment spec.

```json
{
  "root_cause_category": "oom_killed",
  "resources": [
    {"kind": "Pod", "name": "report-generator-864d75f9-s5bn5"},
    {"kind": "Deployment", "name": "report-generator", "namespace": "scn-reports"}
  ],
  "summary": "The report-generator container is OOMKilled because tail /dev/zero consumes unbounded memory exceeding the 32Mi limit"
}
```
