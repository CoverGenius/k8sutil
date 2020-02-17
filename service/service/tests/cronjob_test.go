package servicetest

import (
	"bitbucket.org/welovetravel/xops/service"
	lint "bitbucket.org/welovetravel/xops/service/lint"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	"testing"
)

func TestCronjobWithinNamespace(t *testing.T) {
	resource := service.AttachMetaData(cronJobYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(service.CronJobRules(resource))[lint.CRONJOB_WITHIN_NAMESPACE]
	if !rule.Condition() {
		t.Errorf("Cronjob was within namespace but CRONJOB_WITHIN_NAMESPACE did not pass")
	}
	// break it, make sure it fails
	resource.Resource.(*batchV1beta1.CronJob).Namespace = ""
	if rule.Condition() {
		t.Errorf("Cronjob was not within namespace but CRONJOB_WITHIN_NAMESPACE passed")
	}
}

func TestCronjobForbidConcurrent(t *testing.T) {
	resource := service.AttachMetaData(cronJobYaml, "FAKEFILE.yaml")[0]
	rule := CreateTestMap(service.CronJobRules(resource))[lint.CRONJOB_FORBID_CONCURRENT]
	cronJob := resource.Resource.(*batchV1beta1.CronJob)
	if !rule.Condition() {
		t.Errorf("Cronjob did forbid concurrent operations but CRONJOB_FORBID_CONCURRENT did not pass")
	}
	// break it, make sure it fails
	cronJob.Spec.ConcurrencyPolicy = batchV1beta1.AllowConcurrent
	if rule.Condition() {
		t.Errorf("Cronjob set to allow concurrent operations but CRONJOB_FORBID_CONCURRENT passed")
	}
}

// The container within the cronjob will be testing by container_test.go
