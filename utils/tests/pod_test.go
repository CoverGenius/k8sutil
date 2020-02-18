package servicetest

import (
	lint "github.com/rdowavic/k8sutil/utils/lint"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestPodCorrectNumContainers(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.DeploymentRules(resource))[lint.POD_EXACTLY_1_CONTAINER]
	// the static string does only have 1 container
	d := resource.Resource.(*appsv1.Deployment)
	if !rule.Condition() {
		t.Errorf("Deployment has 1 container, test should not fail")
	}
	containers := d.Spec.Template.Spec.Containers
	d.Spec.Template.Spec.Containers = append(containers, v1.Container{})
	if rule.Condition() {
		t.Errorf("Expected there to be 2 containers, there was %v, test should fail", len(d.Spec.Template.Spec.Containers))
	}
}

func TestDeploymentUserGroupID(t *testing.T) {
	rule := rules[lint.POD_CORRECT_USER_GROUP_ID]
	tests := []Test{
		{
			Mutate: func() {
				deployment.Spec.Template.Spec.SecurityContext.RunAsUser = nil
			},
			Expected: false,
		},
		{
			Mutate: func() {
				deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = nil
			},
			Expected: false,
		},
		{
			Mutate: func() {
				runAsUser := int64(44444)
				runAsGroup := int64(44444)
				deployment.Spec.Template.Spec.SecurityContext.RunAsUser = &runAsUser
				deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = &runAsGroup
			},
			Expected: true,
		},
		{
			Mutate: func() {
				runAsUser := int64(1000)
				runAsGroup := int64(44444)
				deployment.Spec.Template.Spec.SecurityContext.RunAsUser = &runAsUser
				deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = &runAsGroup

			},
			Expected: false,
		},
		{
			Mutate: func() {
				runAsUser := int64(10000)
				runAsGroup := int64(10000)
				deployment.Spec.Template.Spec.SecurityContext.RunAsUser = &runAsUser
				deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = &runAsGroup

			},
			Expected: false,
		},
	}

	for _, tc := range tests {
		tc.Mutate()
		result := rule.Condition()
		if result != tc.Expected {
			t.Errorf("expected rule pass: %v, but got %v for rule POD_CORRECT_USER_GROUP_ID",
				tc.Expected,
				result,
			)
		}
	}
}
