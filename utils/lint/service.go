package lint

import (
	"regexp"

	v1 "k8s.io/api/core/v1"
)

func ServiceRules(resource *YamlDerivedKubernetesResource) []*Rule {
	service, isService := resource.Resource.(*v1.Service)
	if !isService {
		return nil
	}
	return []*Rule{
		{
			ID: SERVICE_WITHIN_NAMESPACE,
			Condition: func() bool {
				return service.Namespace != ""
			},
			Message:   "A service should have a namespace specified",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID: SERVICE_NAME_VALID_DNS,
			Condition: func() bool {
				validDNS := regexp.MustCompile(ACCEPTABLE_DNS)
				return validDNS.MatchString(service.Name)
			},
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Message:   "A service's name needs to be a valid DNS",
			Fix:       func() bool { return false }, // we can't fix this ourselves
		},
	}
}
