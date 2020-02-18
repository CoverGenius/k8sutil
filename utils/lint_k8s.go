package utils

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/instrumenta/kubeval/kubeval"
	"github.com/instrumenta/kubeval/log"
	"github.com/rdowavic/k8sutil/utils/rulesorter"
	appsv1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	v1beta1Extensions "k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	rbacV1beta1 "k8s.io/api/rbac/v1beta1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ALLOWED_DOCKER_REGISTRIES []string = []string{"277433404353.dkr.ecr.eu-central-1.amazonaws.com"}
var errorFixes []string

func GetErrorFixes() []string { return errorFixes }

func isImageAllowed(image string) bool {
	for _, r := range ALLOWED_DOCKER_REGISTRIES {
		if strings.HasPrefix(image, r) {
			return true
		}
	}
	return false

}

func KubevalLint(data *bytes.Buffer, filename string) {
	config := kubeval.NewDefaultConfig()
	config.FileName = filename
	outputManager := kubeval.GetOutputManager(config.OutputFormat)
	results, err := kubeval.Validate(data.Bytes(), config)

	if err != nil {
		log.Error(err)
	}
	for _, result := range results {
		err = outputManager.Put(result)
	}
}

// main function that performs the linting
func Lint(k8sObjects []*YamlDerivedKubernetesResource, standaloneLintMode bool, fix bool) int {
	var ruleGroups [][]*Rule

	for _, wrappedObject := range k8sObjects {
		switch wrappedObject.Resource.(type) {
		case *v1.Namespace:
			ruleGroups = append(ruleGroups, NamespaceRules(wrappedObject))
		case *v1.PersistentVolumeClaim:
			ruleGroups = append(ruleGroups, PersistentVolumeClaimRules(wrappedObject))
		case *appsv1.Deployment:
			ruleGroups = append(ruleGroups, DeploymentRules(wrappedObject))
		case *v1beta1Extensions.Deployment:
			ruleGroups = append(ruleGroups, DeprecatedDeploymentAPIVersion(wrappedObject))
		case *batchV1.Job:
			ruleGroups = append(ruleGroups, JobRules(wrappedObject))
		case *batchV1beta1.CronJob:
			ruleGroups = append(ruleGroups, CronJobRules(wrappedObject))
		case *v1beta1Extensions.Ingress:
			ruleGroups = append(ruleGroups, IngressRules(wrappedObject))
		case *networkingV1.NetworkPolicy:
			ruleGroups = append(ruleGroups, NetworkPolicyRules(wrappedObject))
		case *v1beta1Extensions.NetworkPolicy:
			ruleGroups = append(ruleGroups, DeprecatedNetworkPolicyAPIVersion(wrappedObject))
		case *rbacV1.Role:
			ruleGroups = append(ruleGroups, RoleRules(wrappedObject))
		case *rbacV1beta1.RoleBinding:
			ruleGroups = append(ruleGroups, RoleBindingRules(wrappedObject))
		case *v1.ServiceAccount:
			ruleGroups = append(ruleGroups, ServiceAccountRules(wrappedObject))
		case *v1.Service:
			ruleGroups = append(ruleGroups, ServiceRules(wrappedObject))
		default:
			log.Error(fmt.Errorf("I don't know how to handle: %#v yet\n", wrappedObject))
		}
	}

	if !standaloneLintMode {
		ruleGroups = append(ruleGroups, InterdependentRules(k8sObjects))
	}
	// test each rule in the list. the exit code should be the highest of any rule tested that fails.
	var exitCode Level = SUCCESS
	for _, ruleList := range ruleGroups {
		// create the data structure!!
		ruleSorter := rulesorter.New(ruleList)
		// retrieve each rule as long as the ruleSorter isn't exhausted
		for !ruleSorter.IsEmpty() {
			rule := ruleSorter.PopNextAvailable()
			code, fixed := testRule(rule, fix)
			if code != SUCCESS && !fixed {
				// Then we need to make sure we don't test any rules dependent on this one!
				// They have already failede
				dangerousRules := ruleSorter.PopDependentRules(rule.ID)
				for _, rule := range dangerousRules {
					Report(rule)
				}
			}
			exitCode = updateExitCode(exitCode, code)
		}
	}
	return int(exitCode)
}

func updateExitCode(oldLevel Level, newLevel Level) Level {
	if newLevel == ERROR || oldLevel == ERROR {
		return ERROR
	}
	if newLevel == WARN && oldLevel != ERROR {
		return WARN
	}
	return oldLevel
}

func Report(rule *Rule) {
	switch rule.Level {
	case WARN:
		log.Warn(LinterMessage(rule.Message, rule.Resources))
	case ERROR:
		log.Error(fmt.Errorf(LinterMessage(rule.Message, rule.Resources)))
	}
}

func testRule(rule *Rule, userWantsFix bool) (Level, bool) {
	var exitCode Level = SUCCESS
	fixed := false
	if !rule.Condition() {
		exitCode = rule.Level
		Report(rule)
		if userWantsFix {
			fixed = rule.Fix()
			if fixed {
				errorFixes = append(errorFixes, rule.FixDescription)
			}
		}
	}
	return exitCode, fixed
}

func LinterMessage(message string, resources []*YamlDerivedKubernetesResource) string {
	switch len(resources) {
	case 0:
		return message
	case 1:
		typed, err := meta.TypeAccessor(resources[0].Resource)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%s:%d (%s %s): %s",
			filepath.Base(resources[0].FilePath), resources[0].LineNumber,
			typed.GetKind(), resources[0].Resource.(metav1.Object).GetName(),
			message)
	case 2:
		typed0, err := meta.TypeAccessor(resources[0].Resource)
		if err != nil {
			panic(err)
		}
		typed1, err := meta.TypeAccessor(resources[1].Resource)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%s:%d (%s %s) <-> %s:%d (%s %s): %s",
			filepath.Base(resources[0].FilePath), resources[0].LineNumber,
			typed0.GetKind(), resources[0].Resource.(metav1.Object).GetName(),
			filepath.Base(resources[1].FilePath), resources[1].LineNumber,
			typed1.GetKind(), resources[1].Resource.(metav1.Object).GetName(),
			message)
	default:
		panic(fmt.Errorf("Cannot handle the rule: %s, too many resources are involved", message))
	}
	return ""
}
