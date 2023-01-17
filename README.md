# Colibri API Server

Following the [custom metrics apiserver](https://github.com/kubernetes-sigs/custom-metrics-apiserver) of Kubernetes,
Colibri API Server publishes run-as-demand metrics APIs in K8s's control plane for resource management.

Build and run Colibri API server on your K8s cluster with `Dockerfile` and `colibri-apiserver.yml`.

And, you can access API server by sending HTTP requests. Please referring following steps and directions.

1. Starting proxy entry of Kubernetes API server on master node.

First, you need to open K8s API accessibility by enable proxy.

```
// open port to listening on 8080
$ kubectl proxy -p 8080 &
Starting to serve on 127.0.0.1:8080
```

2. Sending request to API

After proxy is enable and you deploy Colibri API server, you can start to send API requests.

```
// make sure Colibri API server is running 
$ kubectl get pod -n colibri
NAME                                 READY   STATUS    RESTARTS   AGE
colibri-apiserver-55fbbb5594-7pmrm   1/1     Running   0          13m
```

The endpoint URL would be `http://localhost:8080/api/v1/namespaces/colibri/services/colibri-apiserver:http/proxy/colibri`.

```
// send a API requst

$ curl --request POST -H 'Content-Type: application/json' http://localhost:8080/api/v1/namespaces/colibri/services/colibri-apiserver:http/proxy/colibri/default/obj-detect-tf-serving-6c56b6c79c-zqw46/26386 --data-raw '{"freq": 10, "iter": 20000, "pert": 99}'
Running colibri: default obj-detect-tf-serving-6c56b6c79c-zqw46 26386

```

More available API pathes and their payloads are listed as below:

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /{namespace}/{pod}/{processId} | [run a job](#run-job) | Running a job with requested configurations |
| GET | /{namespace}/{pod}/{processId}/param | [check query parameters](#check-job) | Review a parameter set of a job |
| POST | /{requestId} | [save a result](#store-job) | Store/send back the result (of a job) |
| GET | /{namespace}/{pod}/{processId} | [check a result](#read-job) | Read a result |

## Paths

### <span id="run-job"></span> Running a job with requested configurations

```
POST /{namespace}/{pod}/{processId}
```

#### Consumes
  * application/json

#### Produces
  * text/plain

#### Parameters

| Name | Source | Type | Required | Default | Description |
|------|--------|------| :------: |---------|-------------|
| namespace | `path` | string | ✓ | | The K8s Namespace of the targeted application |
| pod | `path` | string | ✓ | | The K8s Pod of the targeted application |
| processId | `path` | string | ✓ | | The process ID of the targeted application |
| freq | `body` | int | ✓ | | The query interval in millisecond |
| iter | `body` | int | ✓ | | The query iterations |
| pert | `body` | int | ✓ | | The percentile number for data analytic |


#### All responses
| Code | Status | Description |
|------|--------|-------------|
| 200 |  |  | 
| 400 | Bad request | Pod is not existed / the format of parameter set is not correct |


### <span id="check-job"></span> Review a parameter set of a job

```
GET /{namespace}/{pod}/{processId}/param
```

#### Produces
  * application/json


#### Parameters

| Name | Source | Type  | Required | Default | Description |
|------|--------|------| :------: |---------|-------------|
| namespace | `path` | string | ✓ | | The K8s Namespace of the targeted application |
| pod | `path` | string | ✓ | | The K8s Pod of the targeted application |
| processId | `path` | string | ✓ | | The process ID of the targeted application |

#### All responses
| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Return a parameter set | 
| 400 | Bad request | There is no such targeted resource|



### <span id="store-job"></span> Store/send back the result (of a job)

```
POST /{requestId}
```

#### Consumes
  * application/json

#### Produces
  * text/plain

#### Parameters

| Name | Source | Type  | Required | Default | Description |
|------|--------|------| :------: |---------|-------------|
| requestId | `path` | string | ✓ | | The uuid for a specific job |
| cpu | `body` | string | ✓ | | CPU utilization |
| ram | `body` | string | ✓ | | Memory utilization |
| ingress | `body` | string | ✓ | | Ingress traffic bandwidth utilization |
| egress | `body` | string | ✓ | | Egress traffic bandwidth utilization |

#### All responses
| Code | Status | Description |
|------|--------|-------------|
| 200 | OK  |  | 
| 400 | Bad request | Pod is not existed / the format of parameter set is not correct |
| 500 | Internal server error  | Cannot store the metrics | 


### <span id="read-job"></span> Read a result

```
GET /{namespace}/{pod}/{processId}
```

#### Produces
  * application/json

#### Parameters

| Name | Source | Type | Required | Default | Description |
|------|--------|------| :------: |---------|-------------|
| namespace | `path` | string | ✓ | | The K8s Namespace of the targeted application |
| pod | `path` | string | ✓ | | The K8s Pod of the targeted application |
| processId | `path` | string | ✓ | | The process ID of the targeted application |

#### All responses
| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Return a result including four metrics | 
| 400 | Bad request | Pod/result is not existed |

