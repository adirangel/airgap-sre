# 4. pending-unschedulable — Pod Pending from an unsatisfiable nodeSelector

- Run: 2026-11-25 00:23 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `unschedulable`, agent said `unschedulable`
- Must name Deployment `batch-worker`: named, also in the JSON block
- Must name NodeLabel `disktype`: named, also in the JSON block
- Agent time: 14s, tool calls: 5, tokens: 20575

## Symptom given to the agent

> The batch-worker pods in namespace scn-batch never start. Find the root cause.

## Final answer

The root cause is clear from the evidence:

**What's broken:** The pod `batch-worker-97cf85589-29cq6` remains in `Pending` status because it cannot be scheduled on any node.

**Evidence:**
1. Pod status shows `0/1 READY` with `STATUS` = `Pending`
2. Events show `FailedScheduling` with message: "0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector"
3. The deployment `batch-worker` has a `Node-Selector: disktype=ssd`
4. The only node in the cluster (`airgap-sre-control-plane`) does NOT have the label `disktype=ssd`

**Root cause:** The pod cannot be scheduled because its node selector requires `disktype=ssd`, but no nodes in the cluster have this label.

```json
{
  "root_cause_category": "unschedulable",
  "resources": [
    {"kind": "Pod", "name": "batch-worker-97cf85589-29cq6"},
    {"kind": "Deployment", "name": "batch-worker"},
    {"kind": "Node", "name": "airgap-sre-control-plane"}
  ],
  "summary": "The pod cannot be scheduled because its node selector requires disktype=ssd, but no nodes have this label."
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
batch-worker-97cf85589-29cq6   0/1     Pending   0          3s    <none>   <none>   <none>           <none>
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
            "firstTimestamp": "2026-11-24T22:24:23Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "batch-worker-97cf85589-29cq6",
                "namespace": "scn-batch",
                "resourceVersion": "23457",
                "uid": "7ec3ca4c-618e-4ad6-ab04-6b61a08343b8"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:24:23Z",
            "message": "0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector. preemption: 0/1 nodes are available: 1 Preemption is not helpful for scheduling.",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:24:23Z",
                "name": "batch-worker-97cf85589-29cq6.18eb1b748e4629e5",
                "namespace": "scn-batch",
                "resourceVersion": "23460",
                "uid": "94a750bc-118f-4f27-9f2b-55926b3891a7"
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
            "firstTimestamp": "2026-11-24T22:24:23Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "batch-worker-97cf85589",
                "namespace": "scn-batch",
                "resourceVersion": "23454",
                "uid": "0d8e78b5-089d-4cc1-a4f9-6c43b303824b"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:24:23Z",
            "message": "Created pod: batch-worker-97cf85589-29cq6",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:24:23Z",
                "name": "batch-worker-97cf85589.18eb1b748e3d8844",
                "namespace": "scn-batch",
                "resourceVersion": "23458",
                "uid": "c7a5e3dc-1e02-41a9-89be-c37094b20830"
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
            "firstTimestamp": "2026-11-24T22:24:23Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "batch-worker",
                "namespace": "scn-batch",
                "resourceVersion": "23453",
                "uid": "b10fb94c-e0de-4467-ae49-68b93c01c04f"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:24:23Z",
            "message": "Scaled up replica set batch-worker-97cf85589 from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:24:23Z",
                "name": "batch-worker.18eb1b748d55e79c",
                "namespace": "scn-batch",
                "resourceVersion": "23455",
                "uid": "dd09c550-46d5-4f58-9b2c-76baa98f37ee"
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
{"namespace":"scn-batch","resource_name":"batch-worker","resource_type":"deployment"}
```

### 7. tool result: `k8s_describe_resource`

```text
Name:                   batch-worker
Namespace:              scn-batch
CreationTimestamp:      Tue, 24 Nov 2026 22:24:23 +0000
Labels:                 app=batch-worker
Annotations:            deployment.kubernetes.io/revision: 1
Selector:               app=batch-worker
Replicas:               1 desired | 1 updated | 1 total | 0 available | 1 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=batch-worker
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
      cpu:         10m
      memory:      16Mi
    Environment:   <none>
    Mounts:        <none>
  Volumes:         <none>
  Node-Selectors:  disktype=ssd
  Tolerations:     <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      False   MinimumReplicasUnavailable
  Progressing    True    ReplicaSetUpdated
OldReplicaSets:  <none>
NewReplicaSet:   batch-worker-97cf85589 (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  5s    deployment-controller  Scaled up replica set batch-worker-97cf85589 from 0 to 1
```

### 8. tool call: `k8s_get_resources`

```json
{"all_namespaces":true,"output":"wide","resource_type":"node"}
```

### 9. tool result: `k8s_get_resources`

```text
NAME                       STATUS   ROLES           AGE     VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE                       KERNEL-VERSION                             CONTAINER-RUNTIME
airgap-sre-control-plane   Ready    control-plane   4h13m   v1.37.0   172.18.0.2    <none>        Debian GNU/Linux 13 (trixie)   6.6.87.2-microsoft-standard-WSL2 (amd64)   containerd://2.3.4
```

### 10. tool call: `k8s_describe_resource`

```json
{"namespace":"","resource_name":"airgap-sre-control-plane","resource_type":"node"}
```

### 11. tool result: `k8s_describe_resource`

```text
Name:               airgap-sre-control-plane
Roles:              control-plane
Labels:             beta.kubernetes.io/arch=amd64
                    beta.kubernetes.io/os=linux
                    kubernetes.io/arch=amd64
                    kubernetes.io/hostname=airgap-sre-control-plane
                    kubernetes.io/os=linux
                    node-role.kubernetes.io/control-plane=
Annotations:        node.alpha.kubernetes.io/ttl: 0
                    volumes.kubernetes.io/controller-managed-attach-detach: true
CreationTimestamp:  Tue, 24 Nov 2026 18:10:36 +0000
Taints:             <none>
Unschedulable:      false
Lease:              Failed to get lease: leases.coordination.k8s.io "airgap-sre-control-plane" is forbidden: User "system:serviceaccount:kagent:kagent-tools" cannot get resource "leases" in API group "coordination.k8s.io" in the namespace "kube-node-lease"
Conditions:
  Type             Status  LastHeartbeatTime                 LastTransitionTime                Reason                       Message
  ----             ------  -----------------                 ------------------                ------                       -------
  MemoryPressure   False   Tue, 24 Nov 2026 22:22:12 +0000   Tue, 24 Nov 2026 18:10:35 +0000   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure     False   Tue, 24 Nov 2026 22:22:12 +0000   Tue, 24 Nov 2026 18:10:35 +0000   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure      False   Tue, 24 Nov 2026 22:22:12 +0000   Tue, 24 Nov 2026 18:10:35 +0000   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            True    Tue, 24 Nov 2026 22:22:12 +0000   Tue, 24 Nov 2026 18:10:56 +0000   KubeletReady                 kubelet is posting ready status
Addresses:
  InternalIP:  172.18.0.2
  Hostname:    airgap-sre-control-plane
Capacity:
  cpu:                24
  ephemeral-storage:  1081101176832
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             32710796Ki
  pods:               110
Allocatable:
  cpu:                24
  ephemeral-storage:  1081101176832
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             32710796Ki
  pods:               110
System Info:
  Machine ID:                 a92098ea1fb04099b6a0392cdda3086b
  System UUID:                a92098ea1fb04099b6a0392cdda3086b
  Boot ID:                    eeef8fff-ce7b-4697-a211-32144d725600
  Kernel Version:             6.6.87.2-microsoft-standard-WSL2
  OS Image:                   Debian GNU/Linux 13 (trixie)
  Operating System:           linux
  Architecture:               amd64
  Container Runtime Version:  containerd://2.3.4
  Kubelet Version:            v1.37.0
PodCIDR:                      10.244.0.0/24
PodCIDRs:                     10.244.0.0/24
ProviderID:                   kind://docker/airgap-sre/airgap-sre-control-plane
Non-terminated Pods:          (14 in total)
  Namespace                   Name                                                CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
  ---------                   ----                                                ------------  ----------  ---------------  -------------  ---
  kagent                      kagent-controller-7dff99bd87-5jw7s                  100m (0%)     2 (8%)      128Mi (0%)       512Mi (1%)     14m
  kagent                      kagent-postgresql-856b475f59-4cs5g                  250m (1%)     500m (2%)   256Mi (0%)       512Mi (1%)     14m
  kagent                      kagent-tools-58b9fc5cfc-52zql                       100m (0%)     1 (4%)      128Mi (0%)       512Mi (1%)     14m
  kagent                      kagent-ui-57444c488-qpfl7                           100m (0%)     1 (4%)      256Mi (0%)       1Gi (3%)       14m
  kagent                      sre-agent-57766988f-kck7g                           100m (0%)     2 (8%)      384Mi (1%)       1Gi (3%)       105s
  kube-system                 coredns-559f6c778d-wjpwz                            100m (0%)     0 (0%)      70Mi (0%)        170Mi (0%)     4h13m
  kube-system                 coredns-559f6c778d-xqf82                            100m (0%)     0 (0%)      70Mi (0%)        170Mi (0%)     4h13m
  kube-system                 etcd-airgap-sre-control-plane                       100m (0%)     0 (0%)      100Mi (0%)       0 (0%)         4h13m
  kube-system                 kindnet-qwq79                                       100m (0%)     0 (0%)      50Mi (0%)        0 (0%)         4h13m
  kube-system                 kube-apiserver-airgap-sre-control-plane             250m (1%)     0 (0%)      0 (0%)           0 (0%)         4h13m
  kube-system                 kube-controller-manager-airgap-sre-control-plane    200m (0%)     0 (0%)      0 (0%)           0 (0%)         4h13m
  kube-system                 kube-proxy-ldwjq                                    0 (0%)        0 (0%)      0 (0%)           0 (0%)         4h13m
  kube-system                 kube-scheduler-airgap-sre-control-plane             100m (0%)     0 (0%)      0 (0%)           0 (0%)         4h13m
  local-path-storage          local-path-provisioner-75f7fc7dc5-88rqw             0 (0%)        0 (0%)      0 (0%)           0 (0%)         4h13m
Allocated resources:
  (Total limits may be over 100 percent, i.e., overcommitted.)
  Resource           Requests     Limits
  --------           --------     ------
  cpu                1600m (6%)   6500m (27%)
  memory             1442Mi (4%)  3924Mi (12%)
  ephemeral-storage  0 (0%)       0 (0%)
  hugepages-1Gi      0 (0%)       0 (0%)
  hugepages-2Mi      0 (0%)       0 (0%)
Events:              <none>
```

### 12. agent

The root cause is clear from the evidence:

**What's broken:** The pod `batch-worker-97cf85589-29cq6` remains in `Pending` status because it cannot be scheduled on any node.

**Evidence:**
1. Pod status shows `0/1 READY` with `STATUS` = `Pending`
2. Events show `FailedScheduling` with message: "0/1 nodes are available: 1 node(s) didn't match Pod's node affinity/selector"
3. The deployment `batch-worker` has a `Node-Selector: disktype=ssd`
4. The only node in the cluster (`airgap-sre-control-plane`) does NOT have the label `disktype=ssd`

**Root cause:** The pod cannot be scheduled because its node selector requires `disktype=ssd`, but no nodes in the cluster have this label.

```json
{
  "root_cause_category": "unschedulable",
  "resources": [
    {"kind": "Pod", "name": "batch-worker-97cf85589-29cq6"},
    {"kind": "Deployment", "name": "batch-worker"},
    {"kind": "Node", "name": "airgap-sre-control-plane"}
  ],
  "summary": "The pod cannot be scheduled because its node selector requires disktype=ssd, but no nodes have this label."
}
```
