package lint

import (
	"regexp"

	v1 "k8s.io/api/core/v1"
)

func NamespaceRules(resource *YamlDerivedKubernetesResource) []*Rule {
	namespace, isNamespace := resource.Resource.(*v1.Namespace)
	if !isNamespace {
		return nil
	}

	return []*Rule{
		{
			ID: NAMESPACE_VALID_DNS,
			Condition: func() bool {
				validDNS := regexp.MustCompile(ACCEPTABLE_DNS)
				return validDNS.MatchString(namespace.Name)
			},
			Message:   "A Namespace needs to be a valid DNS name",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
	}
}
