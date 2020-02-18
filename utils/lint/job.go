package lint

import (
	"fmt"

	batchV1 "k8s.io/api/batch/v1"
)

func JobRules(resource *YamlDerivedKubernetesResource) []*Rule {
	job, isJob := resource.Resource.(*batchV1.Job)

	if !isJob {
		return nil
	}

	rules := []*Rule{
		// Check if the job has a namespace specified
		{
			ID: JOB_WITHIN_NAMESPACE,
			Condition: func() bool {
				return job.Namespace != ""
			},
			Message:   "A Job should have a namespace specified",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
		// Check if the job sets the restart policy appropriately
		{
			ID: JOB_RESTART_NEVER,
			Condition: func() bool {
				return len(job.Spec.Template.Spec.RestartPolicy) != 0 &&
					job.Spec.Template.Spec.RestartPolicy == "Never"

			},
			Message:   "A Job's restart policy should be set to Never",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix: func() bool {
				job.Spec.Template.Spec.RestartPolicy = "Never"
				return true
			},
			FixDescription: fmt.Sprintf("Set job %s's restart policy to Never", job.Name),
		},
		// Check if the job sets a ttl
		{
			ID: JOB_EXISTS_TTL,
			Condition: func() bool {
				return job.Spec.TTLSecondsAfterFinished != nil

			},
			Message:   "A Job should set a TTLSecondsAfterFinished value",
			Level:     ERROR,
			Resources: []*YamlDerivedKubernetesResource{resource},
			Fix:       func() bool { return false },
		},
	}
	rules = append(rules, PodRules(&job.Spec.Template.Spec, resource)...)
	return rules
}
