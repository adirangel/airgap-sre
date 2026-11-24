# 5. service-selector-mismatch — Service with a selector mismatch (no endpoints)

- Run: 2026-11-25 00:23 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `service_selector_mismatch`, agent said `service_selector_mismatch`
- Must name Service `storefront`: named, also in the JSON block
- Must name Deployment `shop-frontend`: named, also in the JSON block
- Agent time: 11s, tool calls: 4, tokens: 12340

## Symptom given to the agent

> Requests to storefront in namespace scn-shop fail with connection refused, although the application pods look healthy. Find the root cause.

## Final answer

I've found the root cause. The Service `storefront` has a selector of `app=shop-front`, but the application pods have the label `app=shop-frontend`. These labels don't match, so the Service has no endpoints and cannot route traffic to any pod.

**Evidence:**
- Service `storefront` in namespace `scn-shop` has selector: `app=shop-front`
- Pods `shop-frontend-69df64b78f-5qwhx` and `shop-frontend-69df64b78f-tvvb7` have label: `app=shop-frontend`
- Service `storefront` shows `Endpoints:` as empty

The root cause is a **label mismatch** between the Service selector and the pod labels.

```json
{
  "root_cause_category": "service_selector_mismatch",
  "resources": [
    {"kind": "Service", "name": "storefront"},
    {"kind": "Pod", "name": "shop-frontend-69df64b78f-5qwhx"}
  ],
  "summary": "The Service storefront selector app=shop-front does not match the pod label app=shop-frontend, resulting in no endpoints."
}
```

## Full transcript

### 1. user

Requests to storefront in namespace scn-shop fail with connection refused, although the application pods look healthy. Find the root cause.

### 2. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-shop","resource_name":"storefront","resource_type":"service"}
```

### 3. tool call: `k8s_get_resources`

```json
{"namespace":"scn-shop","output":"wide","resource_type":"pod"}
```

### 4. tool result: `k8s_describe_resource`

```text
Name:                     storefront
Namespace:                scn-shop
Labels:                   <none>
Annotations:              <none>
Selector:                 app=shop-front
Type:                     ClusterIP
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.96.84.51
IPs:                      10.96.84.51
Port:                     <unset>  80/TCP
TargetPort:               8080/TCP
Endpoints:                
Session Affinity:         None
Internal Traffic Policy:  Cluster
Events:                   <none>
```

### 5. tool result: `k8s_get_resources`

```text
NAME                             READY   STATUS    RESTARTS   AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
shop-frontend-69df64b78f-5qwhx   1/1     Running   0          5s    10.244.0.36   airgap-sre-control-plane   <none>           <none>
shop-frontend-69df64b78f-tvvb7   1/1     Running   0          5s    10.244.0.37   airgap-sre-control-plane   <none>           <none>
```

### 6. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-shop","resource_name":"shop-frontend-69df64b78f-5qwhx","resource_type":"pod"}
```

### 7. tool result: `k8s_describe_resource`

```text
Name:             shop-frontend-69df64b78f-5qwhx
Namespace:        scn-shop
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:24:43 +0000
Labels:           app=shop-frontend
                  pod-template-hash=69df64b78f
Annotations:      <none>
Status:           Running
IP:               10.244.0.36
IPs:
  IP:           10.244.0.36
Controlled By:  ReplicaSet/shop-frontend-69df64b78f
Containers:
  web:
    Container ID:  containerd://649654aac802b4a42ba5969d7f65528df70b8819713503bb67a69f590000f726
    Image:         docker.io/library/busybox:1.37.0
    Image ID:      docker.io/library/busybox@sha256:66a6306db78bf2dbf3487f293aa8d6990d8e506fdffab9cc43fe422becf886e4
    Port:          8080/TCP
    Host Port:     0/TCP
    Command:
      sh
      -c
      mkdir -p /www && echo "storefront ok" > /www/index.html
      exec httpd -f -p 8080 -h /www
      
    State:          Running
      Started:      Tue, 24 Nov 2026 22:24:44 +0000
    Ready:          True
    Restart Count:  0
    Limits:
      memory:  32Mi
    Requests:
      cpu:        10m
      memory:     16Mi
    Readiness:    http-get http://:8080/ delay=0s timeout=1s period=2s successThreshold=1 failureThreshold=3
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-rq4gw (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       True 
  ContainersReady             True 
  PodScheduled                True 
Volumes:
  kube-api-access-rq4gw:
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
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  7s    default-scheduler  Successfully assigned scn-shop/shop-frontend-69df64b78f-5qwhx to airgap-sre-control-plane
  Normal  Pulled     6s    kubelet            spec.containers{web}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal  Created    6s    kubelet            spec.containers{web}: Container created
  Normal  Started    6s    kubelet            spec.containers{web}: Container started
```

### 8. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-shop","resource_name":"shop-frontend","resource_type":"deployment"}
```

### 9. tool result: `k8s_describe_resource`

```text
Name:                   shop-frontend
Namespace:              scn-shop
CreationTimestamp:      Tue, 24 Nov 2026 22:24:43 +0000
Labels:                 app=shop-frontend
Annotations:            deployment.kubernetes.io/revision: 1
Selector:               app=shop-frontend
Replicas:               2 desired | 2 updated | 2 total | 2 available | 0 unavailable
StrategyType:           RollingUpdate
MinReadySeconds:        0
RollingUpdateStrategy:  25% max unavailable, 25% max surge
Pod Template:
  Labels:  app=shop-frontend
  Containers:
   web:
    Image:      docker.io/library/busybox:1.37.0
    Port:       8080/TCP
    Host Port:  0/TCP
    Command:
      sh
      -c
      mkdir -p /www && echo "storefront ok" > /www/index.html
      exec httpd -f -p 8080 -h /www
      
    Limits:
      memory:  32Mi
    Requests:
      cpu:         10m
      memory:      16Mi
    Readiness:     http-get http://:8080/ delay=0s timeout=1s period=2s successThreshold=1 failureThreshold=3
    Environment:   <none>
    Mounts:        <none>
  Volumes:         <none>
  Node-Selectors:  <none>
  Tolerations:     <none>
Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      True    MinimumReplicasAvailable
  Progressing    True    NewReplicaSetAvailable
OldReplicaSets:  <none>
NewReplicaSet:   shop-frontend-69df64b78f (2/2 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  8s    deployment-controller  Scaled up replica set shop-frontend-69df64b78f from 0 to 2
```

### 10. agent

I've found the root cause. The Service `storefront` has a selector of `app=shop-front`, but the application pods have the label `app=shop-frontend`. These labels don't match, so the Service has no endpoints and cannot route traffic to any pod.

**Evidence:**
- Service `storefront` in namespace `scn-shop` has selector: `app=shop-front`
- Pods `shop-frontend-69df64b78f-5qwhx` and `shop-frontend-69df64b78f-tvvb7` have label: `app=shop-frontend`
- Service `storefront` shows `Endpoints:` as empty

The root cause is a **label mismatch** between the Service selector and the pod labels.

```json
{
  "root_cause_category": "service_selector_mismatch",
  "resources": [
    {"kind": "Service", "name": "storefront"},
    {"kind": "Pod", "name": "shop-frontend-69df64b78f-5qwhx"}
  ],
  "summary": "The Service storefront selector app=shop-front does not match the pod label app=shop-frontend, resulting in no endpoints."
}
```
