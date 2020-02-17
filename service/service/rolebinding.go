package service

import (
	. "bitbucket.org/welovetravel/xops/service/lint"
	rbacV1beta1 "k8s.io/api/rbac/v1beta1"
)

func RoleBindingRules(resource *YamlDerivedKubernetesResource) []*Rule {
	if _, isRoleBinding := resource.Resource.(*rbacV1beta1.RoleBinding); !isRoleBinding {
		return nil
	}
	return nil
}
