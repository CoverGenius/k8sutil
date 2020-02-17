package service

import (
	. "bitbucket.org/welovetravel/xops/service/lint"
	rbacV1 "k8s.io/api/rbac/v1"
)

func RoleRules(resource *YamlDerivedKubernetesResource) []*Rule {
	if _, isRole := resource.Resource.(*rbacV1.Role); !isRole {
		return nil
	}
	return nil
}
