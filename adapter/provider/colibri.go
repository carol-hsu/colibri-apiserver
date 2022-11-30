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
    "net/http"

	"k8s.io/klog/v2"
	"github.com/emicklei/go-restful"
    apierr "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

type jobParam struct {
    Frequency    int `json:"freq" description:"frequency of query" default:"10"`
    Iteration   int `json:"iter" description:"iteration of query" default:"1000"`
    Percentile  int `json:"pert" description:"percentile of data analytics" default:"99"`
}


//Implementation of provider.CustomMetricsProvider
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

func (p *colibriProvider) webService() *restful.WebService {
	ws := new(restful.WebService)
    ws.Path("/colibri").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

    ws.Route(ws.POST("/{namespace}/{pod}/{container}").
                To(p.setColibriJob).
                Reads(jobParam{}))

//    ws.Route(ws.POST("/{namespace}/{pod}/{container}").
//                To(p.runColibriJob))

    ws.Route(ws.GET("/{namespace}/{pod}/{container}").
                To(p.showColibriJob).
                Writes(jobParam{}))
	return ws
}

func (p *colibriProvider) infoWrapper(metric string, nsname types.NamespacedName) customKey{

    groupResource := schema.ParseGroupResource("pods")
    info := provider.CustomMetricInfo{
                        GroupResource: groupResource,
                        Metric:        metric,
                        Namespaced:    true,
                    }

    info, _, err := info.Normalized(p.mapper)

    if err != nil {
		klog.Errorf("Error normalizing info: %s", err)
	}

    return customKey {
                CustomMetricInfo: info,
                NamespacedName:   nsname,
	       }

}

//set parameter for a colibri job
func (p *colibriProvider) setColibriJob(request *restful.Request, response *restful.Response){
    ns := request.PathParameter("namespace")
    pname := request.PathParameter("pod")
    cname := request.PathParameter("container")

    klog.Infof("Get: "+ ns +" "+ pname +" "+ cname)
    namespacedName := types.NamespacedName{
                            Name: pname,
                            Namespace: ns,
                      }

    //TODO: check all naming is legel
    //param == freq, iter, pert

    param := new(jobParam)
    if err := request.ReadEntity(&param); err != nil {
        response.WriteError(http.StatusInternalServerError, err)
        return
    }

    freqInfo := p.infoWrapper(cname+"-freq", namespacedName)
    p.values[freqInfo] = *resource.NewQuantity(int64(param.Frequency), resource.DecimalSI)

    iterInfo := p.infoWrapper(cname+"-iter", namespacedName)
    p.values[iterInfo] = *resource.NewQuantity(int64(param.Iteration), resource.DecimalSI)

    pertInfo := p.infoWrapper(cname+"-pert", namespacedName)
    p.values[pertInfo] = *resource.NewQuantity(int64(param.Percentile), resource.DecimalSI)

    response.WriteEntity(param)

}

//get parameters of a job
func (p *colibriProvider) showColibriJob(request *restful.Request, response *restful.Response){
    ns := request.PathParameter("namespace")
    pname := request.PathParameter("pod")
    cname := request.PathParameter("container")

    klog.Infof("Get: "+ ns +" "+ pname +" "+ cname)
    namespacedName := types.NamespacedName{
                            Name: pname,
                            Namespace: ns,
                      }
    //TODO: check all naming is legel
    freqInfo := p.infoWrapper(cname+"-freq", namespacedName)
    freq, found := p.values[freqInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(freqInfo.GroupResource, freqInfo.Metric).Error())
        return
    }

    iterInfo := p.infoWrapper(cname+"-iter", namespacedName)
    iter, found := p.values[iterInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(iterInfo.GroupResource, iterInfo.Metric).Error())
        return
    }

    pertInfo := p.infoWrapper(cname+"-pert", namespacedName)
    pert, found := p.values[pertInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(pertInfo.GroupResource, pertInfo.Metric).Error())
        return
    }


    response.WriteEntity(jobParam{
                            Frequency: int(freq.Value()),
                            Iteration: int(iter.Value()),
                            Percentile: int(pert.Value()),
                        })

}


func (p *colibriProvider) runColibriJob(request *restful.Request, response *restful.Response){
    ns := request.PathParameter("namespace")
    pname := request.PathParameter("pod")
    cname := request.PathParameter("container")
    //TODO: check ns, pname, and cname are real and existed in K8s

    klog.Infof("Run job on: "+ ns +" "+ pname +" "+ cname)
    namespacedName := types.NamespacedName{
                            Name: pname,
                            Namespace: ns,
                      }
    //check if parameters are ready
    freqInfo := p.infoWrapper(cname+"-freq", namespacedName)
    iterInfo := p.infoWrapper(cname+"-iter", namespacedName)
    pertInfo := p.infoWrapper(cname+"-pert", namespacedName)

    freq, found := p.values[freqInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(freqInfo.GroupResource, freqInfo.Metric).Error())
        return
    }

    iter, found := p.values[iterInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(iterInfo.GroupResource, iterInfo.Metric).Error())
        return
    }

    pert, found := p.values[pertInfo]
    if !found {
        response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(pertInfo.GroupResource, pertInfo.Metric).Error())
        return
    }

    klog.Infof(freq.String())
    klog.Infof(iter.String())
    klog.Infof(pert.String())
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

//list all info of metrics, from the map of provider (p.values)
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

//get the standardize metric (info+value) by name
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
