package tests

import (
	"testing"

	lint "github.com/CoverGenius/k8sutil/utils/lint"
	batchV1 "k8s.io/api/batch/v1"
)

func TestJobWithinNamespace(t *testing.T) {
	resource := lint.AttachMetaData(jobYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.JobRules(resource))[lint.JOB_WITHIN_NAMESPACE]
	if !rule.Condition() {
		t.Errorf("Job was within namespace but JOB_WITHIN_NAMESPACE did not pass")
	}
	// break it, make sure it fails
	resource.Resource.(*batchV1.Job).Namespace = ""
	if rule.Condition() {
		t.Errorf("Job was not within namespace but JOB_WITHIN_NAMESPACE passed")
	}

}

func TestJobNeverRestarts(t *testing.T) {
	resource := lint.AttachMetaData(jobYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.JobRules(resource))[lint.JOB_RESTART_NEVER]
	if !rule.Condition() {
		t.Errorf("Job was set to restart: Never but JOB_RESTART_NEVER still failed")
	}
	job := resource.Resource.(*batchV1.Job)
	// should fail now
	job.Spec.Template.Spec.RestartPolicy = "Always"
	if rule.Condition() {
		t.Errorf("Job's restart policy was set to Always but JOB_RESTART_NEVER still passed")
	}
	job.Spec.Template.Spec.RestartPolicy = ""
	if rule.Condition() {
		t.Errorf("Job's restart policy was missing but JOB_RESTART_NEVER still passed")
	}
	// apply fix
	rule.Fix()
	if !rule.Condition() {
		t.Errorf("Restart policy fix was applied but JOB_RESTART_NEVER still failed")
	}
}

func TestJobExistsTTL(t *testing.T) {
	resource := lint.AttachMetaData(jobYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(lint.JobRules(resource))[lint.JOB_EXISTS_TTL]
	job := resource.Resource.(*batchV1.Job)
	if !rule.Condition() {
		t.Errorf("TTL was set on Job but JOB_EXISTS_TTL still failed")
	}
	job.Spec.TTLSecondsAfterFinished = nil
	if rule.Condition() {
		t.Errorf("TTL was missing on Job but JOB_EXISTS_TTL still passed")
	}
}
