package main

import (
	"context"
	"os"
	"strconv"
	"strings"

	scenarioconfig "github.com/randsw/cascadescenariocontroller/cascadescenario"
	"github.com/randsw/cascadescenariocontroller/logger"
	webhook "github.com/randsw/cascadescenariocontroller/webhook"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

type JobStatus int

const (
	notStarted JobStatus = iota
	Running
	Succeeded
	Failed
)

type s3PackagePath struct {
	stageNum    int
	path        string
	isLastStage bool
}

func connectToK8s() *kubernetes.Clientset {
	var kubeconfig string

	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfigkubeconfig := filepath.Join("~", ".kube", "config")
		if envvar := os.Getenv("KUBECONFIG"); len(envvar) > 0 {
			kubeconfig = envvar
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Zaplog.Error("The kubeconfig cannot be loaded", zap.String("err", err.Error()))
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Zaplog.Error("Failed to create K8s clientset")
		os.Exit(1)
	}

	return clientset
}

func launchK8sJob(clientset *kubernetes.Clientset, namespace string, config *scenarioconfig.CascadeScenarios, s3path *s3PackagePath) {
	jobs := clientset.BatchV1().Jobs(namespace)

	labels := map[string]string{"app": "Cascade", "modulename": config.ModuleName}
	// Get Job pod spec
	JobTemplate := config.Template
	// Get Scenario parameters
	ScenarioParameters := config.Configuration

	var podEnv []v1.EnvVar
	// Fill pod env vars with scenario parameters
	for key, value := range ScenarioParameters {
		podEnv = append(podEnv, v1.EnvVar{Name: key, Value: value})
	}
	if s3path.stageNum > 0 {
		split := strings.Split(s3path.path, ".tgz")
		podEnv = append(podEnv, v1.EnvVar{Name: "s3path", Value: split[0] + "-stage-" + strconv.Itoa(s3path.stageNum) + ".tgz"})
	}
	if s3path.isLastStage {
		podEnv = append(podEnv, v1.EnvVar{Name: "finalstage", Value: "true"})
	}
	JobTemplate.Spec.Containers[0].Env = podEnv

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ModuleName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template:                JobTemplate,
			TTLSecondsAfterFinished: config.TTLSecondsAfterFinished,
			BackoffLimit:            config.BackoffLimit,
			ActiveDeadlineSeconds:   config.ActiveDeadlineSeconds,
		},
	}

	_, err := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		logger.Zaplog.Fatal("Failed to create K8s job.", zap.String("error", err.Error()))
	}

	//print job details
	logger.Zaplog.Info("Created K8s job successfully", zap.String("JobName", config.ModuleName))
}

func getJobStatus(clientset *kubernetes.Clientset, jobName string, jobNamespace string) (JobStatus, error) {
	job, err := clientset.BatchV1().Jobs(jobNamespace).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err != nil {
		return notStarted, err
	}

	if job.Status.Active == 0 && job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		return notStarted, nil
	}

	if job.Status.Succeeded > 0 {
		logger.Zaplog.Info("Job ran successfully", zap.String("JobName", jobName))
		return Succeeded, nil // Job ran successfully
	}

	if job.Status.Failed > 0 {
		logger.Zaplog.Error("Job ran failed", zap.String("JobName", jobName))
		return Failed, nil // Job ran successfully
	}

	return Running, nil
}

func deleteSuccessJob(clientset *kubernetes.Clientset, jobName string, jobNamespace string) error {
	background := metav1.DeletePropagationBackground
	err := clientset.BatchV1().Jobs(jobNamespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{PropagationPolicy: &background})
	return err
}

func main() {
	//Loger Initialization
	logger.InitLogger()

	//Get Config from file mounted in tmp folder
	configFilename := "/tmp/configuration"

	//configFilename := "example_cm.yaml"

	CascadeScenatioConfig := scenarioconfig.ReadConfigJSON(configFilename)

	//Connect to k8s api server
	k8sApiClientset := connectToK8s()

	//Get pod namespace
	jobNamespace := "default"
	if envvar := os.Getenv("POD_NAMESPACE"); len(envvar) > 0 {
		jobNamespace = envvar
	}

	//Get status server address
	statusServerAddress := "127.0.0.1:8000"
	if envvar := os.Getenv("STATUS_SERVER"); len(envvar) > 0 {
		statusServerAddress = envvar
	}

	//Get status server address
	scenarioName := "Test-image-processing"
	if envvar := os.Getenv("SCENARIO_NAME"); len(envvar) > 0 {
		scenarioName = envvar
	}

	// Initialize s3path struct
	s3PackagePath := new(s3PackagePath)
	for i, jobConfig := range CascadeScenatioConfig {
		// First stage. Get path from config
		if i == 0 {
			s3PackagePath.path = jobConfig.Configuration["s3path"]
		}
		if i == len(CascadeScenatioConfig)-1 {
			s3PackagePath.isLastStage = true
		}
		s3PackagePath.stageNum = i
		//Start k8s job
		launchK8sJob(k8sApiClientset, jobNamespace, &jobConfig, s3PackagePath)
		start := true
		//Check for job status
		for {
			status, err := getJobStatus(k8sApiClientset, jobConfig.ModuleName, jobNamespace)
			if err != nil {
				logger.Zaplog.Error("Get Job status fail", zap.String("JobName", jobConfig.ModuleName), zap.String("error", err.Error()))
			}
			// Job starting
			if status == Running && start {
				logger.Zaplog.Info("Job started ", zap.String("JobName", jobConfig.ModuleName))
				start = false
			} else if status == Succeeded { // Job finished succesfuly
				statusCode, err := webhook.SendWebHook("Module "+jobConfig.ModuleName+" in scenario "+scenarioName+" finished successfully", statusServerAddress)
				if err != nil {
					logger.Zaplog.Error("Webhook failed", zap.String("error", err.Error()))
				}
				if statusCode != "200" {
					logger.Zaplog.Error("Webhook return fail code", zap.String("error", err.Error()))
				}
				// Delete finished Job
				err = deleteSuccessJob(k8sApiClientset, jobConfig.ModuleName, jobNamespace)
				if err != nil {
					logger.Zaplog.Error("Failed to delete successfull job", zap.String("JobName", jobConfig.ModuleName), zap.String("error", err.Error()))
				}
				break
			} else if status == Failed { // Job failed
				statusCode, err := webhook.SendWebHook("Module "+jobConfig.ModuleName+" in scenario "+scenarioName+" failed", statusServerAddress)
				if err != nil {
					logger.Zaplog.Error("Webhook failed", zap.String("error", err.Error()))
				}
				if statusCode != "200" {
					logger.Zaplog.Error("Webhook return fail code", zap.String("error", err.Error()))
				}
				logger.Zaplog.Error("Scenario execution failed", zap.String("Failed Job", jobConfig.ModuleName))
				os.Exit(1)
			}
		}
	}
	logger.Zaplog.Info("Scenario execution finished successfully")
	split := strings.Split(s3PackagePath.path, ".tgz")
	resultS3path := split[0] + "-final" + ".tgz"
	statusCode, err := webhook.SendWebHook("Scenario "+scenarioName+" completed successfully. Package address - "+resultS3path, statusServerAddress)
	if err != nil {
		logger.Zaplog.Error("Webhook failed", zap.String("error", err.Error()))
	}
	if statusCode != "200" {
		logger.Zaplog.Error("Webhook return fail code", zap.String("error", err.Error()))
	}
	os.Exit(0)
}
