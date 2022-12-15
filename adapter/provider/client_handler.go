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
    "strconv"
//    "errors"
//    "time"

	"k8s.io/klog/v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (p *colibriProvider) checkPod(namespaceName string, podName string) (*unstructured.Unstructured, error) {

    //check namespace
    res := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
    if _, err := p.client.Resource(res).Get(context.TODO(), namespaceName, metav1.GetOptions{}); err != nil {
        return nil, err
    }
    //check pod
    res = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
    pod, err :=  p.client.Resource(res).Namespace(namespaceName).Get(context.TODO(), podName, metav1.GetOptions{})
    if err != nil {
        return nil, err
    }
    //no need to check container for now
    //check container
    /*
    containers := pod.Object["spec"].(map[string]interface{})["containers"].([]interface{})
    for _, c := range containers {
        if containerName == c.(map[string]interface{})["name"].(string) {
            return nil
        }
    }

    return errors.New("container \"" + containerName + "\" not found")
    */

    return pod, nil
}

func (p *colibriProvider) runColibriJob(pod *unstructured.Unstructured, params *jobParam, namespaceName string, podName string, pid string) {

    //get node
    node := pod.Object["spec"].(map[string]interface{})["nodeName"].(string)

    //create service account
	klog.Infof("Creating Job...")

    //job runned by api server doesn't keep output files (currently), and running all metrics types

    jobResource := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
    job := &unstructured.Unstructured{
                Object: map[string]interface{}{
                    "apiVersion": "batch/v1",
                    "kind":       "Job",
                    "metadata": map[string]interface{}{
                            "name": podName + "-" + pid + "-colibri-job",
                            "namespace": "colibri",
                    },
                    "spec": map[string]interface{}{
                            "template": map[string]interface{}{
                                "spec": map[string]interface{}{
                                    "nodeName": node,
                                    "serviceAccountName": "colibri-job",
                                    "restartPolicy": "Never",
                                    "volumes": []map[string]interface{}{
                                        {
                                            "name": "proc-dir",
                                            "hostPath": map[string]interface{}{
                                                "type": "Directory",
                                                "path": "/proc",
                                            },
                                        },
                                        {
                                            "name": "cgroup-dir",
                                            "hostPath": map[string]interface{}{
                                                "type": "Directory",
                                                "path": "/sys/fs/cgroup",
                                            },
                                        },
                                    },
                                    "containers": []map[string]interface{}{
                                        {
                                            "name": "cjob",
                                            "image": "gabbro:30500/colibri-job:latest",
                                            "imagePullPolicy": "Never",
                                            "command": []string{
                                                "colibri", "--pid", pid,
                                                           "--freq", strconv.Itoa(params.Frequency),
                                                           "--iter", strconv.Itoa(params.Iteration),
                                                           "--pert", strconv.Itoa(params.Percentile),
                                                           "--out", "api:" + namespaceName + "." + podName + "." + pid,
                                                           "--mtype", "all",
                                            },
                                            "volumeMounts": []map[string]interface{}{
                                                {
                                                    "mountPath": "/tmp/proc",
                                                    "name": "proc-dir",
                                                },
                                                {
                                                    "mountPath": "/tmp/cgroup",
                                                    "name": "cgroup-dir",
                                                },
                                            },
                                        },
                                    },
                                },
                            },
                    },
                },
           }

	result, err := p.client.Resource(jobResource).Namespace("colibri").Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("Failed to create job: %s", err)
        return
	}
	klog.Infof("Created job %q", result.GetName())

}











