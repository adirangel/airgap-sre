# 5. service-selector-mismatch — Service with a selector mismatch (no endpoints)

- Run: 2026-11-25 00:16 IST, model `qwen3.5:4b` (Ollama), agent `sre-agent`
- Result: **PASS**, 1.00 points
- Category: expected `service_selector_mismatch`, agent said `service_selector_mismatch`
- Must name Service `storefront`: named
- Must name Deployment `shop-frontend`: named
- Agent time: 20s, tool calls: 8, tokens: 34492

## Symptom given to the agent

> Requests to storefront in namespace scn-shop fail with connection refused, although the application pods look healthy. Find the root cause.

## Final answer

## Root Cause Analysis

**What's broken:** The Service `storefront` cannot route traffic to the application pods because there are no endpoints configured for it.

**Evidence:**
- The Service `storefront` has selector `app=shop-front` (note: "shop-front")
- The actual pods have label `app=shop-frontend` (note: "shop-frontend")
- These labels do not match, so the Service has no endpoints

**Root cause category:** `service_selector_mismatch`

```json
{
  "root_cause_category": "service_selector_mismatch",
  "resources": [
    {"kind": "Service", "name": "storefront"},
    {"kind": "Pod", "name": "shop-frontend-69df64b78f-qpjpf"},
    {"kind": "Label", "value": "app=shop-front"}
  ],
  "summary": "The Service storefront selector app=shop-front does not match the pod labels app=shop-frontend, resulting in no endpoints."
}
```

## Full transcript

### 1. user

Requests to storefront in namespace scn-shop fail with connection refused, although the application pods look healthy. Find the root cause.

### 2. tool call: `k8s_get_resources`

```json
{"namespace":"scn-shop","output":"wide"}
```

### 3. tool result: `k8s_get_resources`

```text
Tool execution failed. Details: resource_type parameter is required
```

### 4. tool call: `k8s_get_resources`

```json
{"namespace":"scn-shop","output":"wide","resource_type":"pod"}
```

### 5. tool result: `k8s_get_resources`

```text
NAME                             READY   STATUS    RESTARTS   AGE   IP            NODE                       NOMINATED NODE   READINESS GATES
shop-frontend-69df64b78f-qpjpf   1/1     Running   0          5s    10.244.0.30   airgap-sre-control-plane   <none>           <none>
shop-frontend-69df64b78f-xbczt   1/1     Running   0          5s    10.244.0.29   airgap-sre-control-plane   <none>           <none>
```

### 6. tool call: `k8s_get_resources`

```json
{"namespace":"scn-shop","output":"wide","resource_type":"service"}
```

### 7. tool result: `k8s_get_resources`

```text
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE   SELECTOR
storefront   ClusterIP   10.96.169.146   <none>        80/TCP    7s    app=shop-front
```

### 8. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-shop","resource_name":"storefront","resource_type":"service"}
```

### 9. tool result: `k8s_describe_resource`

```text
Name:                     storefront
Namespace:                scn-shop
Labels:                   <none>
Annotations:              <none>
Selector:                 app=shop-front
Type:                     ClusterIP
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.96.169.146
IPs:                      10.96.169.146
Port:                     <unset>  80/TCP
TargetPort:               8080/TCP
Endpoints:                <none>
Session Affinity:         None
Internal Traffic Policy:  Cluster
Events:                   <none>
```

### 10. tool call: `k8s_describe_resource`

```json
{"namespace":"scn-shop","resource_name":"shop-frontend-69df64b78f-qpjpf","resource_type":"pod"}
```

### 11. tool result: `k8s_describe_resource`

```text
Name:             shop-frontend-69df64b78f-qpjpf
Namespace:        scn-shop
Priority:         0
Service Account:  default
Node:             airgap-sre-control-plane/172.18.0.2
Start Time:       Tue, 24 Nov 2026 22:20:25 +0000
Labels:           app=shop-frontend
                  pod-template-hash=69df64b78f
Annotations:      <none>
Status:           Running
IP:               10.244.0.30
IPs:
  IP:           10.244.0.30
Controlled By:  ReplicaSet/shop-frontend-69df64b78f
Containers:
  web:
    Container ID:  containerd://aa76fc81314cefb4d6c8384990b509a4f2c565bee64673e1e18f15eebb6bff17
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
      Started:      Tue, 24 Nov 2026 22:20:25 +0000
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
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-cx85t (ro)
Conditions:
  Type                        Status
  PodReadyToStartContainers   True 
  Initialized                 True 
  Ready                       True 
  ContainersReady             True 
  PodScheduled                True 
Volumes:
  kube-api-access-cx85t:
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
  Normal  Scheduled  10s   default-scheduler  Successfully assigned scn-shop/shop-frontend-69df64b78f-qpjpf to airgap-sre-control-plane
  Normal  Pulled     10s   kubelet            spec.containers{web}: Container image "docker.io/library/busybox:1.37.0" already present on machine and can be accessed by the pod
  Normal  Created    10s   kubelet            spec.containers{web}: Container created
  Normal  Started    10s   kubelet            spec.containers{web}: Container started
```

### 12. tool call: `k8s_get_resource_yaml`

```json
{"namespace":"scn-shop","resource_name":"storefront","resource_type":"service"}
```

### 13. tool result: `k8s_get_resource_yaml`

```text
apiVersion: v1
kind: Service
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Service","metadata":{"annotations":{},"name":"storefront","namespace":"scn-shop"},"spec":{"ports":[{"port":80,"targetPort":8080}],"selector":{"app":"shop-front"}}}
  creationTimestamp: "2026-11-24T22:20:25Z"
  name: storefront
  namespace: scn-shop
  resourceVersion: "22753"
  uid: 83cf7d41-78a0-49a9-a19d-cbaea50c618a
spec:
  clusterIP: 10.96.169.146
  clusterIPs:
  - 10.96.169.146
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: shop-front
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

### 14. tool call: `k8s_get_resource_yaml`

```json
{"namespace":"scn-shop","resource_name":"shop-frontend","resource_type":"deployment"}
```

### 15. tool result: `k8s_get_resource_yaml`

```text
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"labels":{"app":"shop-frontend"},"name":"shop-frontend","namespace":"scn-shop"},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"shop-frontend"}},"template":{"metadata":{"labels":{"app":"shop-frontend"}},"spec":{"containers":[{"command":["sh","-c","mkdir -p /www \u0026\u0026 echo \"storefront ok\" \u003e /www/index.html\nexec httpd -f -p 8080 -h /www\n"],"image":"docker.io/library/busybox:1.37.0","name":"web","ports":[{"containerPort":8080}],"readinessProbe":{"httpGet":{"path":"/","port":8080},"periodSeconds":2},"resources":{"limits":{"memory":"32Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}}],"terminationGracePeriodSeconds":1}}}}
  creationTimestamp: "2026-11-24T22:20:25Z"
  generation: 1
  labels:
    app: shop-frontend
  name: shop-frontend
  namespace: scn-shop
  resourceVersion: "22784"
  uid: 21f62cd2-f6ff-4028-95cc-3ff4439ffc61
spec:
  progressDeadlineSeconds: 600
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: shop-frontend
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: shop-frontend
    spec:
      containers:
      - command:
        - sh
        - -c
        - |
          mkdir -p /www && echo "storefront ok" > /www/index.html
          exec httpd -f -p 8080 -h /www
        image: docker.io/library/busybox:1.37.0
        imagePullPolicy: IfNotPresent
        name: web
        ports:
        - containerPort: 8080
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /
            port: 8080
            scheme: HTTP
          periodSeconds: 2
          successThreshold: 1
          timeoutSeconds: 1
        resources:
          limits:
            memory: 32Mi
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
  availableReplicas: 2
  conditions:
  - lastTransitionTime: "2026-11-24T22:20:25Z"
    lastUpdateTime: "2026-11-24T22:20:25Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2026-11-24T22:20:25Z"
    lastUpdateTime: "2026-11-24T22:20:25Z"
    message: ReplicaSet "shop-frontend-69df64b78f" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 1
  readyReplicas: 2
  replicas: 2
  terminatingReplicas: 0
  updatedReplicas: 2
```

### 16. agent

I've found the root cause. Let me verify with a few more checks:

### 17. tool call: `k8s_get_events`

```json
{"namespace":"scn-shop"}
```

### 18. tool result: `k8s_get_events`

```text
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "count": 1,
            "eventTime": null,
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-qpjpf",
                "namespace": "scn-shop",
                "resourceVersion": "22757",
                "uid": "53f04d93-1e61-4b6c-9317-8554082ef9b6"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Successfully assigned scn-shop/shop-frontend-69df64b78f-qpjpf to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-qpjpf.18eb1b3d009ec06b",
                "namespace": "scn-shop",
                "resourceVersion": "22764",
                "uid": "92e6edce-f543-4911-ad22-49cb6c5a6b90"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-qpjpf",
                "namespace": "scn-shop",
                "resourceVersion": "22760",
                "uid": "53f04d93-1e61-4b6c-9317-8554082ef9b6"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-qpjpf.18eb1b3d1c0fb4f9",
                "namespace": "scn-shop",
                "resourceVersion": "22771",
                "uid": "fa9d9f32-e6e3-4be7-81e6-308f331e340b"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-qpjpf",
                "namespace": "scn-shop",
                "resourceVersion": "22760",
                "uid": "53f04d93-1e61-4b6c-9317-8554082ef9b6"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-qpjpf.18eb1b3d1fbd8882",
                "namespace": "scn-shop",
                "resourceVersion": "22774",
                "uid": "90635a27-be6e-4afe-8fca-4590b6db5025"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-qpjpf",
                "namespace": "scn-shop",
                "resourceVersion": "22760",
                "uid": "53f04d93-1e61-4b6c-9317-8554082ef9b6"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-qpjpf.18eb1b3d23560231",
                "namespace": "scn-shop",
                "resourceVersion": "22776",
                "uid": "07be609e-fc39-4935-a5b8-bf141cf2490e"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-xbczt",
                "namespace": "scn-shop",
                "resourceVersion": "22752",
                "uid": "eb2e35a7-15a6-451a-b189-b0bd03ef8c7f"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Successfully assigned scn-shop/shop-frontend-69df64b78f-xbczt to airgap-sre-control-plane",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-xbczt.18eb1b3cffbb323e",
                "namespace": "scn-shop",
                "resourceVersion": "22761",
                "uid": "5c531e6e-71a7-41a1-8c04-0e0fad6eb8f1"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-xbczt",
                "namespace": "scn-shop",
                "resourceVersion": "22754",
                "uid": "eb2e35a7-15a6-451a-b189-b0bd03ef8c7f"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container image \"docker.io/library/busybox:1.37.0\" already present on machine and can be accessed by the pod",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-xbczt.18eb1b3d1b217f48",
                "namespace": "scn-shop",
                "resourceVersion": "22769",
                "uid": "e4973a11-157f-4a21-8b22-0d10e98ac9ac"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-xbczt",
                "namespace": "scn-shop",
                "resourceVersion": "22754",
                "uid": "eb2e35a7-15a6-451a-b189-b0bd03ef8c7f"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container created",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-xbczt.18eb1b3d1ef42afd",
                "namespace": "scn-shop",
                "resourceVersion": "22773",
                "uid": "3d751a7f-df3e-47d2-9859-edfc849d984d"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "v1",
                "fieldPath": "spec.containers{web}",
                "kind": "Pod",
                "name": "shop-frontend-69df64b78f-xbczt",
                "namespace": "scn-shop",
                "resourceVersion": "22754",
                "uid": "eb2e35a7-15a6-451a-b189-b0bd03ef8c7f"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Container started",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f-xbczt.18eb1b3d22ec9e71",
                "namespace": "scn-shop",
                "resourceVersion": "22775",
                "uid": "65b9ccc1-3109-4817-91f2-b0bdbf3c25e1"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "shop-frontend-69df64b78f",
                "namespace": "scn-shop",
                "resourceVersion": "22748",
                "uid": "60422533-6b85-4f27-9de3-4b1d77e83f2a"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Created pod: shop-frontend-69df64b78f-xbczt",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f.18eb1b3cff3ae303",
                "namespace": "scn-shop",
                "resourceVersion": "22755",
                "uid": "ea93c28f-3d9b-4552-8f2d-37d4f3381b06"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "ReplicaSet",
                "name": "shop-frontend-69df64b78f",
                "namespace": "scn-shop",
                "resourceVersion": "22748",
                "uid": "60422533-6b85-4f27-9de3-4b1d77e83f2a"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Created pod: shop-frontend-69df64b78f-qpjpf",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend-69df64b78f.18eb1b3cffb920ef",
                "namespace": "scn-shop",
                "resourceVersion": "22762",
                "uid": "71ce01f9-e1ab-4488-aa8d-5d55195cf495"
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
            "firstTimestamp": "2026-11-24T22:20:25Z",
            "involvedObject": {
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "name": "shop-frontend",
                "namespace": "scn-shop",
                "resourceVersion": "22747",
                "uid": "21f62cd2-f6ff-4028-95cc-3ff4439ffc61"
            },
            "kind": "Event",
            "lastTimestamp": "2026-11-24T22:20:25Z",
            "message": "Scaled up replica set shop-frontend-69df64b78f from 0 to 2",
            "metadata": {
                "creationTimestamp": "2026-11-24T22:20:25Z",
                "name": "shop-frontend.18eb1b3cfe7da18a",
                "namespace": "scn-shop",
                "resourceVersion": "22749",
                "uid": "ee0d4e93-c764-433c-a935-de6b9f0df36e"
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

### 19. agent

## Root Cause Analysis

**What's broken:** The Service `storefront` cannot route traffic to the application pods because there are no endpoints configured for it.

**Evidence:**
- The Service `storefront` has selector `app=shop-front` (note: "shop-front")
- The actual pods have label `app=shop-frontend` (note: "shop-frontend")
- These labels do not match, so the Service has no endpoints

**Root cause category:** `service_selector_mismatch`

```json
{
  "root_cause_category": "service_selector_mismatch",
  "resources": [
    {"kind": "Service", "name": "storefront"},
    {"kind": "Pod", "name": "shop-frontend-69df64b78f-qpjpf"},
    {"kind": "Label", "value": "app=shop-front"}
  ],
  "summary": "The Service storefront selector app=shop-front does not match the pod labels app=shop-frontend, resulting in no endpoints."
}
```
