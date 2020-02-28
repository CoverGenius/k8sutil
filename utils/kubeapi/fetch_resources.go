package kubeapi

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/CoverGenius/k8sutil/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// just trial it on this subset first!
var specialMethods = []string{
	"ClusterRoles",
	"ClusterRoleBindings",
	"RoleBindings",
	"PriorityClasses",
	"VolumeAttachments",
	"Roles",
	"DaemonSets",
	"Deployments",
	"Ingresses",
	"ReplicaSets",
	"NetworkPolicies",
	"Events",
	"Cronjobs",
	"Jobs",
	"PersistentVolumeClaims",
	"StatefulSets",
	"ServiceAccounts",
	"ConfigMaps",
}

func Convert(resources []interface{}) []*utils.ResourceInfo {
	// These are the basic Deployment objects
	// They will usually have the needed fields. We will do some reflection to
	// get what we need out. Because I am worried about some of the shit not being a runtime object
	// for some reason.
	var result []*utils.ResourceInfo
	for _, resource := range resources {
		value := reflect.ValueOf(resource)
		if value.Kind() != reflect.Struct {
			continue // have no idea what it is then lmfao
		}
		r := &utils.ResourceInfo{}
		nameValue := value.FieldByName("Name")
		if nameValue.IsValid() && nameValue.Kind() == reflect.String {
			// then let's grab it out and assign it to r
			r.Name = nameValue.Interface().(string)
		}
		namespaceValue := value.FieldByName("Namespace")
		if namespaceValue.IsValid() && namespaceValue.Kind() == reflect.String {
			r.Namespace = namespaceValue.Interface().(string)
		}
		kindValue := value.FieldByName("Kind")
		if kindValue.IsValid() && kindValue.Kind() == reflect.String && kindValue.Interface().(string) != "" {
			r.Kind = kindValue.Interface().(string)
		} else {
			// put in the typename of this object
			r.Kind = value.Type().Name()
		}
		// try to get out the labels
		labels := value.FieldByName("Labels")
		if !labels.IsValid() || labels.Type().Kind() != reflect.Map ||
			labels.Type().Key().Kind() != reflect.String ||
			labels.Type().Elem().Kind() != reflect.String {
			continue
			// it's just not the type I was expecting so I'm gunna ignore it.
		}
		r.Labels = labels.Interface().(map[string]string)
		result = append(result, r)
	}
	return result
}

func GetResources(clientset *kubernetes.Clientset, namespace string) ([]interface{}, error) {
	var resources []interface{}
	// iterate through the methods of Clientset
	clientsetType := reflect.TypeOf(clientset)
	clientsetValue := reflect.ValueOf(clientset)
	// we need this namespace thing later
	args := []reflect.Value{reflect.ValueOf(namespace)}
	options := []reflect.Value{reflect.ValueOf(metav1.ListOptions{})}
	for i := 0; i < clientsetType.NumMethod(); i++ {
		APIGroupRetriever := clientsetValue.Method(i)
		if !RegularAPIGroupInterfaceRetriever(APIGroupRetriever.Type()) {
			continue
		}
		// now we know this takes 0 params and gives 1 value back.
		APIGroup := APIGroupRetriever.Call(nil)[0]
		// loop through API Group's methods
		for j := 0; j < APIGroup.Type().NumMethod(); j++ {
			resourceFetcher := APIGroup.Method(j)
			if !CanHelpGetResources(resourceFetcher, APIGroup.Type().Method(j).Name) {
				continue
			}
			// now we now resourceFetcher has 1 string param and returns 1 thing.
			// It usually returns a DeploymentInterface for example.
			ResourceInterface := resourceFetcher.Call(args)[0]
			// Now that we have that, there should be a list method on it which will return
			// for example a DeploymentList.
			listMethod := ResourceInterface.MethodByName("List")
			// check if that was a fail
			if !listMethod.IsValid() {
				return nil, errors.New("There has been a change in the ClientSet API")
			}
			listAndErr := listMethod.Call(options)
			err := listAndErr[1].Interface()
			if err != nil {
				// try to put to error type
				continue
			}
			list := listAndErr[0].Elem() // we know this will work because out(0).Kind == Ptr
			// check underlying is a struct actually
			if list.Kind() != reflect.Struct {
				return nil, errors.New("There has been a change in the ClientSet API")
			}
			items := list.FieldByName("Items")
			if !items.IsValid() || items.Kind() != reflect.Slice {
				return nil, errors.New("There has been a change in the ClientSet API")
			}
			for k := 0; k < items.Len(); k++ {
				if !items.Index(k).CanInterface() {
					continue
				}
				resources = append(resources, items.Index(k).Interface())
			}
		}

	}
	if len(resources) == 0 {
		return resources, errors.New(fmt.Sprintf("No resources found under the namespace %s", namespace))
	}
	return resources, nil
}

//List(opts metav1.ListOptions) (*v1.ControllerRevisionList, error)
// Example of appropriate signature
func RegularListMethod(method reflect.Type) bool {
	return method.NumIn() == 1 && method.NumOut() == 2 && method.Out(0).Kind() == reflect.Ptr
}

// func (c *Clientset) AppsV1() appsv1.AppsV1Interface
// This is an example of a regular API Group Interface retriever.
func RegularAPIGroupInterfaceRetriever(method reflect.Type) bool {
	return method.NumIn() == 0 && method.NumOut() == 1
}

func CanHelpGetResources(method reflect.Value, methodName string) bool {
	if method.Type().NumIn() != 1 ||
		method.Type().NumOut() != 1 ||
		method.Type().In(0).Kind() != reflect.String {
		return false
	}
	for _, name := range specialMethods {
		if methodName == name {
			return true
		}
	}
	return false
}
