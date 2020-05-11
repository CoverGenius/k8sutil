package tests

import (
	"testing"

	lint "github.com/CoverGenius/k8sutil/utils/lint"
	appsv1 "k8s.io/api/apps/v1"
	rsc "k8s.io/apimachinery/pkg/api/resource"
	// v1 "k8s.io/api/core/v1"
)

func TestContainerExistsSecurityContext(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.ContainerRules(c, resource))[lint.CONTAINER_EXISTS_SECURITY_CONTEXT]
	if !rule.Condition() {
		t.Errorf("test CONTAINER_EXISTS_SECURITY_CONTEXT shouldn't have failed, check the static yaml")
	}
	c.SecurityContext = nil
	if rule.Condition() {
		t.Errorf("test CONTAINER_EXISTS_SECURITY_CONTEXT should have failed, security context was set to nil")
	}
}

func TestContainerAllowPrivilegeEscalation(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.ContainerRules(c, resource))[lint.CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE]
	// check correct yaml passes
	if !rule.Condition() {
		t.Errorf("rule CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE failed when the yaml was valid")
	}
	// check bad yaml fails
	bad := true
	c.SecurityContext.AllowPrivilegeEscalation = &bad
	if rule.Condition() {
		t.Errorf("rule CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE passed when the yaml was invalid")
	}
	// check the fix is applied correctly
	rule.Fix()
	if !rule.Condition() {
		t.Errorf("fix was not able to satisfy rule CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE after application")
	}
}

func TestContainerPrivileged(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.ContainerRules(c, resource))[lint.CONTAINER_PRIVILEGED_FALSE]
	if !rule.Condition() {
		t.Errorf("static yaml set container as privileged: false, test CONTAINER_PRIVILEGED_FALSE should succeed")
	}
	// make it break
	c.SecurityContext.Privileged = nil
	if rule.Condition() {
		t.Errorf("Privileged key not present on container, CONTAINER_PRIVILEGED_FALSE should fail")
	}
	// check that the fix works
	rule.Fix()
	if !rule.Condition() {
		t.Errorf("Container privilege fix applied, but test CONTAINER_PRIVILEGED_FALSE still failed")
	}
}

func TestContainerResourceLimits(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.ContainerRules(c, resource))[lint.CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS]
	// original container should work
	if !rule.Condition() {
		t.Errorf("test CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS failed but resource limits were set")
	}
	// make it break, test it now doesn't pass
	c.Resources.Limits = nil
	if rule.Condition() {
		t.Errorf("resource limits key missing but CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS passed")
	}
}

func TestContainerImageNotAllowed(t *testing.T) {
	rule := rules[lint.CONTAINER_VALID_IMAGE]
	// the deployment object is initially correctly set
	if !rule.Condition() {
		t.Errorf("test CONTAINER_VALID_IMAGE should have succeeded as the static yaml has a valid image")
	}
	deployment.Spec.Template.Spec.Containers[0].Image = "INVALID"
	if rule.Condition() {
		t.Errorf("test CONTAINER_VALID_IMAGE should fail, the image is INVALID")
	}
}

func TestContainerRequests(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.ContainerRules(c, resource))[lint.CONTAINER_REQUESTS_CPU_REASONABLE]
	// original container should work
	if !rule.Condition() {
		t.Errorf("test CONTAINER_REQUESTS_CPU_REASONABLE failed but resource cpu request was 0.5")
	}
	// make it break, test it now doesn't pass
	q, err := rsc.ParseQuantity("1.2")
	if err != nil {
		t.Fatal("Conversion of quantity value 1.2 failed")
	}
	c.Resources.Requests["cpu"] = q
	if rule.Condition() {
		t.Errorf("requests 1.2 cpu but CONTAINER_REQUESTS_CPU_REASONABLE still passed")
	}
}
