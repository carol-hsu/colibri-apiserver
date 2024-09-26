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
	"net/http"
	"strings"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

type jobParam struct {
	Frequency  int `json:"freq" description:"frequency of query" default:"10"`
	Iteration  int `json:"iter" description:"iteration of query" default:"1000"`
	Percentile int `json:"pert" description:"percentile of data analytics" default:"99"`
}

// The returned results could directly used on K8s deployment: with unit tag if required
type jobResult struct {
	Cpu     string `json:"cpu" description:"CPU utilization" default:"0m"`
	Ram     string `json:"ram" description:"Memory utilization" default:"0Mi"`
	Ingress string `json:"ingress" description:"Ingress traffic bandwidth" default:"0k"`
	Egress  string `json:"egress" description:"Egress traffic bandwidth" default:"0k"`
}

func (p *colibriProvider) webService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/colibri").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	//run Colibri with specified parameters
	ws.Route(ws.POST("/{namespace}/{pod}/{process}").
		To(p.runJob).
		Reads(jobParam{}))

	//put result (from colibri job)
	ws.Route(ws.POST("/{resultId}").
		To(p.putResult).
		Reads(jobResult{}))

	//get parameters
	ws.Route(ws.GET("/{namespace}/{pod}/{process}/param").
		To(p.getParameter).
		Writes(jobParam{}))

	//get result
	ws.Route(ws.GET("/{namespace}/{pod}/{process}").
		To(p.getResult).
		Writes(jobResult{}))

	return ws
}

func (p *colibriProvider) infoWrapper(metric string, nsname types.NamespacedName) customKey {

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

	return customKey{
		CustomMetricInfo: info,
		NamespacedName:   nsname,
	}

}

// set parameter for a colibri job
// and run colibri
func (p *colibriProvider) runJob(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")
	pname := request.PathParameter("pod")
	pid := request.PathParameter("process")

	klog.Infof("Run Colibri for: " + ns + "." + pname + "." + pid)
	namespacedName := types.NamespacedName{
		Name:      pname,
		Namespace: ns,
	}

	// check all naming on the path is existing/running compute unit
	pod, err := p.checkPod(ns, pname)
	if err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	//TODO: check what if parameters are not valid numbers
	params := new(jobParam)
	if err := request.ReadEntity(&params); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	freqInfo := p.infoWrapper(pid+"-freq", namespacedName)
	p.values[freqInfo] = *resource.NewQuantity(int64(params.Frequency), resource.DecimalSI)

	iterInfo := p.infoWrapper(pid+"-iter", namespacedName)
	p.values[iterInfo] = *resource.NewQuantity(int64(params.Iteration), resource.DecimalSI)

	pertInfo := p.infoWrapper(pid+"-pert", namespacedName)
	p.values[pertInfo] = *resource.NewQuantity(int64(params.Percentile), resource.DecimalSI)

	p.runColibriJob(pod, params, ns, pname, pid)
	klog.Infof("Started Colibri job: " + ns + "." + pname + "." + pid)
	response.Write([]byte("Running colibri: " + ns + " " + pname + " " + pid + "\n"))

}

// get parameters of a job
func (p *colibriProvider) getParameter(request *restful.Request, response *restful.Response) {
	ns := request.PathParameter("namespace")
	pname := request.PathParameter("pod")
	pid := request.PathParameter("process")

	klog.Infof("Get parameters of: " + ns + " " + pname + " " + pid)
	namespacedName := types.NamespacedName{
		Name:      pname,
		Namespace: ns,
	}
	//TODO: check all naming is legel
	freqInfo := p.infoWrapper(pid+"-freq", namespacedName)
	freq, found := p.values[freqInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(freqInfo.GroupResource, freqInfo.Metric).Error())
		return
	}

	iterInfo := p.infoWrapper(pid+"-iter", namespacedName)
	iter, found := p.values[iterInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(iterInfo.GroupResource, iterInfo.Metric).Error())
		return
	}

	pertInfo := p.infoWrapper(pid+"-pert", namespacedName)
	pert, found := p.values[pertInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(pertInfo.GroupResource, pertInfo.Metric).Error())
		return
	}

	response.WriteEntity(jobParam{
		Frequency:  int(freq.Value()),
		Iteration:  int(iter.Value()),
		Percentile: int(pert.Value()),
	})
}

func (p *colibriProvider) putMetric(value string, key string, nsname types.NamespacedName) error {
	q, err := resource.ParseQuantity(value)
	if err != nil {
		return err
	}
	info := p.infoWrapper(key, nsname)
	p.values[info] = q

	return nil
}

func (p *colibriProvider) putResult(request *restful.Request, response *restful.Response) {

	klog.Infof("Get request for putting result")

	names := strings.Split(request.PathParameter("resultId"), ".")
	if len(names) < 3 {
		response.WriteErrorString(http.StatusBadRequest, "Result ID is not existed\n")
		return
	}
	ns, pname, pid := names[0], names[1], names[2]

	// check all naming on the path is existing/running compute unit
	if _, err := p.checkPod(ns, pname); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	//TODO: check what if parameters are not valid numbers
	metrics := new(jobResult)
	if err := request.ReadEntity(&metrics); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	namespacedName := types.NamespacedName{
		Name:      pname,
		Namespace: ns,
	}

	if err := p.putMetric(metrics.Cpu, pid+"-cpu", namespacedName); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	if err := p.putMetric(metrics.Ram, pid+"-ram", namespacedName); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	if err := p.putMetric(metrics.Ingress, pid+"-ig", namespacedName); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	if err := p.putMetric(metrics.Egress, pid+"-eg", namespacedName); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	klog.Infof("Put result for: " + ns + "." + pname + "." + pid)
	response.Write([]byte("Put Colibri result: " + ns + "." + pname + "." + pid + "\n"))

}

func (p *colibriProvider) getResult(request *restful.Request, response *restful.Response) {

	ns := request.PathParameter("namespace")
	pname := request.PathParameter("pod")
	pid := request.PathParameter("process")

	klog.Infof("Get results of: " + ns + " " + pname + " " + pid)
	namespacedName := types.NamespacedName{
		Name:      pname,
		Namespace: ns,
	}

	cpuInfo := p.infoWrapper(pid+"-cpu", namespacedName)
	cpu, found := p.values[cpuInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(cpuInfo.GroupResource, cpuInfo.Metric).Error())
		return
	}

	ramInfo := p.infoWrapper(pid+"-ram", namespacedName)
	ram, found := p.values[ramInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(ramInfo.GroupResource, ramInfo.Metric).Error())
		return
	}

	igInfo := p.infoWrapper(pid+"-ig", namespacedName)
	ig, found := p.values[igInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(igInfo.GroupResource, igInfo.Metric).Error())
		return
	}

	egInfo := p.infoWrapper(pid+"-eg", namespacedName)
	eg, found := p.values[egInfo]
	if !found {
		response.WriteErrorString(http.StatusBadRequest, provider.NewMetricNotFoundError(egInfo.GroupResource, egInfo.Metric).Error())
		return
	}

	response.WriteEntity(jobResult{
		Cpu:     cpu.String(),
		Ram:     ram.String(),
		Ingress: ig.String(),
		Egress:  eg.String(),
	})
}
