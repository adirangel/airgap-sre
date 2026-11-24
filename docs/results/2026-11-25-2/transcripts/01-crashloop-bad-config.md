# 1. crashloop-bad-config — CrashLoopBackOff from a bad config value

- Run: 2026-11-25 00:23 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `bad_config`, agent said `bad_config`
- Must name Deployment `checkout`: named, also in the JSON block
- Must name ConfigMap `checkout-config`: named, also in the JSON block
- Must name ConfigKey `DB_PORT`: named, also in the JSON block
- Agent time: 14s, tool calls: 5, tokens: 18825

## Symptom given to the agent

> The checkout service in namespace scn-checkout is down. Find the root cause.

## Final answer

The checkout service is down because the pod is in CrashLoopBackOff state. The container starts but immediately exits with exit code 1 due to a configuration validation failure.

**Evidence:**
- The pod `checkout-69f48dc5fd-xj7rj` shows status `CrashLoopBackOff` with restart count of 1
- The container command validates that `DB_PORT` must be a valid port number (digits only) using a case statement: `case "${DB_PORT}" in ''|*[!0-9]*) ... exit 1 ;; esac`
- The ConfigMap `checkout-config` contains `DB_PORT: 5432a` which includes the non-numeric character 'a'
- This causes the validation to fail and the container to exit with code 1

**Root cause:** The `DB_PORT` value in the ConfigMap is malformed - it should be a numeric port number but contains "5432a" instead of "5432".

```json
{
  "root_cause_category": "bad_config",
  "resources": [
    {"kind": "ConfigKey", "name": "checkout-config/DB_PORT"},
    {"kind": "Pod", "name": "checkout-69f48dc5fd-xj7rj"}
  ],
  "summary": "The checkout pod crashes because the DB_PORT configuration value '5432a' contains a non-numeric character, causing the application's validation to fail."
}
```

## Full transcript

### 1. user

The checkout service in namespace scn-checkout is down. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-checkout","output":"wide","resource_type":"pod"}
```

### 3. tool result: `k8s_get_resources`

```text
NAME                        READY   STATUS             RESTARTS     AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
checkout-69f48dc5fd-xj7rj   0/1     CrashLoopBackOff   1 (5s ago)   6s    10.244.0.32   airgap-sre-control-plane   <none>           <none>
```

### 4. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-checkout","resource_name":"checkout-69f48dc5fd-xj7rj","resource_type":"pod"}
```

### 5. tool call: `k8s_get_events`

```json
{"namespace":"scn-checkout"}
```

### 6. tool result: `k8s_describe_resource`

```text
Name:             checkout-69f48dc5fd-xj7rj
Namespace:        scn-checkout
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:23:01 +0000
Labels:           app=checkout
                  pod-template-hash=69f48dc5fd
Annotations:      <none>
Status:           Running
IP:               10.244.0.32
IPs:
  IP:           10.244.0.32
Controlled By:  ReplicaSet/checkout-69f48dc5fd
Containers:
  checkout:
    Container ID:  containerd://e0c82add86407c81581514c36f6119c9bc16a0c4924d587035821ba1cc3a6035
    Image:         docker.io/library/busybox:1.37.0
    Image ID:      docker.io/library/busybox@sha256:66a6306db78bf2dbf3487f293aa8d6990d8e506fdffab9cc43fe422becf886e4
    Port:          <none>
    Host Port:     <none>
    Command:
      sh
      -c
      echo "checkout starting; db=${DB_HOST}:${DB_PORT} log=${LOG_LEVEL}"
      case "${DB_PORT}" in
        ''|*[!0-9]*)
          echo "FATAL: invalid configuration: DB_PORT='${DB_PORT}' is not a valid port number" >&2
          exit 1
          ;;
      esac
      echo "checkout ready"
      while true; do sleep 3600; done
      
    State:          Waiting
      Reason:       CrashLoopBackOff
    Last State:     Terminated
      Reason:       Error
      Exit Code:    1
      Started:      Tue, 24 Nov 2026 22:23:02 +0000
      Finished:     Tue, 24 Nov 2026 22:23:02 +0000
    Ready:          False
    Restart Count:  1
    Limits:
      memory:  64Mi
    Requests:
      cpu:     10m
      memory:  16Mi
    Environment Variables from:
      checkout-config  ConfigMap  Optional: false
    Environment:       <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-6kngn (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-6kngn:
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
  Normal   Scheduled  8s               default-scheduler  Successfully assigned scn-checkout/checkout-69f48dc5fd-xj7rj to airgap-sre-control-plane
  Normal   Pulled     7s (x2 over 8s)  kubelet            spec.containers{checkout}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal   Created    7s (x2 over 8s)  kubelet            spec.containers{checkout}: Container created
  Normal   Started    7s (x2 over 8s)  kubelet            spec.containers{checkout}: Container started
  Warning  BackOff    5s (x2 over 6s)  kubelet            spec.containers{checkout}: Back-off restarting failed container checkout in pod checkout-69f48dc5fd-xj7rj_scn-checkout(d04669cc-8e1e-4f78-81da-50de9118f8e4)
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "checkout-69f48dc5fd-xj7rj",
                "namespace": "scn-checkout",
                "resourceVersion": "23103",
                "uid": "d04669cc-8e1e-4f78-81da-50de9118f8e4"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:01Z",
            "message": "Successfully assigned scn-checkout/checkout-69f48dc5fd-xj7rj to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout-69f48dc5fd-xj7rj.18eb1b6155aa1b82",
                "namespace": "scn-checkout",
                "resourceVersion": "23108",
                "uid": "56fed3aa-c191-4a31-9ddf-aea42ab1eb62"
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{checkout}",
                "kind": "Pod",
                "name": "checkout-69f48dc5fd-xj7rj",
                "namespace": "scn-checkout",
                "resourceVersion": "23105",
                "uid": "d04669cc-8e1e-4f78-81da-50de9118f8e4"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:02Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout-69f48dc5fd-xj7rj.18eb1b6170fdfa25",
                "namespace": "scn-checkout",
                "resourceVersion": "23121",
                "uid": "fb0ebc35-34ed-46b1-b804-fa1bbe90375f"
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{checkout}",
                "kind": "Pod",
                "name": "checkout-69f48dc5fd-xj7rj",
                "namespace": "scn-checkout",
                "resourceVersion": "23105",
                "uid": "d04669cc-8e1e-4f78-81da-50de9118f8e4"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:02Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout-69f48dc5fd-xj7rj.18eb1b6173990b8f",
                "namespace": "scn-checkout",
                "resourceVersion": "23125",
                "uid": "1bea27ab-6471-4443-a410-12f3e951c25e"
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{checkout}",
                "kind": "Pod",
                "name": "checkout-69f48dc5fd-xj7rj",
                "namespace": "scn-checkout",
                "resourceVersion": "23105",
                "uid": "d04669cc-8e1e-4f78-81da-50de9118f8e4"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:02Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout-69f48dc5fd-xj7rj.18eb1b6177b0dc8c",
                "namespace": "scn-checkout",
                "resourceVersion": "23126",
                "uid": "9421746c-a458-47b7-97ff-67d771aa42d1"
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
            "firstTimestamp": "2026-11-24T22:23:03Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{checkout}",
                "kind": "Pod",
                "name": "checkout-69f48dc5fd-xj7rj",
                "namespace": "scn-checkout",
                "resourceVersion": "23105",
                "uid": "d04669cc-8e1e-4f78-81da-50de9118f8e4"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:04Z",
            "message": "Back-off restarting failed container checkout in pod checkout-69f48dc5fd-xj7rj_scn-checkout(d04669cc-8e1e-4f78-81da-50de9118f8e4)",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:03Z",
                "name": "checkout-69f48dc5fd-xj7rj.18eb1b61eb29d8e8",
                "namespace": "scn-checkout",
                "resourceVersion": "23136",
                "uid": "cd263463-083d-4e22-aebc-1df418c14d9c"
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "checkout-69f48dc5fd",
                "namespace": "scn-checkout",
                "resourceVersion": "23100",
                "uid": "078573a4-5486-4e6c-b6e8-5f84b207e39c"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:01Z",
            "message": "Created pod: checkout-69f48dc5fd-xj7rj",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout-69f48dc5fd.18eb1b6154ee9b12",
                "namespace": "scn-checkout",
                "resourceVersion": "23104",
                "uid": "c49a9569-d598-4766-8f53-5702423d6197"
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
            "firstTimestamp": "2026-11-24T22:23:01Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "checkout",
                "namespace": "scn-checkout",
                "resourceVersion": "23099",
                "uid": "4613d8c1-c8cc-4153-a445-b6c895f9a864"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:23:01Z",
            "message": "Scaled up replica set checkout-69f48dc5fd from 0 to 1",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:23:01Z",
                "name": "checkout.18eb1b6154241775",
                "namespace": "scn-checkout",
                "resourceVersion": "23101",
                "uid": "431abddf-c8e5-4e12-a108-e987a67eb183"
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
{"namespace":"scn-checkout","resource_name":"checkout-config","resource_type":"configmap"}
```

### 9. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-checkout","resource_name":"checkout","resource_type":"deployment"}
```

### 10. tool result: `k8s_get_resource_yaml`

```text
apiVersion: v1
data:
  DB_HOST: orders-db.internal
  DB_PORT: 5432a
  LOG_LEVEL: info
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"DB_HOST":"orders-db.internal","DB_PORT":"5432a","LOG_LEVEL":"info"},"kind":"ConfigMap","metadata":{"annotations":{},"name":"checkout-config","namespace":"scn-checkout"}}
  creationTimestamp: "2026-11-24T22:23:01Z"
  name: checkout-config
  namespace: scn-checkout
  resourceVersion: "23098"
  uid: 786dd3b4-c146-4aec-b790-2fcca92830f9
```

### 11. tool result: `k8s_describe_resource`

```text
Name:                   checkout
Namespace:              scn-checkout
CreationTimestamp:      Tue, 24 Nov 2026 22:23:01 +0000
Labels:                 app=checkout
Annotations:            deployment.kubernetes.io/revision: 1
Selector:               app=checkout
Replicas:               1 desired | 1 updated | 1 total | 0 available | 1 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=checkout
  Containers:
   checkout:
    Image:      docker.io/library/busybox:1.37.0
    Port:       <none>
    Host Port:  <none>
    Command:
      sh
      -c
      echo "checkout starting; db=${DB_HOST}:${DB_PORT} log=${LOG_LEVEL}"
      case "${DB_PORT}" in
        ''|*[!0-9]*)
          echo "FATAL: invalid configuration: DB_PORT='${DB_PORT}' is not a valid port number" >&2
          exit 1
          ;;
      esac
      echo "checkout ready"
      while true; do sleep 3600; done
      
    Limits:
      memory:  64Mi
    Requests:
      cpu:     10m
      memory:  16Mi
    Environment Variables from:
      checkout-config  ConfigMap  Optional: false
    Environment:       <none>
    Mounts:            <none>
  Volumes:             <none>
  Node-Selectors:      <none>
  Tolerations:         <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Progressing    True    NewReplicaSetAvailable
  Available      False   MinimumReplicasUnavailable
OldReplicaSets:  <none>
NewReplicaSet:   checkout-69f48dc5fd (1/1 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  12s   deployment-controller  Scaled up replica set checkout-69f48dc5fd from 0 to 1
```

### 12. agent

The checkout service is down because the pod is in CrashLoopBackOff state. The container starts but immediately exits with exit code 1 due to a configuration validation failure.

**Evidence:**
- The pod `checkout-69f48dc5fd-xj7rj` shows status `CrashLoopBackOff` with restart count of 1
- The container command validates that `DB_PORT` must be a valid port number (digits only) using a case statement: `case "${DB_PORT}" in ''|*[!0-9]*) ... exit 1 ;; esac`
- The ConfigMap `checkout-config` contains `DB_PORT: 5432a` which includes the non-numeric character 'a'
- This causes the validation to fail and the container to exit with code 1

**Root cause:** The `DB_PORT` value in the ConfigMap is malformed - it should be a numeric port number but contains "5432a" instead of "5432".

```json
{
  "root_cause_category": "bad_config",
  "resources": [
    {"kind": "ConfigKey", "name": "checkout-config/DB_PORT"},
    {"kind": "Pod", "name": "checkout-69f48dc5fd-xj7rj"}
  ],
  "summary": "The checkout pod crashes because the DB_PORT configuration value '5432a' contains a non-numeric character, causing the application's validation to fail."
}
```
