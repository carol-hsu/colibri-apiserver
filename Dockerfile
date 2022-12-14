# Copyright 2022 Carol Hsu
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.18-alpine as builder
ARG VERSION=0.1
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# build
WORKDIR /go/src/colibri-apiserver/
COPY . .
RUN go install -mod=readonly k8s.io/kube-openapi/cmd/openapi-gen && \
    /go/bin/openapi-gen --logtostderr \
        -i k8s.io/metrics/pkg/apis/custom_metrics,k8s.io/metrics/pkg/apis/custom_metrics/v1beta1,k8s.io/metrics/pkg/apis/custom_metrics/v1beta2,k8s.io/metrics/pkg/apis/external_metrics,k8s.io/metrics/pkg/apis/external_metrics/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1 \
        -h ./hack/boilerplate.go.txt \
        -p ./adapter/generated/openapi \
        -O zz_generated.openapi \
        -o ./ \
        -r /dev/null

RUN GO111MODULE=on go mod download
RUN go build -o /go/bin/colibri-apiserver colibri-apiserver/adapter

# runtime image
FROM gcr.io/google_containers/ubuntu-slim:0.14
COPY --from=builder /go/bin/colibri-apiserver /usr/bin/colibri-apiserver
CMD ["colibri-apiserver"]
