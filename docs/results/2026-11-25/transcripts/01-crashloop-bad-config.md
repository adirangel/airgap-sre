# 1. crashloop-bad-config — CrashLoopBackOff from a bad config value

- Run: 2026-11-25 00:16 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `bad_config`, agent said `bad_config`
- Must name Deployment `checkout`: named
- Must name ConfigMap `checkout-config`: named
- Must name ConfigKey `DB_PORT`: named
- Agent time: 11s, tool calls: 4, tokens: 14904

## Symptom given to the agent

> The checkout service in namespace scn-checkout is down. Find the root cause.

## Final answer

The checkout service is down because the pod is crashing repeatedly (CrashLoopBackOff). The container exits with code 1 due to an invalid configuration value for `DB_PORT`.

**Evidence:**
- Pod `checkout-69f48dc5fd-2qxll` shows status `CrashLoopBackOff` with 4 restarts
- Container logs show: `FATAL: invalid configuration: DB_PORT='5432a' is not a valid port number`
- The ConfigMap `checkout-config` contains `DB_PORT: 5432a`, which is not a valid numeric port

**Root cause:** The `DB_PORT` value in the ConfigMap is malformed (`5432a` instead of a numeric port like `5432`). This causes the application to exit immediately on startup.

```json
{
  "root_cause_category": "bad_config",
  "resources": [
    {"kind": "ConfigKey", "name": "checkout-config.DB_PORT"},
    {"kind": "ConfigMap", "name": "checkout-config"}
  ],
  "summary": "The checkout pod crashes because the DB_PORT configuration value '5432a' in ConfigMap checkout-config is not a valid numeric port number."
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
NAME                        READY   STATUS             RESTARTS      AGE     IP            NODE                       NOMINATED NODE   READINESS GATES
checkout-69f48dc5fd-2qxll   0/1     CrashLoopBackOff   4 (65s ago)   2m28s   10.244.0.25   airgap-sre-control-plane   <none>           <none>
```

### 4. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-checkout","resource_name":"checkout-69f48dc5fd-2qxll","resource_type":"pod"}
```

### 5. tool result: `k8s_describe_resource`

```text
Name:             checkout-69f48dc5fd-2qxll
Namespace:        scn-checkout
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:16:23 +0000
Labels:           app=checkout
                  pod-template-hash=69f48dc5fd
Annotations:      <none>
Status:           Running
IP:               10.244.0.25
IPs:
  IP:           10.244.0.25
Controlled By:  ReplicaSet/checkout-69f48dc5fd
Containers:
  checkout:
    Container ID:  containerd://8a11accec8bfa053ef83f5519190db4ef5d9471fe6126cef5761ef3b21d9f19b
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
      Started:      Tue, 24 Nov 2026 22:17:46 +0000
      Finished:     Tue, 24 Nov 2026 22:17:46 +0000
    Ready:          False
    Restart Count:  4
    Limits:
      memory:  64Mi
    Requests:
      cpu:     10m
      memory:  16Mi
    Environment Variables from:
      checkout-config  ConfigMap  Optional: false
    Environment:       <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-q8c5m (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       False 
  ContainersReady             False 
  PodScheduled                True 
Volumes:
  kube-api-access-q8c5m:
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
  Type     Reason     Age                  From               Message
  ----     ------     ----                 ----               -------
  Normal   Scheduled  2m29s                default-scheduler  Successfully assigned scn-checkout/checkout-69f48dc5fd-2qxll to airgap-sre-control-plane
  Normal   Pulled     66s (x5 over 2m28s)  kubelet            spec.containers{checkout}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal   Created    66s (x5 over 2m28s)  kubelet            spec.containers{checkout}: Container created
  Normal   Started    66s (x5 over 2m28s)  kubelet            spec.containers{checkout}: Container started
  Warning  BackOff    5s (x5 over 2m26s)   kubelet            spec.containers{checkout}: Back-off restarting failed container checkout in pod checkout-69f48dc5fd-2qxll_scn-checkout(7bbd4f48-7b26-406b-8ace-fed5a66649fe)
```

### 6. tool call: `k8s_get_pod_logs`

```json
{"container":"checkout","namespace":"scn-checkout","pod_name":"checkout-69f48dc5fd-2qxll"}
```

### 7. tool result: `k8s_get_pod_logs`

```text
FATAL: invalid configuration: DB_PORT='5432a' is not a valid port number
checkout starting; db=orders-db.internal:5432a log=info
```

### 8. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-checkout","resource_name":"checkout-config","resource_type":"configmap"}
```

### 9. tool result: `k8s_describe_resource`

```text
Name:         checkout-config
Namespace:    scn-checkout
Labels:       <none>
Annotations:  <none>

Data
====
DB_HOST:
----
orders-db.internal

DB_PORT:
----
5432a

LOG_LEVEL:
----
info


BinaryData
====

Events:  <none>
```

### 10. agent

The checkout service is down because the pod is crashing repeatedly (CrashLoopBackOff). The container exits with code 1 due to an invalid configuration value for `DB_PORT`.

**Evidence:**
- Pod `checkout-69f48dc5fd-2qxll` shows status `CrashLoopBackOff` with 4 restarts
- Container logs show: `FATAL: invalid configuration: DB_PORT='5432a' is not a valid port number`
- The ConfigMap `checkout-config` contains `DB_PORT: 5432a`, which is not a valid numeric port

**Root cause:** The `DB_PORT` value in the ConfigMap is malformed (`5432a` instead of a numeric port like `5432`). This causes the application to exit immediately on startup.

```json
{
  "root_cause_category": "bad_config",
  "resources": [
    {"kind": "ConfigKey", "name": "checkout-config.DB_PORT"},
    {"kind": "ConfigMap", "name": "checkout-config"}
  ],
  "summary": "The checkout pod crashes because the DB_PORT configuration value '5432a' in ConfigMap checkout-config is not a valid numeric port number."
}
```
