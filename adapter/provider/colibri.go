/*
Copyright 2022 Carol Hsu

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"time"

	"github.com/emicklei/go-restful"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/metrics/pkg/apis/custom_metrics"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider/helpers"
)

type customKey struct {
	provider.CustomMetricInfo
	types.NamespacedName
}

// Implementation of provider.CustomMetricsProvider
type colibriProvider struct {
	client dynamic.Interface
	mapper apimeta.RESTMapper

	values map[customKey]resource.Quantity
}

func NewProvider(client dynamic.Interface, mapper apimeta.RESTMapper) (provider.CustomMetricsProvider, *restful.WebService) {
	p := &colibriProvider{
		client: client,
		mapper: mapper,
		values: make(map[customKey]resource.Quantity),
	}
	return p, p.webService()
}

// get the value from the map of provider (p.values)
func (p *colibriProvider) valueFor(info provider.CustomMetricInfo, name types.NamespacedName) (resource.Quantity, error) {
	info, _, err := info.Normalized(p.mapper)
	if err != nil {
		return resource.Quantity{}, err
	}
	ckey := customKey{
		CustomMetricInfo: info,
		NamespacedName:   name,
	}

	value, found := p.values[ckey]
	if !found {
		return resource.Quantity{}, provider.NewMetricNotFoundError(info.GroupResource, info.Metric)
	}

	return value, nil
}

// come out a standardize metric: info+value
func (p *colibriProvider) metricFor(value resource.Quantity,
	name types.NamespacedName,
	info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
	objRef, err := helpers.ReferenceFor(p.mapper, name, info)
	if err != nil {
		return nil, err
	}

	return &custom_metrics.MetricValue{
		DescribedObject: objRef,
		Metric:          custom_metrics.MetricIdentifier{Name: info.Metric},
		Timestamp:       metav1.Time{time.Now()},
		Value:           value,
	}, nil
}

// list all info of metrics, from the map of provider (p.values)
func (p *colibriProvider) ListAllMetrics() []provider.CustomMetricInfo {

	// Get unique CustomMetricInfos from wrapper CustomMetricResources
	infos := make(map[provider.CustomMetricInfo]struct{})
	for resource := range p.values {
		infos[resource.CustomMetricInfo] = struct{}{}
	}

	// Build slice of CustomMetricInfos to be returns
	metrics := make([]provider.CustomMetricInfo, 0, len(infos))
	for info := range infos {
		metrics = append(metrics, info)
	}

	return metrics
}

// get the standardize metric (info+value) by name
func (p *colibriProvider) GetMetricByName(ctx context.Context,
	name types.NamespacedName,
	info provider.CustomMetricInfo,
	metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	value, err := p.valueFor(info, name)
	if err != nil {
		return nil, err
	}
	return p.metricFor(value, name, info)
}

func (p *colibriProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector,
	info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {

	names, err := helpers.ListObjectNames(p.mapper, p.client, namespace, selector, info)
	if err != nil {
		return nil, err
	}

	res := make([]custom_metrics.MetricValue, len(names))
	for i, name := range names {
		// TODO: not sure what this function used for, need to update later
		namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
		value, err := p.valueFor(info, namespacedName)
		if err != nil {
			if apierr.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		metric, err := p.metricFor(value, namespacedName, info)
		if err != nil {
			return nil, err
		}
		res[i] = *metric
	}

	return &custom_metrics.MetricValueList{Items: res}, nil
}
