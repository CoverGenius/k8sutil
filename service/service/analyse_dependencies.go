package service

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacV1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DependencyInformation struct {
	Object       metav1.Object
	Type         meta.Type
	Requirements []string
}

/**
* DEPENDENCIES TO LOOK FOR:
*	Common Dependencies:
*	* For every resource with a namespace field, this namespace must exist.
*
*	RoleBinding:
*	* roleRef.name refers to an existing Role, so a Role by this name must exist.
*	* subects[0].kind and name should refer to a resource in the namespace of the RoleBinding resource
*
*   NetworkPolicy:	(LEAVE FOR NOW)
*	* networkPolicy.spec.ingress[0].from[0].matchLabels is a map and it contains
*	  key/value pairs and these key/value pairs should be present on some other resource.
*
*	Deployment:
*	* A deployment's pod spec may have a key 'volumes', within which there could be a reference to a
*	  PersistentVolumeClaim if a volume's VolumeSource.PersistentVolumeClaim is non-nil.
*	  We therefore require a PVC in the default namespace if the deployment's namespace is left unspecified
*	  or in the same namespace as the deployment.
*	StatefulSet: DONE
*	* Same as deployment
*
*	Service:
*	* A service.spec.selector is a map of key/value pairs, these key/value pairs must be present
*	  somewhere else. either a deployment or a stateful set.
**/

/**
* This script is designed to accept a []metav1.Object and send back a []*DependencyInformation
* (essentially it is the metav1.Object wrapped up with extra dependency information)
**/
func GetDependencyInformation(object metav1.Object, typed meta.Type) *DependencyInformation {
	// 1. Create the wrapped object
	d := &DependencyInformation{Object: object, Type: typed}
	// 2. Analyse common dependencies
	d.AnalyseCommonDependencies()
	// 3. Based on the concrete type of the metav1.Object, analyse those type-specific dependencies.
	switch concrete := object.(type) {
	case *appsv1.Deployment:
		d.AnalyseDeploymentDependencies(concrete)
	case *rbacV1beta1.RoleBinding:
		d.AnalyseRoleBindingDependencies(concrete)
	case *v1.Service:
		d.AnalyseServiceDependencies(concrete)
	default:
	}
	return d
}

/**
*   Common Dependencies:
*	* For every resource with a namespace, the namespace must exist.
**/
func (d *DependencyInformation) AnalyseCommonDependencies() {
	if d.Object.GetNamespace() == "" {
		return
	}
	d.Requirements = append(d.Requirements,
		fmt.Sprintf("The namespace %s must exist", d.Object.GetNamespace()),
	)
}

/**
* 	RoleBinding:
*	* roleRef.name refers to an existing Role, so a Role by this name must exist.
*	* subects[i].kind and name should refer to a resource in the namespace of the RoleBinding resource
**/
func (d *DependencyInformation) AnalyseRoleBindingDependencies(roleBinding *rbacV1beta1.RoleBinding) {
	reference := roleBinding.RoleRef.Name
	if reference != "" {
		message := fmt.Sprintf("There must be a Role %s ", reference)
		if roleBinding.Namespace != "" {
			message += fmt.Sprintf("in the namespace %s ", roleBinding.Namespace)
		}
		message += "or a ClusterRole of the same name in the global namespace"
		d.Requirements = append(d.Requirements, message)
	}
	for _, subject := range roleBinding.Subjects {
		requirement := fmt.Sprintf("There must be a %s called %s", subject.Kind, subject.Name)
		if subject.Namespace != "" {
			requirement += fmt.Sprintf(" in the namespace %s", subject.Namespace)
		}
		d.Requirements = append(d.Requirements, requirement)
	}
}

/**
*	Service:
*	* A service.spec.selector is a map of key/value pairs, these key/value pairs must be present
*	  somewhere else. either a deployment or a stateful set.
**/
func (d *DependencyInformation) AnalyseServiceDependencies(service *v1.Service) {
	// for example, in xcover-batch-app/production/db.yaml, the service has project:xcover-batch in its selector map,
	// This label is also present in the deployment.metadata.labels map
	for key, value := range service.Spec.Selector {
		requirement := fmt.Sprintf("There should be a Deployment or StatefulSet labelled with \"%s:%s\"", key, value)
		if service.Namespace != "" {
			requirement += fmt.Sprintf(" in the namespace %s", service.Namespace)
		}
		d.Requirements = append(d.Requirements, requirement)
	}
}

/**
*	Deployment:
*	* A deployment's pod spec may have a key 'volumes', within which there could be a reference to a
*	  PersistentVolumeClaim if a volume's VolumeSource.PersistentVolumeClaim is non-nil.
*	  We therefore require a PVC in the default namespace if the deployment's namespace is left unspecified
*	  or in the same namespace as the deployment.
**/
func (d *DependencyInformation) AnalyseDeploymentDependencies(deployment *appsv1.Deployment) {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			claimName := volume.VolumeSource.PersistentVolumeClaim.ClaimName
			requirement := fmt.Sprintf("There must be a PersistentVolumeClaim %s", claimName)
			if deployment.Namespace != "" {
				requirement += fmt.Sprintf("in the namespace %s", deployment.Namespace)
			}
			d.Requirements = append(d.Requirements, requirement)
		}
	}
}

/**
*	StatefulSet:
*	* Same as deployment
**/
func (d *DependencyInformation) AnalyseStatefulSetDependencies(statefulSet *appsv1.StatefulSet) {
	for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			claimName := volume.VolumeSource.PersistentVolumeClaim.ClaimName
			requirement := fmt.Sprintf("There must be a PersistentVolumeClaim %s", claimName)
			if statefulSet.Namespace != "" {
				requirement += fmt.Sprintf("in the namespace %s", statefulSet.Namespace)
			}
			d.Requirements = append(d.Requirements, requirement)
		}
	}
}
