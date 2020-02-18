package lint

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	v1beta1Extensions "k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	rbacV1beta1 "k8s.io/api/rbac/v1beta1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func InterdependentRules(context []*YamlDerivedKubernetesResource) []*Rule {
	var rules []*Rule
	// important objects to locate and refer to
	// The Namespace
	// Count number of service objects (should be zero or one)
	numServices := 0
	// We need to find the namespace first
	namespaces := findNamespaces(context)

	for _, wrapped := range context {
		switch wrapped.Resource.(type) {
		case *v1.Pod:
		case *v1.Namespace:
		case *v1.PersistentVolumeClaim:
		case *appsv1.Deployment:
		case *batchV1.Job:
		case *batchV1beta1.CronJob:
		case *v1beta1Extensions.Ingress:
		case *networkingV1.NetworkPolicy:
		case *rbacV1.Role:
		case *rbacV1beta1.RoleBinding:
		case *v1.ServiceAccount:
		case *v1.Service:
			numServices += 1
		}
		// all resources should be within the correct namespace, if there is one defined.
	}

	if len(namespaces) == 1 {
		for _, resource := range context {
			rules = append(rules, MatchingNamespace(resource, namespaces[0]))
		}
	}

	rules = append(rules,
		&Rule{
			ID: INTERDEPENDENT_AT_MOST_1_SERVICE,
			Condition: func() bool {
				return numServices <= 1
			},
			Message: "Only one service should be defined, if any",
			Level:   ERROR,
			Fix:     func() bool { return false },
		},
		&Rule{
			ID: INTERDEPENDENT_NAMESPACE_REQUIRED,
			Condition: func() bool {
				return len(namespaces) > 0
			},
			Message: "A new Namespace must be defined",
			Level:   ERROR,
			Fix:     func() bool { return false },
		},
		&Rule{
			ID: INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE,
			Condition: func() bool {
				// I can't apply this rule unless there is exactly one namespace defined.
				if len(namespaces) != 1 {
					return true
				}
				// try to find the network policy
				return containsNetworkPolicy(context)
			},
			Message:   "There should be a network policy defined for the namespace",
			Level:     ERROR,
			Resources: namespaces,
			Fix:       func() bool { return false },
		},
		&Rule{
			ID: INTERDEPENDENT_EXACTLY_1_NAMESPACE,
			Condition: func() bool {
				return len(namespaces) <= 1
			},
			Message:   "You shouldn't need to define more than one namespace per analysis unit",
			Level:     ERROR,
			Resources: namespaces,
			Fix:       func() bool { return false },
		},
	)
	return rules
}

func MatchingNamespace(resource *YamlDerivedKubernetesResource, namespace *YamlDerivedKubernetesResource) *Rule {
	typed, err := meta.TypeAccessor(resource.Resource)
	if err != nil {
		panic(err)
	}
	return &Rule{
		ID: INTERDEPENDENT_MATCHING_NAMESPACE,
		Condition: func() bool {
			// never perform this check on a namespace. all other resource types are fine
			if _, isNamespace := resource.Resource.(*v1.Namespace); isNamespace {
				return true
			}
			return resource.Resource.(metav1.Object).GetNamespace() ==
				namespace.Resource.(metav1.Object).GetName()
		},
		Message:   fmt.Sprintf("The %s's namespace is set incorrectly", typed.GetKind()),
		Level:     ERROR,
		Resources: []*YamlDerivedKubernetesResource{resource, namespace},
		Fix: func() bool {
			// cast to interface that will help us set the namespace uniformly
			// I've got a great idea! How cool is this. Assert that
			// every concrete object conforms to this interface Object, and it also conforms to Type.
			// I can use these assertions to basically change the metadata in a uniform way,
			// how exciting is THAT! :)
			resource.Resource.(metav1.Object).SetNamespace(namespace.Resource.(metav1.Object).GetName())
			return true
		},
		FixDescription: fmt.Sprintf("Set %s %s's namespace to %s\n",
			typed.GetKind(),
			resource.Resource.(metav1.Object).GetName(),
			namespace.Resource.(metav1.Object).GetName()),
	}
}

// END OF RULES

// helper functions to help analyse the environment
func findNamespaces(context []*YamlDerivedKubernetesResource) []*YamlDerivedKubernetesResource {
	var namespaces []*YamlDerivedKubernetesResource
	for _, resource := range context {
		if _, isNamespace := resource.Resource.(*v1.Namespace); isNamespace {
			namespaces = append(namespaces, resource)
		}
	}
	return namespaces
}

func containsNetworkPolicy(context []*YamlDerivedKubernetesResource) bool {
	for _, resource := range context {
		switch resource.Resource.(type) {
		case *networkingV1.NetworkPolicy:
			return true
		}
	}
	return false
}
