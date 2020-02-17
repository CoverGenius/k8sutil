package service

import (
	. "bitbucket.org/welovetravel/xops/service/lint"
	v1beta1Extensions "k8s.io/api/extensions/v1beta1"
)

func IngressRules(resource *YamlDerivedKubernetesResource) []*Rule {
	if _, isIngress := resource.Resource.(*v1beta1Extensions.Ingress); !isIngress {
		return nil
	}
	return nil
}
