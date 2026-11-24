# 3. imagepull-not-mirrored — ImagePullBackOff because the image was never mirrored

- Run: 2026-11-25 00:23 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `image_pull_failure`, agent said `image_pull_failure`
- Must name Deployment `session-cache`: named, also in the JSON block
- Must name Image `redis:7.4-alpine`: named, also in the JSON block
- Agent time: 16s, tool calls: 5, tokens: 19595

## Symptom given to the agent

> The session-cache workload in namespace scn-sessions never starts. Find the root cause.

## Final answer

The pod `session-cache-6bb4646854-z4v7k` in namespace `scn-sessions` is stuck in `ErrImagePull` state. The events show that Kubernetes attempted to pull the image `docker.io/library/redis:7.4-alpine` but received a "NotFound" error, meaning the image cannot be resolved or pulled from the registry.

**Root cause:** The container image `docker.io/library/redis:7.4-alpine` is not available in the cluster's image registry (or the registry is unreachable).

```json
{
  "root_cause_category": "image_pull_failure",
  "resources": [
    {"kind": "Image", "name": "docker.io/library/redis:7.4-alpine"},
    {"kind": "Pod", "name": "session-cache-6bb4646854-z4v7k"},
    {"kind": "Deployment", "name": "session-cache"}
  ],
  "summary": "The pod cannot start because the image docker.io/library/redis:7.4-alpine cannot be pulled from the registry."
}
```

## Full transcript

### 1. user

The session-cache workload in namespace scn-sessions never starts. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-sessions","resource_type":"pod"}
```

### 3. tool result: `k8s_get_resources`

```text
NAME                             READY   STATUS         RESTARTS   AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
session-cache-6bb4646854-z4v7k   0/1     ErrImagePull   0          4s    10.244.0.35   airgap-sre-control-plane   <none>           <none>
```

### 4. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-sessions","resource_name":"session-cache-6bb4646854-z4v7k","resource_type":"pod"}
```

### 5. tool result: `k8s_describe_resource`

```text
Name:             session-cache-6bb4646854-z4v7k
Namespace:        scn-sessions
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:23:58 +0000
Labels:           app=session-cache
                  pod-template-hash=6bb4646854
Annotations:      <none>
Status:           Pending
IP:               10.244.0.35
IPs:
  IP:           10.244.0.35
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
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-p96hc (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-p96hc:
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
  Type     Reason     Age   From               Message
  ----     ------     ----  ----               -------
  Normal   Scheduled  5s    default-scheduler  Successfully assigned scn-sessions/session-cache-6bb4646854-z4v7k to airgap-sre-control-plane
  Normal   Pulling    5s    kubelet            spec.containers{redis}: Pulling image "docker.io/library/redis:7.4-alpine"
  Warning  Failed     5s    kubelet            spec.containers{redis}: Failed to pull image "docker.io/library/redis:7.4-alpine": rpc error: code = NotFound desc = failed to pull and unpack image "docker.io/library/redis:7.4-alpine": failed to resolve reference "docker.io/library/redis:7.4-alpine": docker.io/library/redis:7.4-alpine: not found
  Warning  Failed     5s    kubelet            spec.containers{redis}: Error: ErrImagePull
  Normal   BackOff    4s    kubelet            spec.containers{redis}: Back-off pulling image "docker.io/library/redis:7.4-alpine"
  Warning  Failed     4s    kubelet            spec.containers{redis}: Error: ImagePullBackOff
```

### 6. tool call: `k8s_get_resources`

```json
{"namespace":"scn-sessions","resource_type":"deployment"}
```

### 7. tool result: `k8s_get_resources`

```text
NAME            READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS   IMAGES                               SELECTOR
session-cache   0/1     1            0           7s    redis        docker.io/library/redis:7.4-alpine   app=session-cache
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
  creationTimestamp: "2026-11-24T22:23:58Z"
  generation: 1
  labels:
    app: session-cache
  name: session-cache
  namespace: scn-sessions
  resourceVersion: "23371"
  uid: 923ac15a-5ee3-406b-ab26-e219ad381e8f
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
  - lastTransitionTime: "2026-11-24T22:23:58Z"
    lastUpdateTime: "2026-11-24T22:23:58Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: "2026-11-24T22:23:58Z"
    lastUpdateTime: "2026-11-24T22:23:58Z"
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

### 10. tool call: `k8s_get_resource_yaml`

```json
{"namespace":"scn-sessions","resource_name":"session-cache","resource_type":"deployment"}
```

### 11. tool result: `k8s_get_resource_yaml`

```text
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"session-cache"},"name":"session-cache","namespace":"scn-sessions"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"session-cache"}},"template":{"metadata":{"labels":{"app":"session-cache"}},"spec":{"containers":[{"image":"docker.io/library/redis:7.4-alpine","name":"redis","ports":[{"containerPort":6379}],"resources":{"limits":{"memory":"64Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}}],"terminationGracePeriodSeconds":1}}}}
  creationTimestamp: "2026-11-24T22:23:58Z"
  generation: 1
  labels:
    app: session-cache
  name: session-cache
  namespace: scn-sessions
  resourceVersion: "23371"
  uid: 923ac15a-5ee3-406b-ab26-e219ad381e8f
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
  - lastTransitionTime: "2026-11-24T22:23:58Z"
    lastUpdateTime: "2026-11-24T22:23:58Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: "2026-11-24T22:23:58Z"
    lastUpdateTime: "2026-11-24T22:23:58Z"
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

### 12. agent

The pod `session-cache-6bb4646854-z4v7k` in namespace `scn-sessions` is stuck in `ErrImagePull` state. The events show that Kubernetes attempted to pull the image `docker.io/library/redis:7.4-alpine` but received a "NotFound" error, meaning the image cannot be resolved or pulled from the registry.

**Root cause:** The container image `docker.io/library/redis:7.4-alpine` is not available in the cluster's image registry (or the registry is unreachable).

```json
{
  "root_cause_category": "image_pull_failure",
  "resources": [
    {"kind": "Image", "name": "docker.io/library/redis:7.4-alpine"},
    {"kind": "Pod", "name": "session-cache-6bb4646854-z4v7k"},
    {"kind": "Deployment", "name": "session-cache"}
  ],
  "summary": "The pod cannot start because the image docker.io/library/redis:7.4-alpine cannot be pulled from the registry."
}
```
