package service

// In each file in linter_rules directory, the only thing is just a list of these Rule structs. That's it.
import (
	. "bitbucket.org/welovetravel/xops/service/lint"
	"fmt"
	v1 "k8s.io/api/core/v1"
)

func ContainerRules(container *v1.Container, resource *YamlDerivedKubernetesResource) []*Rule {

	return []*Rule{
		// Every Container should have a security context
		{
			ID: CONTAINER_EXISTS_SECURITY_CONTEXT,
			Condition: func() bool {
				return container.SecurityContext != nil
			},
			Message:   "The container's security Context key is missing",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				container.SecurityContext = &v1.SecurityContext{}
				return true
			},
			FixDescription: fmt.Sprintf("Set container %s's security context to an empty map", container.Name),
		},
		{
			ID:      CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE,
			Prereqs: []RuleID{CONTAINER_EXISTS_SECURITY_CONTEXT},
			Condition: func() bool {
				return container.SecurityContext.AllowPrivilegeEscalation != nil &&
					*container.SecurityContext.AllowPrivilegeEscalation == false
			},
			Message:   "Expected AllowPrivilegeEscalation to be present and set to false",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				desired := false
				container.SecurityContext.AllowPrivilegeEscalation = &desired
				return true
			},
			FixDescription: fmt.Sprintf("Set AllowPrivilegeEscalation to false on Container %s", container.Name),
		},
		{
			ID: CONTAINER_VALID_IMAGE,
			Condition: func() bool {
				return isImageAllowed(container.Image)
			},
			Message:   fmt.Sprintf("The image from this registry is not allowed. Expected an image from: %#v, Got image: %#v", ALLOWED_DOCKER_REGISTRIES, container.Image),
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID:      CONTAINER_PRIVILEGED_FALSE,
			Prereqs: []RuleID{CONTAINER_EXISTS_SECURITY_CONTEXT},
			Condition: func() bool {
				return container.SecurityContext.Privileged != nil &&
					*container.SecurityContext.Privileged == false
			},
			Message:   "Expected Privileged to be present and set to false",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				privileged := false
				container.SecurityContext.Privileged = &privileged
				return true
			},
			FixDescription: fmt.Sprintf("Set Privileged key on container %s to false", container.Name),
		},
		{
			ID: CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS,
			Condition: func() bool {
				return container.Resources.Limits != nil && container.Resources.Requests != nil
			},
			Message:   "Resource limits must be set for the container (resources.requests) and (resources.limits)",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID:      CONTAINER_REQUESTS_CPU_REASONABLE,
			Prereqs: []RuleID{CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS},
			Condition: func() bool {
				// If the container is requesting CPU, it shouldn't be more than 1 unit.
				cpuUsage := container.Resources.Requests.Cpu()
				return cpuUsage.CmpInt64(1) != 1
			},
			Message:   "You should request less than 1 unit of CPU",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
	}
}
