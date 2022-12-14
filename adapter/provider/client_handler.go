package provider

import (
    "context"
//    "errors"
//    "time"

//	"k8s.io/klog/v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

//var ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)

func (p *colibriProvider) checkPod(namespace_name string, pod_name string) error {

    //check namespace
    res := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
    if _, err := p.client.Resource(res).Get(context.TODO(), namespace_name, metav1.GetOptions{}); err != nil {
        return err
    }
    //check pod
    res = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
    if _, err := p.client.Resource(res).Namespace(namespace_name).Get(context.TODO(), pod_name, metav1.GetOptions{});
          err != nil {
        return err
    }
    //no need to check container for now
    //check container
    /*
    containers := pod.Object["spec"].(map[string]interface{})["containers"].([]interface{})
    for _, c := range containers {
        if container_name == c.(map[string]interface{})["name"].(string) {
            return nil
        }
    }

    return errors.New("container \"" + container_name + "\" not found")
    */

    return nil
}
