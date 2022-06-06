package cascadescenario

import (
	corev1 "k8s.io/api/core/v1"
	"gopkg.in/yaml.v3"
	"os"
	"github.com/randsw/cascadescenariocontroller/logger"
)

type CascadeScenario struct {
	// Configuration parameter for Cascade Module
	// +patchMergeKey=name
	// +patchStrategy=merge
	ModuleName    string            `json:"modulename"`
	Configuration map[string]string `json:"configuration"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Specifies the duration in seconds relative to the startTime that the job may be active
	// before the system tries to terminate it; value must be positive integer
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,3,opt,name=activeDeadlineSeconds"`

	// Specifies the number of retries before marking this job failed.
	// Defaults to 6
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty" protobuf:"varint,7,opt,name=backoffLimit"`

	// ttlSecondsAfterFinished limits the lifetime of a Job that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Job finishes, it is eligible to be
	// automatically deleted. When the Job is being deleted, its lifecycle
	// guarantees (e.g. finalizers) will be honored. If this field is unset,
	// the Job won't be automatically deleted. If this field is set to zero,
	// the Job becomes eligible to be deleted immediately after it finishes.
	// This field is alpha-level and is only honored by servers that enable the
	// TTLAfterFinished feature.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty" protobuf:"varint,8,opt,name=ttlSecondsAfterFinished"`

	// Describes the pod that will be created when executing a job.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
	Template corev1.PodTemplateSpec `json:"template" protobuf:"bytes,6,opt,name=template"`
}

func ReadConfigYAML (filename string) []CascadeScenario {
	var Config []CascadeScenario
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
    	logger.Zaplog.Error("Cant read config file")
	}
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
    	logger.Zaplog.Error("YAML unmarshal failed")
	}
	return Config
}
