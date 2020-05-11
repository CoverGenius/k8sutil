package lint

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

func DeploymentRules(resource *YamlDerivedKubernetesResource) []*Rule {
	deployment, isDeployment := resource.Resource.(*appsv1.Deployment)
	if !isDeployment {
		return nil
	}
	deploymentRules := []*Rule{
		// A Deployment should have a metadata label: project (?)
		{
			ID: DEPLOYMENT_EXISTS_PROJECT_LABEL,
			Condition: func() bool {
				_, found := deployment.Spec.Template.Labels["project"]
				return found
			},
			Message:   "There should be a project label present under the deployment's spec.template.labels",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		// A Deployment should have a metadata label: app.kubernetes.io/name
		{
			ID: DEPLOYMENT_EXISTS_APP_K8S_LABEL,
			Condition: func() bool {
				_, found := deployment.Spec.Template.Labels["app.kubernetes.io/name"]
				return found
			},
			Message:   "There should be an app.kubernetes.io/name label present for the deployment's spec.template",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				label, found := deployment.Spec.Template.Labels["app"]
				if found {
					delete(deployment.Spec.Template.Labels, "app")
					deployment.Spec.Template.Labels["app.kubernetes.io/name"] = label
					return true
				}
				return false
			},
			FixDescription: fmt.Sprintf("Found app label in deployment %s and used this value to populate the \"app.kubernetes.io/name\" key", deployment.Name),
		},
		// A Deployment should only use image from AWS ECR
		{
			ID: DEPLOYMENT_WITHIN_NAMESPACE,
			Condition: func() bool {
				return deployment.Namespace != ""
			},
			Message:   "The resource must be within a namespace",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID:      DEPLOYMENT_CONTAINER_EXISTS_LIVENESS,
			Prereqs: []RuleID{POD_NON_ZERO_CONTAINERS},
			Condition: func() bool {
				return deployment.Spec.Template.Spec.Containers[0].LivenessProbe != nil &&
					deployment.Spec.Template.Spec.Containers[0].LivenessProbe.Handler.HTTPGet != nil
			},
			Message:   "Expected declaration of liveness probe for the container (livenessProbe)",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID:      DEPLOYMENT_CONTAINER_EXISTS_READINESS,
			Prereqs: []RuleID{POD_NON_ZERO_CONTAINERS},
			Condition: func() bool {
				return deployment.Spec.Template.Spec.Containers[0].ReadinessProbe != nil &&
					deployment.Spec.Template.Spec.Containers[0].ReadinessProbe.Handler.HTTPGet != nil
			},
			Message:   "Expected declaration of readiness probe for the container (readinessProbe)",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		{
			ID:      DEPLOYMENT_LIVENESS_READINESS_NONMATCHING,
			Prereqs: []RuleID{POD_NON_ZERO_CONTAINERS, DEPLOYMENT_CONTAINER_EXISTS_READINESS, DEPLOYMENT_CONTAINER_EXISTS_LIVENESS},
			Condition: func() bool {
				container := deployment.Spec.Template.Spec.Containers[0]
				return container.LivenessProbe.Handler.HTTPGet.Path != container.ReadinessProbe.Handler.HTTPGet.Path
			},
			Message:   "It's recommended that the readiness and liveness probe endpoints don't match",
			Level:     WARN,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
	}
	// A Deployment object contains a PodSpec object and that contains a list of containers. We don't have to worry
	// about containers at this level. Pod will take care of this for us.
	deploymentRules = append(deploymentRules, PodRules(&deployment.Spec.Template.Spec, resource)...)
	return deploymentRules
}
