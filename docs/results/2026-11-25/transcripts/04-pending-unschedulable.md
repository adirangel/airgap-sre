# 4. pending-unschedulable — Pod Pending from an unsatisfiable nodeSelector

- Run: 2026-11-25 00:16 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `unschedulable`, agent said `unschedulable`
- Must name Deployment `batch-worker`: named
- Must name NodeLabel `disktype`: named
- Agent time: 26s, tool calls: 11, tokens: 47933

## Symptom given to the agent

> The batch-worker pods in namespace scn-batch never start. Find the root cause.

## Final answer

Based on my investigation, here's what I found:

**What is broken:** The batch-worker pod (`batch-worker-97cf85589-9qdq8`) is stuck in "Pending" status and cannot be scheduled.

**Evidence:**
- The event from the default-scheduler shows: `0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector`
- The Deployment `batch-worker` has a `nodeSelector` configured with `disktype: ssd`
- The pod description confirms it inherits this selector via `spec.template.spec.nodeSelector.disktype: ssd`

**Root cause:** The scheduler cannot place the pod because no nodes in the cluster match the required node selector `disktype=ssd`. This is a classic unschedulable scenario caused by a mismatch between the pod's scheduling requirements and available node labels.

```json
{
  "root_cause_category": "unschedulable",
  "resources": [
    {"kind": "Pod", "name": "batch-worker-97cf85589-9qdq8"},
    {"kind": "Deployment", "name": "batch-worker", "namespace": "scn-batch"}
  ],
  "summary": "The pod cannot be scheduled because the nodeSelector disktype=ssd does not match any available nodes in the cluster."
}
```

## Full transcript

### 1. user

The batch-worker pods in namespace scn-batch never start. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-batch","output":"wide","resource_type":"pod"}
```

### 3. tool call: `k8s_get_events`

```json
{"namespace":"scn-batch"}
```

### 4. tool result: `k8s_get_resources`

```text
NAME                           READY   STATUS    RESTARTS   AGE   IP       NODE     NOMINATED NODE   READINESS GATES
batch-worker-97cf85589-9qdq8   0/1     Pending   0          3s    <none>   <none>   <none>           <none>
```

### 5. tool result: `k8s_get_events`

```text
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:53Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "batch-worker-97cf85589-9qdq8",
                "namespace": "scn-batch",
                "resourceVersion": "22676",
                "uid": "fc201b64-adf5-42ac-b4f9-439c9a419e35"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:53Z",
            "message": "0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector. preemption: 0/1 nodes are available: 1 Preemption is not helpful for scheduling.",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:53Z",
                "name": "batch-worker-97cf85589-9qdq8.18eb1b35b69a2c2d",
                "namespace": "scn-batch",
                "resourceVersion": "22678",
                "uid": "d89931ec-73f7-4ac3-bbb6-e930255e3b96"
            },
            "reason": "FailedScheduling",
            "reportingComponent": "default-scheduler",
            "reportingInstance": "",
            "source": {
                "component": "default-scheduler"
            },
            "type": "Warning"
        },
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:19:53Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "batch-worker-97cf85589",
                "namespace": "scn-batch",
                "resourceVersion": "22673",
                "uid": "4f3f8fc1-1688-4935-b6e0-dc87194b7dcc"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:53Z",
            "message": "Created pod: batch-worker-97cf85589-9qdq8",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:53Z",
                "name": "batch-worker-97cf85589.18eb1b35b6968bbb",
                "namespace": "scn-batch",
                "resourceVersion": "22677",
                "uid": "5f398ef2-f9a0-4aca-9ce5-04ab6a020daf"
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
            "firstTimestamp": "2026-11-24T22:19:53Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "batch-worker",
                "namespace": "scn-batch",
                "resourceVersion": "22672",
                "uid": "cd8d073f-4b82-446f-b1b6-b7b0e9dc13ce"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:19:53Z",
            "message": "Scaled up replica set batch-worker-97cf85589 from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:19:53Z",
                "name": "batch-worker.18eb1b35b5dbdb5f",
                "namespace": "scn-batch",
                "resourceVersion": "22675",
                "uid": "adafde15-27a1-4363-90b0-0e0bd486f621"
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

### 6. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-batch","resource_name":"batch-worker-97cf85589-9qdq8","resource_type":"pod"}
```

### 7. tool call: `k8s_get_resource_yaml`

```json
{"namespace":"scn-batch","resource_name":"batch-worker","resource_type":"deployment"}
```

### 8. tool result: `k8s_describe_resource`

```text
Name:             batch-worker-97cf85589-9qdq8
Namespace:        scn-batch
Priority:         0
Service Account:  default
Node:             <none>
Labels:           app=batch-worker
                  pod-template-hash=97cf85589
Annotations:      <none>
Status:           Pending
IP:               
IPs:              <none>
Controlled By:    ReplicaSet/batch-worker-97cf85589
Containers:
  worker:
    Image:      docker.io/library/busybox:1.37.0
    Port:       <none>
    Host Port:  <none>
    Command:
      sh
      -c
      echo batch-worker ready; while true; do sleep 3600; done
    Limits:
      memory:  32Mi
    Requests:
      cpu:        10m
      memory:     16Mi
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-gfkmj (ro)
Conditions:
  Type           Status
  PodScheduled   False 
Volumes:
  kube-api-access-gfkmj:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    Optional:                false
    DownwardAPI:             true
QoS Class:                   Burstable
Node-Selectors:              disktype=ssd
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  5s    default-scheduler  0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector. preemption: 0/1 nodes are available: 1 Preemption is not helpful for scheduling.
```

### 9. tool result: `k8s_get_resource_yaml`

```text
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"batch-worker"},"name":"batch-worker","namespace":"scn-batch"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"batch-worker"}},"template":{"metadata":{"labels":{"app":"batch-worker"}},"spec":{"containers":[{"command":["sh","-c","echo batch-worker ready; while true; do sleep 3600; done"],"image":"docker.io/library/busybox:1.37.0","name":"worker","resources":{"limits":{"memory":"32Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}}],"nodeSelector":{"disktype":"ssd"},"terminationGracePeriodSeconds":1}}}}
  creationTimestamp: "2026-11-24T22:19:53Z"
  generation: 1
  labels:
    app: batch-worker
  name: batch-worker
  namespace: scn-batch
  resourceVersion: "22683"
  uid: cd8d073f-4b82-446f-b1b6-b7b0e9dc13ce
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: batch-worker
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: batch-worker
    spec:
      containers:
      - command:
        - sh
        - -c
        - echo batch-worker ready; while true; do sleep 3600; done
        image: docker.io/library/busybox:1.37.0
        imagePullPolicy: IfNotPresent
        name: worker
        resources:
          limits:
            memory: 32Mi
          requests:
            cpu: 10m
            memory: 16Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      nodeSelector:
        disktype: ssd
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 1
status:
  conditions:
  - lastTransitionTime: "2026-11-24T22:19:53Z"
    lastUpdateTime: "2026-11-24T22:19:53Z"
    message: Deployment does not have minimum availability.
    reason: MinimumReplicasUnavailable
    status: "False"
    type: Available
  - lastTransitionTime: "2026-11-24T22:19:53Z"
    lastUpdateTime: "2026-11-24T22:19:53Z"
    message: ReplicaSet "batch-worker-97cf85589" is progressing.
    reason: ReplicaSetUpdated
    status: "True"
    type: Progressing
  observedGeneration: 1
  replicas: 1
  terminatingReplicas: 0
  unavailableReplicas: 1
  updatedReplicas: 1
```

### 10. tool call: `k8s_get_resources`

```json
{"output":"wide","resource_type":"node"}
```

### 11. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node -o wide failed: exit status 1
```

### 12. tool call: `k8s_get_resources`

```json
{"all_namespaces":true,"output":"wide","resource_type":"node"}
```

### 13. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node --all-namespaces -o wide failed: exit status 1
```

### 14. tool call: `k8s_get_resources`

```json
{"output":"wide","resource_type":"node"}
```

### 15. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node -o wide failed: exit status 1
```

### 16. tool call: `k8s_get_resources`

```json
{"output":"json","resource_type":"node"}
```

### 17. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node -o json failed: exit status 1
```

### 18. tool call: `k8s_get_resources`

```json
{"output":"json","resource_type":"node"}
```

### 19. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node -o json failed: exit status 1
```

### 20. tool call: `k8s_get_resources`

```json
{"output":"yaml","resource_type":"node"}
```

### 21. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: [Kubernetes] get node -o yaml failed: exit status 1
```

### 22. tool call: `k8s_get_resources`

```json
{"output":"wide"}
```

### 23. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: resource_type parameter is required
```

### 24. agent

Based on my investigation, here's what I found:

**What is broken:** The batch-worker pod (`batch-worker-97cf85589-9qdq8`) is stuck in "Pending" status and cannot be scheduled.

**Evidence:**
- The event from the default-scheduler shows: `0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector`
- The Deployment `batch-worker` has a `nodeSelector` configured with `disktype: ssd`
- The pod description confirms it inherits this selector via `spec.template.spec.nodeSelector.disktype: ssd`

**Root cause:** The scheduler cannot place the pod because no nodes in the cluster match the required node selector `disktype=ssd`. This is a classic unschedulable scenario caused by a mismatch between the pod's scheduling requirements and available node labels.

```json
{
  "root_cause_category": "unschedulable",
  "resources": [
    {"kind": "Pod", "name": "batch-worker-97cf85589-9qdq8"},
    {"kind": "Deployment", "name": "batch-worker", "namespace": "scn-batch"}
  ],
  "summary": "The pod cannot be scheduled because the nodeSelector disktype=ssd does not match any available nodes in the cluster."
}
```
