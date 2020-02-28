package tests

import (
	"testing"

	lint "github.com/CoverGenius/k8sutil/utils/lint"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	interdependentRules map[lint.RuleID]*lint.Rule
	context             []*lint.YamlDerivedKubernetesResource
)

func TestInterdependentOneService(t *testing.T) {
	SetGlobals()
	// basic preloaded context should pass because yaml in constants.go is correct
	rule := interdependentRules[lint.INTERDEPENDENT_AT_MOST_1_SERVICE]
	if !rule.Condition() {
		t.Errorf("Context contains one service exactly but INTERDEPENDENT_AT_MOST_1_SERVICE still fails")
	}
	// add two services, it should break!
	contextWithTwoServices := lint.AttachMetaData(validUnitYaml, "FAKE.yaml")
	for _, resource := range contextWithTwoServices {
		if _, isService := resource.Resource.(*v1.Service); isService {
			contextWithTwoServices = append(contextWithTwoServices, resource)
			break
		}
	}
	// now the contextWithServices should contain two pointers to the same resource,
	r := CreateTestMap(lint.InterdependentRules(contextWithTwoServices))[lint.INTERDEPENDENT_AT_MOST_1_SERVICE]
	if r.Condition() {
		t.Errorf("Context contains two services but INTERDEPENDENT_AT_MOST_1_SERVICE still passes")
	}
	// take the service away, it should still pass
	contextWithNoServices := lint.AttachMetaData(validUnitYaml, "FAKE.yaml")
	r = CreateTestMap(lint.InterdependentRules(contextWithNoServices))[lint.INTERDEPENDENT_AT_MOST_1_SERVICE]
	for _, resource := range contextWithNoServices {
		if _, isService := resource.Resource.(*v1.Service); !isService {
			contextWithNoServices = append(contextWithNoServices, resource)
		}
	}
	if !r.Condition() {
		t.Errorf("Unit contains no services but INTERDEPENDENT_AT_MOST_1_SERVICE still fails")
	}
}

func TestInterdependentNamespaceRequired(t *testing.T) {
	SetGlobals()
	// make sure this one passes because there is one namespace exactly.
	rule := interdependentRules[lint.INTERDEPENDENT_AT_MOST_1_SERVICE]
	if !rule.Condition() {
		t.Errorf("Unit yaml had one namespace but INTERDEPENDENT_AT_MOST_1_SERVICE failed anyway")
	}
	// get rid of namespace from context
	var c []*lint.YamlDerivedKubernetesResource
	// exclude all namespace objects from this slice now
	for _, resource := range context {
		if _, isNamespace := resource.Resource.(*v1.Namespace); !isNamespace {
			c = append(c, resource)
		}
	}
	r := CreateTestMap(lint.InterdependentRules(c))[lint.INTERDEPENDENT_NAMESPACE_REQUIRED]
	if r.Condition() {
		t.Errorf("Context provides no namespace but INTERDEPENDENT_NAMESPACE_REQUIRED still passes")
	}

}

func TestInterdependentNetworkPolicyForNamespace(t *testing.T) {
	SetGlobals()
	// original context should pass
	rule := interdependentRules[lint.INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE]
	if !rule.Condition() {
		t.Errorf("Original unit contains network policy for the namespace but INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE still fails")
	}
	// delete the network policy, make sure it still fails
	var c []*lint.YamlDerivedKubernetesResource
	// exclude all namespace objects from this slice now
	for _, resource := range context {
		if _, isNetworkPolicy := resource.Resource.(*networkingV1.NetworkPolicy); !isNetworkPolicy {
			c = append(c, resource)
		}
	}
	r := CreateTestMap(lint.InterdependentRules(c))[lint.INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE]
	if r.Condition() {
		t.Errorf("No network policy was defined in the unit but INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE still passes")
	}
}

func TestMatchingNamespace(t *testing.T) {
	SetGlobals()
	// original unit has all the namespaces set properly! they all should pass.
	r := lint.InterdependentRules(context) // need a list because the ID Is not unique

	for _, rule := range r {
		if rule.ID == lint.INTERDEPENDENT_MATCHING_NAMESPACE {
			if !rule.Condition() {
				t.Errorf("Original unit set namespaces correctly for %s but INTERDEPENDENT_MATCHING_NAMESPACE still fails",
					rule.Resources[0].Resource.(metav1.Object).GetName(),
				)
			}
		}
	}
	// set one of them incorrectly or something and make sure it fails and the fix is
	// correctly applied
	// grab the deployment
	var deployment *appsv1.Deployment
	for _, resource := range context {
		if d, ok := resource.Resource.(*appsv1.Deployment); ok {
			deployment = d
		}
	}
	deployment.Namespace = "Absolutely Rubbish Namespace, not even DNS conformant"
	// find the deployment namespace rule
	for _, rule := range r {
		if rule.ID == lint.INTERDEPENDENT_MATCHING_NAMESPACE {
			if d, ok := rule.Resources[0].Resource.(*appsv1.Deployment); ok &&
				d == deployment {
				if rule.Condition() {
					t.Errorf("Deployment's namespace was set to some rubbish but INTERDEPENDENT_MATCHING_NAMESPACE still passed")
				}
			}
		}
	}
}

func SetGlobals() {
	// load in the objects and the rules for the context,
	// don't create a new context for everything. Just try to reset it after mutating please.
	// we might actually have to just create local copies but try to avoid it. So costly!
	context = lint.AttachMetaData(validUnitYaml, "FAKE.yaml")
	interdependentRules = CreateTestMap(lint.InterdependentRules(context))
}
