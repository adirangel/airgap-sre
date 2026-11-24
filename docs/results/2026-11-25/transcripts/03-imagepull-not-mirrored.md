# 3. imagepull-not-mirrored — ImagePullBackOff because the image was never mirrored

- Run: 2026-11-25 00:16 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `image_pull_failure`, agent said `image_pull_failure`
- Must name Deployment `session-cache`: named
- Must name Image `redis:7.4-alpine`: named
- Agent time: 13s, tool calls: 4, tokens: 16321

## Symptom given to the agent

> The session-cache workload in namespace scn-sessions never starts. Find the root cause.

## Final answer

Based on my investigation, the pod `session-cache-6bb4646854-vzbwl` is stuck in `ErrImagePull` state. The events clearly show that Kubernetes attempted to pull the image `docker.io/library/redis:7.4-alpine` but received a "NotFound" error - the image cannot be resolved or pulled from the registry.

The deployment spec confirms the container uses this exact image, and the pod is on node `airgap-sre-control-plane`. The scheduler successfully placed the pod (no scheduling issues), but the kubelet failed to pull the container image.

**Root cause:** The image `docker.io/library/redis:7.4-alpine` is not available in the registry or cannot be pulled by the cluster's image registry.

```json
{
  "root_cause_category": "image_pull_failure",
  "resources": [
    {"kind": "Pod", "name": "session-cache-6bb4646854-vzbwl"},
    {"kind": "Deployment", "name": "session-cache"},
    {"kind": "Image", "name": "docker.io/library/redis:7.4-alpine"}
  ],
  "summary": "The pod cannot start because the container image docker.io/library/redis:7.4-alpine cannot be pulled from the registry."
}
```

## Full transcript

### 1. user

The session-cache workload in namespace scn-sessions never starts. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-sessions","output":"wide","resource_type":"pod"}
```

### 3. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-sessions","resource_name":"session-cache","resource_type":"pod"}
```

### 4. tool call: `k8s_get_events`

```json
{"namespace":"scn-sessions"}
```

### 5. tool result: `k8s_get_resources`

```text
NAME                             READY   STATUS         RESTARTS   AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
session-cache-6bb4646854-vzbwl   0/1     ErrImagePull   0          4s    10.244.0.28   airgap-sre-control-plane   <none>           <none>
```

### 6. tool result: `k8s_describe_resource`

```text
Name:             session-cache-6bb4646854-vzbwl
Namespace:        scn-sessions
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:19:33 +0000
Labels:           app=session-cache
                  pod-template-hash=6bb4646854
Annotations:      <none>
Status:           Pending
IP:               10.244.0.28
IPs:
  IP:           10.244.0.28
Controlled By:  ReplicaSet/session-cache-6bb4646854
Containers:
  redis:
    Container ID:   
    Image:          docker.io/library/redis:7.4-alpine
    Image ID:       
    Port:           6379/TCP
    Host Port:      0/TCP
    State:          Waiting
      Reason:       ErrImagePull
    Ready:          False
    Restart Count:  0
    Limits:
      memory:  64Mi
    Requests:
      cpu:        10m
      memory:     16Mi
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-fz86x (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-fz86x:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    Optional:                false
    DownwardAPI:             true
QoS Class:                   Burstable
Node-Selectors:              <none>
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
```

### 7. tool result: `k8s_get_events`

```text
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22595",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Successfully assigned scn-sessions/session-cache-6bb4646854-vzbwl to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b30dc37ad80",
                "namespace": "scn-sessions",
                "resourceVersion": "22600",
                "uid": "6088db5c-d5d7-4bbd-8052-f36f39d0ae92"
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
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{redis}",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22597",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Pulling image \"docker.io/library/redis:7.4-alpine\"",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b30f66cf7ed",
                "namespace": "scn-sessions",
                "resourceVersion": "22604",
                "uid": "7e87f07a-d0db-4d88-b941-ca0c5afe48a2"
            },
            "reason": "Pulling",
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
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{redis}",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22597",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Failed to pull image \"docker.io/library/redis:7.4-alpine\": rpc error: code = NotFound desc = failed to pull and unpack image \"docker.io/library/redis:7.4-alpine\": failed to resolve reference \"docker.io/library/redis:7.4-alpine\": docker.io/library/redis:7.4-alpine: not found",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b30f750718b",
                "namespace": "scn-sessions",
                "resourceVersion": "22606",
                "uid": "48ed62e7-f924-4933-b299-d3080281196d"
            },
            "reason": "Failed",
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
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{redis}",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22597",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Error: ErrImagePull",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b30f7521e46",
                "namespace": "scn-sessions",
                "resourceVersion": "22607",
                "uid": "de911c6d-f6e7-4419-8b1b-19ebb8c9f5de"
            },
            "reason": "Failed",
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
            "firstTimestamp": "2026-11-24T22:19:34Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{redis}",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22597",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:34Z",
            "message": "Back-off pulling image \"docker.io/library/redis:7.4-alpine\"",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:34Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b312c5847df",
                "namespace": "scn-sessions",
                "resourceVersion": "22608",
                "uid": "ae5ced3f-58f3-47cf-a68f-9849fcb3d8f7"
            },
            "reason": "BackOff",
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
            "firstTimestamp": "2026-11-24T22:19:34Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{redis}",
                "kind": "Pod",
                "name": "session-cache-6bb4646854-vzbwl",
                "namespace": "scn-sessions",
                "resourceVersion": "22597",
                "uid": "6cb66fb0-1e28-4d48-a4f8-802b3eaaf9a8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:34Z",
            "message": "Error: ImagePullBackOff",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:34Z",
                "name": "session-cache-6bb4646854-vzbwl.18eb1b312c59187b",
                "namespace": "scn-sessions",
                "resourceVersion": "22610",
                "uid": "4023a4f2-e3a5-4bf1-998e-6deb66871218"
            },
            "reason": "Failed",
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
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "session-cache-6bb4646854",
                "namespace": "scn-sessions",
                "resourceVersion": "22592",
                "uid": "3c17e9df-c0c1-4b2f-9c61-3861f4b15845"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Created pod: session-cache-6bb4646854-vzbwl",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache-6bb4646854.18eb1b30db811998",
                "namespace": "scn-sessions",
                "resourceVersion": "22596",
                "uid": "c143d89f-dd72-41cf-a1ab-a9305ab3c482"
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
            "firstTimestamp": "2026-11-24T22:19:33Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "session-cache",
                "namespace": "scn-sessions",
                "resourceVersion": "22591",
                "uid": "70069073-3e5e-449b-8f73-cdb1747e4801"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:33Z",
            "message": "Scaled up replica set session-cache-6bb4646854 from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:33Z",
                "name": "session-cache.18eb1b30daa48348",
                "namespace": "scn-sessions",
                "resourceVersion": "22593",
                "uid": "8a4d4f06-d15a-4206-bacf-91e0c0940b05"
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

### 8. tool call: `k8s_get_resource_yaml`

```json
{"namespace":"scn-sessions","resource_name":"session-cache","resource_type":"deployment"}
```

### 9. tool result: `k8s_get_resource_yaml`

```text
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"session-cache"},"name":"session-cache","namespace":"scn-sessions"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"session-cache"}},"template":{"metadata":{"labels":{"app":"session-cache"}},"spec":{"containers":[{"image":"docker.io/library/redis:7.4-alpine","name":"redis","ports":[{"containerPort":6379}],"resources":{"limits":{"memory":"64Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}}],"terminationGracePeriodSeconds":1}}}}
  creationTimestamp: "2026-11-24T22:19:33Z"
  generation: 1
  labels:
    app: session-cache
  name: session-cache
  namespace: scn-sessions
  resourceVersion: "22603"
  uid: 70069073-3e5e-449b-8f73-cdb1747e4801
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: session-cache
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: session-cache
    spec:
      containers:
      - image: docker.io/library/redis:7.4-alpine
        imagePullPolicy: IfNotPresent
        name: redis
        ports:
        - containerPort: 6379
          protocol: TCP
        resources:
          limits:
            memory: 64Mi
          requests:
            cpu: 10m
            memory: 16Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 1
status:
  conditions:
  - lastTransitionTime: "2026-11-24T22:19:33Z"
    lastUpdateTime: "2026-11-24T22:19:33Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: "2026-11-24T22:19:33Z"
    lastUpdateTime: "2026-11-24T22:19:33Z"
    message: ReplicaSet "session-cache-6bb4646854" is progressing.
    reason: ReplicaSetUpdated
    status: "True"
    type: Progressing
  observedGeneration: 1
  replicas: 1
  terminatingReplicas: 0
  unavailableReplicas: 1
  updatedReplicas: 1
```

### 10. agent

Based on my investigation, the pod `session-cache-6bb4646854-vzbwl` is stuck in `ErrImagePull` state. The events clearly show that Kubernetes attempted to pull the image `docker.io/library/redis:7.4-alpine` but received a "NotFound" error - the image cannot be resolved or pulled from the registry.

The deployment spec confirms the container uses this exact image, and the pod is on node `airgap-sre-control-plane`. The scheduler successfully placed the pod (no scheduling issues), but the kubelet failed to pull the container image.

**Root cause:** The image `docker.io/library/redis:7.4-alpine` is not available in the registry or cannot be pulled by the cluster's image registry.

```json
{
  "root_cause_category": "image_pull_failure",
  "resources": [
    {"kind": "Pod", "name": "session-cache-6bb4646854-vzbwl"},
    {"kind": "Deployment", "name": "session-cache"},
    {"kind": "Image", "name": "docker.io/library/redis:7.4-alpine"}
  ],
  "summary": "The pod cannot start because the container image docker.io/library/redis:7.4-alpine cannot be pulled from the registry."
}
```
