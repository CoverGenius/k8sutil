package service

import (
	"fmt"

	. "bitbucket.org/welovetravel/xops/service/lint"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
)

func CronJobRules(resource *YamlDerivedKubernetesResource) []*Rule {
	job, isCronJob := resource.Resource.(*batchV1beta1.CronJob)

	if !isCronJob {
		return nil
	}

	rules := []*Rule{
		{
			ID: CRONJOB_WITHIN_NAMESPACE,
			Condition: func() bool {
				return job.Namespace != ""
			},
			Message: "The resource must be within a namespace",
			Level:   ERROR,
			Fix:     func() bool { return false },
		},
		{
			ID: CRONJOB_FORBID_CONCURRENT,
			Condition: func() bool {
				return job.Spec.ConcurrencyPolicy == batchV1beta1.ForbidConcurrent
			},
			Message: "Concurent operations should be forbidden",
			Level:   ERROR,
			Fix: func() bool {
				job.Spec.ConcurrencyPolicy = batchV1beta1.ForbidConcurrent
				return true
			},
			FixDescription: fmt.Sprintf("Set concurrency policy on cronjob %s to forbid concurrent", job.Name),
		},
	}
	rules = append(rules, PodRules(&job.Spec.JobTemplate.Spec.Template.Spec, resource)...)
	// also apply container rules to the nested container
	return rules
}
