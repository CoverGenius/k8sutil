package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"bitbucket.org/welovetravel/xops/service"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serviceGenerateK8sCmd = &cobra.Command{
	Use:   "generate-k8s",
	Short: "Generate k8s YAML files",
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := args[0]
		s := service.ServiceData{
			Name:              serviceName,
			Project:           serviceProjectName,
			Role:              serviceRole,
			DeployEnvironment: deployEnvironment,
			Replicas:          numReplicas,
			WorkerReplicas:    numWorkerReplicas,
			NodeGroup:         nodeGroupName,
		}

		if serviceRole != "worker" {
			s.Port = servicePort
			s.Host = serviceHostName
			s.Type = serviceType
		}

		envData := make(map[string]string)
		for _, e := range envVars {
			kv := strings.Split(e, "=")
			if len(kv) != 2 {
				log.Fatal("Invalid env variable spec")
			}
			envData[kv[0]] = kv[1]
		}

		var secretEnvData []service.SecretKeyData

		for _, e := range secretEnvVars {
			ksv := strings.Split(e, "=")
			if len(ksv) != 2 {
				log.Fatal("Invalid secret env variable spec")
			}
			sv := strings.Split(ksv[1], "/")
			if len(sv) != 2 {
				log.Fatal("Invalid secret env variable spec")
			}

			s := service.SecretKeyData{
				Name:  ksv[0],
				Key:   sv[0],
				Value: sv[1],
			}
			secretEnvData = append(secretEnvData, s)
		}

		var secretVolumeData []service.SecretVolumeData

		for _, e := range secretVolumeMounts {
			ksv := strings.Split(e, "=")
			if len(ksv) != 2 {
				log.Fatal("Invalid secret volume mount spec")
			}
			sv := strings.Split(ksv[1], ";")
			if len(sv) != 2 {
				log.Fatal("Invalid secret volume mount spec")
			}
			s := service.SecretVolumeData{
				Name:   ksv[0],
				Path:   sv[0],
				Secret: sv[1],
			}

			secretVolumeData = append(secretVolumeData, s)

		}

		var pvClaimData []service.PVClaimData

		for _, e := range persistentVolumeMounts {
			ksv := strings.Split(e, "=")
			if len(ksv) != 2 {
				log.Fatal("Invalid persistent volume mount spec")
			}
			sv := strings.Split(ksv[1], ";")
			if len(sv) != 2 {
				log.Fatal("Invalid persistent volume mount spec")
			}
			n, err := strconv.ParseInt(sv[1], 10, 64)
			if err != nil {
				log.Fatal(err)
			}

			s := service.PVClaimData{
				Name:              fmt.Sprintf("%s-%s", serviceName, ksv[0]),
				Path:              sv[0],
				Size:              n,
				Project:           serviceProjectName,
				Role:              serviceRole,
				DeployEnvironment: deployEnvironment,
			}

			pvClaimData = append(pvClaimData, s)

		}

		c := service.ContainerData{
			Image:         containerImage,
			Environ:       envData,
			SecretEnviron: secretEnvData,
			SecretVolumes: secretVolumeData,
			PVClaims:      pvClaimData,
		}

		// Init container data
		var initContainerData []service.InitContainerData
		for _, ic := range initContainers {
			spec := strings.Split(ic, "=")
			if len(spec) != 2 {
				log.Fatal("Invalid init container specification")
			}
			icSpec := service.InitContainerData{Name: spec[0], Command: spec[1]}
			initContainerData = append(initContainerData, icSpec)
		}

		s.Container = c
		s.PVClaims = pvClaimData
		s.DbMigrationJob = dbMigrationJob
		s.DbMigrationTruncate = dbMigrationTruncate
		s.CronSchedule = cronJobSchedule
		s.CIServiceAccount = ciServiceAccount
		s.InitContainers = initContainerData
		service.GenerateK8sService(&s, directoryPath, forceWrite)
	},
	Args: cobra.ExactArgs(1),
}

var (
	directoryPath, cronJobSchedule, serviceRole, serviceType, serviceProjectName, nodeGroupName, deployEnvironment, containerImage, serviceHostName string
	envVars, secretEnvVars, secretVolumeMounts, persistentVolumeMounts, initContainers                                                              []string
	servicePort, numReplicas, numWorkerReplicas                                                                                                     int
	dbMigrationJob, dbMigrationTruncate, ciServiceAccount, forceWrite                                                                               bool
)

func init() {
	serviceCmd.AddCommand(serviceGenerateK8sCmd)
	serviceGenerateK8sCmd.Flags().IntVarP(&numReplicas, "replicas", "", 0, "Number of replicas")
	serviceGenerateK8sCmd.Flags().IntVarP(&numWorkerReplicas, "worker-replicas", "", 0, "Number of worker replicas")
	serviceGenerateK8sCmd.Flags().StringVarP(&serviceRole, "role", "", "", "Role of the service (web/worker)")
	serviceGenerateK8sCmd.Flags().StringVarP(&serviceHostName, "host", "", "", "FQDN for external web services")
	serviceGenerateK8sCmd.Flags().IntVarP(&servicePort, "port", "", 0, "Container service port")
	serviceGenerateK8sCmd.Flags().StringVarP(&serviceType, "type", "", "", "Type of the service (external/internal)")
	serviceGenerateK8sCmd.Flags().StringVarP(&serviceProjectName, "project", "", "", "Project name for the service")
	serviceGenerateK8sCmd.Flags().StringVarP(&containerImage, "container-image", "", "", "Container image to run in the pod (only AWS ECR registries allowed)")
	serviceGenerateK8sCmd.Flags().StringArrayVarP(&envVars, "environ", "", nil, "Environment variables and values (key=value)")
	serviceGenerateK8sCmd.Flags().StringArrayVarP(&secretEnvVars, "secret-environ", "", nil, "Secret environment variables and values (key=secret/key)")
	serviceGenerateK8sCmd.Flags().StringArrayVarP(&secretVolumeMounts, "secret-volumes", "", nil, "Secret volume mounts (name=/path/to/mount;secret)")
	serviceGenerateK8sCmd.Flags().StringArrayVarP(&persistentVolumeMounts, "pvc-claims", "", nil, "Persistent volume claims (name=/path/to/mount;5Gi)")
	serviceGenerateK8sCmd.Flags().StringVarP(&nodeGroupName, "node-group", "", "", "Node group name")
	serviceGenerateK8sCmd.Flags().StringVarP(&deployEnvironment, "deploy-environment", "", "", "Deployment Environment (dev/qa/staging/production)")
	serviceGenerateK8sCmd.Flags().StringVarP(&cronJobSchedule, "cron-schedule", "", "", "Cron job schedule")
	serviceGenerateK8sCmd.Flags().BoolVarP(&dbMigrationJob, "db-migrate", "", false, "Generate database migration job")
	serviceGenerateK8sCmd.Flags().BoolVarP(&dbMigrationTruncate, "db-migrate-truncate", "", false, "Truncate DB before migration")
	serviceGenerateK8sCmd.Flags().BoolVarP(&ciServiceAccount, "ci-service-account", "", false, "Create service account")
	serviceGenerateK8sCmd.Flags().StringArrayVarP(&initContainers, "init-container", "", nil, "Init container (name=command array)")
	serviceGenerateK8sCmd.Flags().StringVarP(&directoryPath, "outdir", "", "", "Directory to output the result to")
	serviceGenerateK8sCmd.Flags().BoolVar(&forceWrite, "force-write", false, "Write to an existing directory and overwrite any generated yaml files")
}
