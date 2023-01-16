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

// send a API requst

$ curl --request GET -H 'Content-Type: application/json' http://localhost:8080/api/v1/namespaces/colibri/services/colibri-apiserver:http/proxy/colibri/default/obj-detect-tf-serving-6c56b6c79c-zqw46/26386 --data-raw '{"freq": 10, "iter": 20000, "pert": 99}'
Running colibri: default obj-detect-tf-serving-6c56b6c79c-zqw46 26386

// get query results

$ curl --request GET -H 'Content-Type: application/json' http://localhost:8080/api/v1/namespaces/colibri/services/colibri-apiserver:http/proxy/colibri/default/obj-detect-tf-serving-6c56b6c79c-zqw46/26386
{
 "cpu": "11051m",
 "ram": "1201Mi",
 "ingress": "103M",
 "egress": "378k"
}
```

More available API pathes and their payloads are listed as below:

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /{namespace}/{pod}/{processId} | [add pet](#add-pet) | Add a new pet to the store |
| GET | /{namespace}/{pod}/{processId}/param | [update-pet-with-form](#update-pet-with-form) | Finds Pets by status |
| POST | /{requestId} | | |
| GET | /{namespace}/{pod}/{processId} | | |

## Paths

### <span id="add-pet"></span> Add a new pet to the store (*addPet*)

```
POST /v2/pet
```

#### Consumes
  * application/json
  * application/xml

#### Produces
  * application/json
  * application/xml

#### Security Requirements
  * petstore_auth: read:pets, write:pets

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| body | `body` | [Pet](#pet) | `models.Pet` | | ✓ | | Pet object that needs to be added to the store |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [405](#add-pet-405) | Method Not Allowed | Invalid input |  | [schema](#add-pet-405-schema) |

#### Responses


##### <span id="add-pet-405"></span> 405 - Invalid input


### <span id="update-pet-with-form"></span> Updates a pet in the store with form data (*updatePetWithForm*)

```
POST /v2/pet/{petId}
```

#### Consumes
  * application/x-www-form-urlencoded

#### Produces
  * application/json
  * application/xml

#### Security Requirements
  * petstore_auth: read:pets, write:pets

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| petId | `path` | int64 (formatted integer) | `int64` |  | ✓ |  | ID of pet that needs to be updated |
| name | `formData` | string | `string` |  |  |  | Updated name of the pet |
| status | `formData` | string | `string` |  |  |  | Updated status of the pet |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [405](#update-pet-with-form-405) | Method Not Allowed | Invalid input |  | [schema](#update-pet-with-form-405-schema) |

#### Responses


##### <span id="update-pet-with-form-405"></span> 405 - Invalid input
Status: Method Not Allowed

###### <span id="update-pet-with-form-405-schema"></span> Schema

