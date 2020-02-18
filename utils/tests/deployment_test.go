package servicetest

import (
	"fmt"
	"os"
	"testing"

	"github.com/rdowavic/k8sutil/utils"
	lint "github.com/rdowavic/k8sutil/utils/lint"
	appsv1 "k8s.io/api/apps/v1"
)

var (
	deployment *appsv1.Deployment
	rules      map[lint.RuleID]*lint.Rule
)

type Test struct {
	Mutate   func()
	Expected bool
}

func TestDeploymentProjectLabel(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.DeploymentRules(resource))[lint.DEPLOYMENT_EXISTS_PROJECT_LABEL]
	d := resource.Resource.(*appsv1.Deployment)
	if !rule.Condition() {
		t.Errorf("The deployment's project label should be recognised, DEPLOYMENT_EXISTS_PROJECT_LABEL")
	}
	delete(d.Spec.Template.Labels, "project")
	if rule.Condition() {
		t.Errorf("The deployment's project label is missing, this test should fail: DEPLOYMENT_EXISTS_PROJECT_LABEL")
	}
}

func TestDeploymentContainerLiveness(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.DeploymentRules(resource))[lint.DEPLOYMENT_CONTAINER_EXISTS_LIVENESS]
	// original container should work
	if !rule.Condition() {
		t.Errorf("test DEPLOYMENT_CONTAINER_EXISTS_LIVENESS failed but yaml declared container liveness probe")
	}
	// make it break, test it now doesn't pass
	c.LivenessProbe = nil
	if rule.Condition() {
		t.Errorf("liveness probe key missing but DEPLOYMENT_CONTAINER_EXISTS_LIVENESS passed")
	}
}

func TestDeploymentContainerReadiness(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	d := resource.Resource.(*appsv1.Deployment)
	c := &d.Spec.Template.Spec.Containers[0]
	rule := CreateTestMap(lint.DeploymentRules(resource))[lint.DEPLOYMENT_CONTAINER_EXISTS_READINESS]
	// original container should work
	if !rule.Condition() {
		t.Errorf("test DEPLOYMENT_CONTAINER_EXISTS_READINESS failed but yaml declared container liveness probe")
	}
	// make it break, test it now doesn't pass
	c.ReadinessProbe = nil
	if rule.Condition() {
		t.Errorf("liveness probe key missing but DEPLOYMENT_CONTAINER_EXISTS_READINESS passed")
	}
}
func TestDeploymentAppK8sLabel(t *testing.T) {
	resource := lint.AttachMetaData(deploymentYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.DeploymentRules(resource))[lint.DEPLOYMENT_EXISTS_APP_K8S_LABEL]
	// set it to exist, check that test passes
	// automatic version, it exists
	if !rule.Condition() {
		t.Errorf("The deployment's app k8s io name label should be recognised")
	}
	d := resource.Resource.(*appsv1.Deployment)
	// set it to not exist, check that test fails
	delete(d.Spec.Template.Labels, "app.kubernetes.io/name")
	if rule.Condition() {
		t.Errorf("Missing label should cause test to fail")
	}
	// check that fix is correctly applied, check this.
	value := "jellyfish"
	d.Spec.Template.Labels["app"] = value
	rule.Fix()
	if v, ok := d.Spec.Template.Labels["app.kubernetes.io/name"]; !ok || v != value {
		t.Errorf("The rule's fix was not applied correctly")
	}
}

func TestDeploymentWithinNamespace(t *testing.T) {
	rule := rules[lint.DEPLOYMENT_WITHIN_NAMESPACE]
	if !rule.Condition() {
		t.Errorf("The static yaml is within a namespace, DEPLOYMENT_WITHIN_NAMESPACE should pass")
	}
	// mutate it to remove the namespace
	deployment.Namespace = ""
	if rule.Condition() {
		t.Errorf("The deployment is not within a namespace, DEPLOYMENT_WITHIN_NAMESPACE should fail")
	}
}

func TestDeploymentCorrectlySerialized(t *testing.T) {
	resources := lint.AttachMetaData(deploymentYaml, "/Users/racheldowavic/FAKEFILE.yaml")
	if len(resources) != 1 {
		t.Errorf("Expected %#v yaml resource, got %#v", 1, len(resources))
	}
	d, ok := resources[0].Resource.(*appsv1.Deployment)
	if !ok {
		t.Errorf("Wrong type decoded from metadata attacher")
	}
	if d.Name != "hello-world-web" {
		t.Errorf("Wrong name set")
	}
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 1 {
		t.Errorf("Number of replicas incorrectly set")
	}
	if d.Spec.Template.Spec.SecurityContext.RunAsNonRoot == nil ||
		*d.Spec.Template.Spec.SecurityContext.RunAsNonRoot != false {
		t.Errorf("RunAsNonRoot not correctly set")
	}
}

func TestMain(m *testing.M) {
	// set up
	// this large string is an example of a TOTALLY valid deployment. we will edit this in small ways
	// and make sure the linter fails on those ones.! :)
	// deserialise into appsv1.Deployment
	resources := lint.AttachMetaData(deploymentYaml, "Users/racheldowavic/FAKEFILE.yaml")
	if len(resources) != 1 {
		panic(fmt.Errorf("Metadata attacher couldn't correctly deserialize test yaml string"))
	}
	deployment = resources[0].Resource.(*appsv1.Deployment)
	// get the rules
	rules = CreateTestMap(lint.DeploymentRules(resources[0]))
	// tear-down
	os.Exit(m.Run())
}
