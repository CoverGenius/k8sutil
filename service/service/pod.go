package service

import (
	"fmt"
	"log"

	. "bitbucket.org/welovetravel/xops/service/lint"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

func PodRules(podSpec *v1.PodSpec, resource *YamlDerivedKubernetesResource) []*Rule {
	outerResource, err := meta.Accessor(resource.Resource)
	outerResourceName := outerResource.GetName()
	if err != nil {
		log.Fatal(err)
	}
	rules := []*Rule{
		{
			ID: POD_NON_NIL_SECURITY_CONTEXT,
			Condition: func() bool {
				return podSpec.SecurityContext != nil
			},
			Message: "The Security context should be present",
			Fix: func() bool {
				podSpec.SecurityContext = &corev1.PodSecurityContext{}
				return true
			},
			Resources:      []*YamlDerivedKubernetesResource{resource},
			Level:          ERROR,
			FixDescription: fmt.Sprintf("Set resource %s's security context to an empty map", outerResourceName),
		},
		{
			ID:      POD_RUN_AS_NON_ROOT,
			Prereqs: []RuleID{POD_NON_NIL_SECURITY_CONTEXT},
			Condition: func() bool {
				return podSpec.SecurityContext.RunAsNonRoot != nil &&
					*podSpec.SecurityContext.RunAsNonRoot == true
			},
			Message: "The Pod Template Spec should enforce that any containers run as non-root users",
			Fix: func() bool {
				runAsNonRoot := true
				podSpec.SecurityContext.RunAsNonRoot = &runAsNonRoot
				return true
			},
			Level:          ERROR,
			Resources:      []*YamlDerivedKubernetesResource{resource},
			FixDescription: fmt.Sprintf("Set runAsNonRoot to true for resource %s", outerResourceName),
		},
		{
			ID:      POD_CORRECT_USER_GROUP_ID,
			Prereqs: []RuleID{POD_NON_NIL_SECURITY_CONTEXT},
			Condition: func() bool {
				return podSpec.SecurityContext.RunAsUser != nil &&
					podSpec.SecurityContext.RunAsGroup != nil &&
					*podSpec.SecurityContext.RunAsUser == 44444 &&
					*podSpec.SecurityContext.RunAsGroup == 44444
			},
			Message: "The user and group ID should be set to 44444",
			Fix: func() bool {
				userId := int64(44444)
				groupId := int64(44444)
				if podSpec.SecurityContext == nil {
					podSpec.SecurityContext = &corev1.PodSecurityContext{}
				}
				podSpec.SecurityContext.RunAsUser = &userId
				podSpec.SecurityContext.RunAsGroup = &groupId
				return true
			},
			Level:          ERROR,
			Resources:      []*YamlDerivedKubernetesResource{resource},
			FixDescription: fmt.Sprintf("Set resource %s's user ID and group ID to 44444", outerResourceName),
		},
		{
			ID: POD_EXACTLY_1_CONTAINER,
			Condition: func() bool {
				return len(podSpec.Containers) == 1
			},
			Message:   fmt.Sprintf("There should be exactly 1 container defined for %s, but there are %#v", outerResourceName, len(podSpec.Containers)),
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				return false // not possible to fix this by myself!  which one would they keep?
			},
		},
		{
			ID: POD_NON_ZERO_CONTAINERS,
			Condition: func() bool {
				return len(podSpec.Containers) != 0
			},
			Message:   fmt.Sprintf("%s should have at least 1 container defined", outerResourceName),
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				return false
			},
		},
	}
	// now add on the container rules!
	for i := range podSpec.Containers {
		rules = append(rules, ContainerRules(&podSpec.Containers[i], resource)...)
	}
	return rules
}
