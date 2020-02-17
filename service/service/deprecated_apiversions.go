package service

import (
	"fmt"

	. "bitbucket.org/welovetravel/xops/service/lint"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeprecatedDeploymentAPIVersion(resource *YamlDerivedKubernetesResource) []*Rule {

	typeInformation, err := meta.TypeAccessor(resource.Resource)
	var name string
	if obj, ok := resource.Resource.(metav1.Object); ok {
		name = obj.GetName()
	}
	deprecatedAPIRules := []*Rule{
		{
			ID: DEPLOYMENT_API_VERSION,
			Condition: func() bool {
				return false
			},
			Message:   "Deployment should use apps/v1 API version",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				if err != nil {
					return false
				}
				typeInformation.SetAPIVersion("apps/v1")
				return true
			},
			FixDescription: fmt.Sprintf("Set Deployment %s's API Version to apps/v1", name),
		},
	}
	return deprecatedAPIRules
}

func DeprecatedNetworkPolicyAPIVersion(resource *YamlDerivedKubernetesResource) []*Rule {

	deprecatedAPIRules := []*Rule{
		{
			ID: NETWORK_POLICY_API_VERSION,
			Condition: func() bool {
				return false
			},
			Message:   "NetworkPolicy should use networking.k8s.io/v1 API version",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
	}
	return deprecatedAPIRules
}
