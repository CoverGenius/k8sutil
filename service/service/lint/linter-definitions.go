package lint

/**
* This contains definitions for linting objects and anything else required by
* the linting process.
**/

import (
	"k8s.io/apimachinery/pkg/runtime"
)

type Level int

const (
	SUCCESS = 0
	ERROR   = 1
	WARN    = 2
)

// Represents a Linter Rule
type Rule struct {
	ID             RuleID
	Prereqs        []RuleID
	Condition      func() bool
	Message        string
	Level          Level
	Resources      []*YamlDerivedKubernetesResource
	Fix            func() bool
	FixDescription string
}

type YamlDerivedKubernetesResource struct {
	Resource runtime.Object
	Metadata
}

// Captures metadata when parsing yaml objects from text files
type Metadata struct {
	LineNumber int
	FilePath   string
}

// I'm defining an enum for all the test IDs so we know at compile time whether
// we're trying to access a non-existent test. The old way caused a panic at runtime
// and was a bit silly.
type RuleID int

const (
	// container tests
	CONTAINER_EXISTS_SECURITY_CONTEXT = iota
	CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE
	CONTAINER_VALID_IMAGE
	CONTAINER_PRIVILEGED_FALSE
	CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS
	CONTAINER_REQUESTS_CPU_REASONABLE
	// deployment specific tests
	DEPLOYMENT_EXISTS_PROJECT_LABEL
	DEPLOYMENT_EXISTS_APP_K8S_LABEL
	DEPLOYMENT_WITHIN_NAMESPACE
	DEPLOYMENT_CONTAINER_EXISTS_LIVENESS
	DEPLOYMENT_CONTAINER_EXISTS_READINESS
	DEPLOYMENT_API_VERSION
	DEPLOYMENT_LIVENESS_READINESS_NONMATCHING
	// cronjob tests
	CRONJOB_WITHIN_NAMESPACE
	CRONJOB_FORBID_CONCURRENT
	// network policy rules
	NETWORK_POLICY_API_VERSION
	// interdependent rules
	INTERDEPENDENT_AT_MOST_1_SERVICE
	INTERDEPENDENT_NAMESPACE_REQUIRED
	INTERDEPENDENT_NETWORK_POLICY_FOR_NAMESPACE
	INTERDEPENDENT_EXACTLY_1_NAMESPACE
	INTERDEPENDENT_MATCHING_NAMESPACE
	// job rules
	JOB_WITHIN_NAMESPACE
	JOB_RESTART_NEVER
	JOB_EXISTS_TTL
	// namespace rules
	NAMESPACE_VALID_DNS
	// pod spec rules
	POD_NON_NIL_SECURITY_CONTEXT
	POD_RUN_AS_NON_ROOT
	POD_CORRECT_USER_GROUP_ID
	POD_NON_ZERO_CONTAINERS
	POD_EXACTLY_1_CONTAINER
	// service rules
	SERVICE_WITHIN_NAMESPACE
	SERVICE_NAME_VALID_DNS
)

const ACCEPTABLE_DNS = `^[a-zA-Z][a-zA-Z0-9\-\.]+[a-zA-Z0-9]$`
