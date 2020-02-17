package service

import (
	. "bitbucket.org/welovetravel/xops/service/lint"
	networkingV1 "k8s.io/api/networking/v1"
)

func NetworkPolicyRules(resource *YamlDerivedKubernetesResource) []*Rule {
	if _, isNetworkPolicy := resource.Resource.(*networkingV1.NetworkPolicy); !isNetworkPolicy {
		return nil
	}
	return nil
}
