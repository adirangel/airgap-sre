# 2. oomkilled — OOMKilled

- Run: 2026-11-25 00:23 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `oom_killed`, agent said `oom_killed`
- Must name Deployment `report-generator`: named, also in the JSON block
- Agent time: 20s, tool calls: 8, tokens: 33718

## Symptom given to the agent

> Reports are not being generated in namespace scn-reports. Find the root cause.

## Final answer

The root cause is clear from the pod state and events:

**What's broken:** The `report-generator` deployment pods are in CrashLoopBackOff state because containers are being killed due to memory exhaustion.

**Evidence:**
- Pod `report-generator-864d75f9-8nrlt` shows `State: Waiting, Reason: CrashLoopBackOff`
- Container's `Last State` shows `Reason: OOMKilled` and `Exit Code: 137` (exit code 128 + 9 for SIGKILL = OOMKilled)
- The container has a memory limit of **32Mi** but the command `tail /dev/zero` continuously writes to stdout, consuming memory until it exceeds the limit
- Events show repeated "Back-off restarting failed container" messages

**Root cause:** The container is being killed for using more memory than its limit. The `tail /dev/zero` command is a memory-intensive operation that exhausts the 32Mi memory limit set in the deployment spec.

```json
{
  "root_cause_category": "oom_killed",
  "resources": [
    {"kind": "Deployment", "name": "report-generator"},
    {"kind": "Pod", "name": "report-generator-864d75f9-8nrlt"},
    {"kind": "ConfigKey", "key": "memory"}
  ],
  "summary": "The report-generator container is being OOMKilled because the tail /dev/zero command exceeds the 32Mi memory limit configured in the deployment spec."
}
```

## Full transcript

### 1. user

Reports are not being generated in namespace scn-reports. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-reports","output":"wide","resource_type":"pod"}
```

### 3. tool call: `k8s_get_resources`

```json
{"namespace":"scn-reports","output":"wide","resource_type":"deployment"}
```

### 4. tool call: `k8s_get_resources`

```json
{"namespace":"scn-reports","output":"wide","resource_type":"service"}
```

### 5. tool result: `k8s_get_resources`

```text
NAME                              READY   STATUS             RESTARTS     AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
report-api-997dbfb6f-r2z4k        1/1     Running            0            5s    10.244.0.34   airgap-sre-control-plane   <none>           <none>
report-generator-864d75f9-8nrlt   0/1     CrashLoopBackOff   1 (4s ago)   5s    10.244.0.33   airgap-sre-control-plane   <none>           <none>
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

### 8. tool call: `k8s_get_events`

```json
{"namespace":"scn-reports"}
```

### 9. tool call: `k8s_get_pod_logs`

```json
{"namespace":"scn-reports","pod_name":"report-generator-864d75f9-8nrlt","previous":true}
```

### 10. tool result: `k8s_get_events`

```text
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-r2z4k",
                "namespace": "scn-reports",
                "resourceVersion": "23219",
                "uid": "11703f70-4c34-485d-8c4e-628209b9d63c"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Successfully assigned scn-reports/report-api-997dbfb6f-r2z4k to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api-997dbfb6f-r2z4k.18eb1b66efadeca7",
                "namespace": "scn-reports",
                "resourceVersion": "23226",
                "uid": "22e36acc-5344-47f3-8495-4357deef353e"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-r2z4k",
                "namespace": "scn-reports",
                "resourceVersion": "23222",
                "uid": "11703f70-4c34-485d-8c4e-628209b9d63c"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api-997dbfb6f-r2z4k.18eb1b670ab04f5e",
                "namespace": "scn-reports",
                "resourceVersion": "23234",
                "uid": "a3619d66-3117-4afb-a37f-b43c86e4c32d"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-r2z4k",
                "namespace": "scn-reports",
                "resourceVersion": "23222",
                "uid": "11703f70-4c34-485d-8c4e-628209b9d63c"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api-997dbfb6f-r2z4k.18eb1b670e0a820b",
                "namespace": "scn-reports",
                "resourceVersion": "23237",
                "uid": "8fc9dd64-cc23-4a36-b0b8-b010b1683126"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{api}",
                "kind": "Pod",
                "name": "report-api-997dbfb6f-r2z4k",
                "namespace": "scn-reports",
                "resourceVersion": "23222",
                "uid": "11703f70-4c34-485d-8c4e-628209b9d63c"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api-997dbfb6f-r2z4k.18eb1b67103b0c8f",
                "namespace": "scn-reports",
                "resourceVersion": "23239",
                "uid": "f9fa80c8-19f0-41b6-8385-3369705632ce"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "report-api-997dbfb6f",
                "namespace": "scn-reports",
                "resourceVersion": "23214",
                "uid": "86879824-7ca3-4d18-bebd-89910f4f9183"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Created pod: report-api-997dbfb6f-r2z4k",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api-997dbfb6f.18eb1b66ef150517",
                "namespace": "scn-reports",
                "resourceVersion": "23221",
                "uid": "c0a6e588-d55d-4ae5-bbb7-0b6dba401c3b"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "report-api",
                "namespace": "scn-reports",
                "resourceVersion": "23209",
                "uid": "1eed9f41-b190-43d2-a717-cb9ba1ab8389"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Scaled up replica set report-api-997dbfb6f from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-api.18eb1b66ee51ea0b",
                "namespace": "scn-reports",
                "resourceVersion": "23217",
                "uid": "4c938fb6-8495-4dd1-a4ea-353e433d62f9"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "report-generator-864d75f9-8nrlt",
                "namespace": "scn-reports",
                "resourceVersion": "23208",
                "uid": "a2ea72b4-05ea-4db9-953a-d30217cdc4a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Successfully assigned scn-reports/report-generator-864d75f9-8nrlt to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator-864d75f9-8nrlt.18eb1b66ee364dfa",
                "namespace": "scn-reports",
                "resourceVersion": "23215",
                "uid": "d67d26bf-a2c2-413e-b6eb-5b39c69128d0"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-8nrlt",
                "namespace": "scn-reports",
                "resourceVersion": "23210",
                "uid": "a2ea72b4-05ea-4db9-953a-d30217cdc4a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:26Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator-864d75f9-8nrlt.18eb1b67096001fa",
                "namespace": "scn-reports",
                "resourceVersion": "23247",
                "uid": "a917d608-c84c-4878-8df1-75484a93f734"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-8nrlt",
                "namespace": "scn-reports",
                "resourceVersion": "23210",
                "uid": "a2ea72b4-05ea-4db9-953a-d30217cdc4a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:26Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator-864d75f9-8nrlt.18eb1b670cd8bc5b",
                "namespace": "scn-reports",
                "resourceVersion": "23251",
                "uid": "c17bcecd-4a5c-468f-94a3-0f33d8bceff3"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-8nrlt",
                "namespace": "scn-reports",
                "resourceVersion": "23210",
                "uid": "a2ea72b4-05ea-4db9-953a-d30217cdc4a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:26Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator-864d75f9-8nrlt.18eb1b670fd516f9",
                "namespace": "scn-reports",
                "resourceVersion": "23252",
                "uid": "4d5b9b97-6eb5-46cb-9c59-52025f34ebfc"
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
            "count": 2,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:23:27Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{generator}",
                "kind": "Pod",
                "name": "report-generator-864d75f9-8nrlt",
                "namespace": "scn-reports",
                "resourceVersion": "23210",
                "uid": "a2ea72b4-05ea-4db9-953a-d30217cdc4a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:28Z",
            "message": "Back-off restarting failed container generator in pod report-generator-864d75f9-8nrlt_scn-reports(a2ea72b4-05ea-4db9-953a-d30217cdc4a8)",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:27Z",
                "name": "report-generator-864d75f9-8nrlt.18eb1b678376d9da",
                "namespace": "scn-reports",
                "resourceVersion": "23262",
                "uid": "3544dcb1-c7b1-4d50-8052-54632aafe010"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "report-generator-864d75f9",
                "namespace": "scn-reports",
                "resourceVersion": "23205",
                "uid": "a0833726-d152-4385-ba41-b6739e0d580a"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Created pod: report-generator-864d75f9-8nrlt",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator-864d75f9.18eb1b66ed9bc4cc",
                "namespace": "scn-reports",
                "resourceVersion": "23211",
                "uid": "ac03bd20-e7a3-459a-a5bf-774f8a049b46"
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
            "firstTimestamp": "2026-11-24T22:23:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "report-generator",
                "namespace": "scn-reports",
                "resourceVersion": "23204",
                "uid": "9ecffb57-21ff-4b01-8423-23f1741d4f1e"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:25Z",
            "message": "Scaled up replica set report-generator-864d75f9 from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:25Z",
                "name": "report-generator.18eb1b66ecd3e3af",
                "namespace": "scn-reports",
                "resourceVersion": "23206",
                "uid": "ccdf5d5a-07dd-4986-808f-3a954166a430"
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

### 11. tool result: `k8s_get_pod_logs`

```text

```

### 12. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-reports","resource_name":"report-generator","resource_type":"deployment"}
```

### 13. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-reports","resource_name":"report-generator-864d75f9-8nrlt","resource_type":"pod"}
```

### 14. tool result: `k8s_describe_resource`

```text
Name:                   report-generator
Namespace:              scn-reports
CreationTimestamp:      Tue, 24 Nov 2026 22:23:25 +0000
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
  Normal  ScalingReplicaSet  11s   deployment-controller  Scaled up replica set report-generator-864d75f9 from 0 to 1
```

### 15. tool result: `k8s_describe_resource`

```text
Name:             report-generator-864d75f9-8nrlt
Namespace:        scn-reports
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:23:25 +0000
Labels:           app=report-generator
                  pod-template-hash=864d75f9
Annotations:      <none>
Status:           Running
IP:               10.244.0.33
IPs:
  IP:           10.244.0.33
Controlled By:  ReplicaSet/report-generator-864d75f9
Containers:
  generator:
    Container ID:  containerd://6e6d0459e627de57abe2912cb192916e617dbdab65f459540e8ddade9b0aca89
    Image:         docker.io/library/busybox:1.37.0
    Image ID:      docker.io/library/busybox@sha256:66a6306db78bf2dbf3487f293aa8d6990d8e506fdffab9cc43fe422becf886e4
    Port:          <none>
    Host Port:     <none>
    Command:
      tail
      /dev/zero
    State:          Waiting
      Reason:       CrashLoopBackOff
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
      Started:      Tue, 24 Nov 2026 22:23:26 +0000
      Finished:     Tue, 24 Nov 2026 22:23:26 +0000
    Ready:          False
    Restart Count:  1
    Limits:
      memory:  32Mi
    Requests:
      cpu:        10m
      memory:     16Mi
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-79wzr (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-79wzr:
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
  Type     Reason     Age                From               Message
  ----     ------     ----               ----               -------
  Normal   Scheduled  11s                default-scheduler  Successfully assigned scn-reports/report-generator-864d75f9-8nrlt to airgap-sre-control-plane
  Normal   Pulled     10s (x2 over 11s)  kubelet            spec.containers{generator}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal   Created    10s (x2 over 11s)  kubelet            spec.containers{generator}: Container created
  Normal   Started    10s (x2 over 11s)  kubelet            spec.containers{generator}: Container started
  Warning  BackOff    8s (x2 over 9s)    kubelet            spec.containers{generator}: Back-off restarting failed container generator in pod report-generator-864d75f9-8nrlt_scn-reports(a2ea72b4-05ea-4db9-953a-d30217cdc4a8)
```

### 16. tool call: `k8s_get_pod_logs`

```json
{"container":"generator","namespace":"scn-reports","pod_name":"report-generator-864d75f9-8nrlt"}
```

### 17. tool result: `k8s_get_pod_logs`

```text

```

### 18. agent

The root cause is clear from the pod state and events:

**What's broken:** The `report-generator` deployment pods are in CrashLoopBackOff state because containers are being killed due to memory exhaustion.

**Evidence:**
- Pod `report-generator-864d75f9-8nrlt` shows `State: Waiting, Reason: CrashLoopBackOff`
- Container's `Last State` shows `Reason: OOMKilled` and `Exit Code: 137` (exit code 128 + 9 for SIGKILL = OOMKilled)
- The container has a memory limit of **32Mi** but the command `tail /dev/zero` continuously writes to stdout, consuming memory until it exceeds the limit
- Events show repeated "Back-off restarting failed container" messages

**Root cause:** The container is being killed for using more memory than its limit. The `tail /dev/zero` command is a memory-intensive operation that exhausts the 32Mi memory limit set in the deployment spec.

```json
{
  "root_cause_category": "oom_killed",
  "resources": [
    {"kind": "Deployment", "name": "report-generator"},
    {"kind": "Pod", "name": "report-generator-864d75f9-8nrlt"},
    {"kind": "ConfigKey", "key": "memory"}
  ],
  "summary": "The report-generator container is being OOMKilled because the tail /dev/zero command exceeds the 32Mi memory limit configured in the deployment spec."
}
```
