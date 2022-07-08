package main

import (
	"os"
	"strings"
	scenarioconfig "github.com/randsw/cascadescenariocontroller/cascadescenario"
	k8sClient "github.com/randsw/cascadescenariocontroller/k8sclient"
	"github.com/randsw/cascadescenariocontroller/logger"
	webhook "github.com/randsw/cascadescenariocontroller/webhook"
	"go.uber.org/zap"
)

const (
	notStarted k8sClient.JobStatus = iota
	Running
	Succeeded
	Failed
)

func main() {
	//Loger Initialization
	logger.InitLogger()

	//Get Config from file mounted in tmp folder
	configFilename := "/tmp/configuration"

	//configFilename := "cascadescenario/test/test_fail_first.json"

	CascadeScenatioConfig := scenarioconfig.ReadConfigJSON(configFilename)
	//Get pod namespace
	jobNamespace := "cascade-manual"
	if envvar := os.Getenv("POD_NAMESPACE"); len(envvar) > 0 {
		jobNamespace = envvar
	}
	//Get status server address
	statusServerAddress := "http://127.0.0.1:8000"
	if envvar := os.Getenv("STATUS_SERVER"); len(envvar) > 0 {
		statusServerAddress = envvar
	}
	//Get scenario name
	scenarioName := "Test-image-processing"
	if envvar := os.Getenv("SCENARIO_NAME"); len(envvar) > 0 {
		scenarioName = envvar
	}
	//Connect to k8s api server
	k8sAPIClientset := k8sClient.ConnectToK8s()

	// Initialize s3path struct
	s3PkgPath := new(k8sClient.S3PackagePath)
	for i, jobConfig := range CascadeScenatioConfig {
		// First stage. Get path from config
		if i == 0 {
			s3PkgPath.Path = jobConfig.Configuration["s3path"]
		}
		if i == len(CascadeScenatioConfig)-1 {
			s3PkgPath.IsLastStage = true
		}
		s3PkgPath.StageNum = i
		//Start k8s job
		k8sClient.LaunchK8sJob(k8sAPIClientset, jobNamespace, &jobConfig, s3PkgPath)
		start := true
		//Check for job status
		for {
			status, err := k8sClient.GetJobStatus(k8sAPIClientset, jobConfig.ModuleName, jobNamespace)
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
				err = k8sClient.DeleteSuccessJob(k8sAPIClientset, jobConfig.ModuleName, jobNamespace)
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
	split := strings.Split(s3PkgPath.Path, ".tgz")
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
