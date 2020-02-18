package lint

import (
	v1 "k8s.io/api/core/v1"
)

func ServiceAccountRules(resource *YamlDerivedKubernetesResource) []*Rule {
	if _, isServiceAccount := resource.Resource.(*v1.ServiceAccount); !isServiceAccount {
		return nil
	}
	return nil
}
