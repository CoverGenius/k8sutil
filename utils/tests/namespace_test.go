package servicetest

import (
	lint "github.com/rdowavic/k8sutil/utils/lint"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func TestNamespaceValidDNS(t *testing.T) {
	resource := lint.AttachMetaData(namespaceYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.NamespaceRules(resource))[lint.NAMESPACE_VALID_DNS]
	namespace := resource.Resource.(*v1.Namespace)
	if !rule.Condition() {
		t.Errorf("Namespace's name was %s but NAMESPACE_VALID_DNS still failed", namespace.Name)
	}
	namespace.Name = "1AbsoluteRubbish"
	if rule.Condition() {
		t.Errorf("Namespace's name was NAMESPACE_VALID_DNS but %s still passed", namespace.Name)
	}
	namespace.Name = ""
	if rule.Condition() {
		t.Errorf("Namespace's name was empty but NAMESPACE_VALID_DNS still passed")
	}
}
