package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var getContextCmd = &cobra.Command{
	Use:   "get-context",
	Short: "Show the current context name, cluster, namespace, and user details defined in the kubernetes config",
	Run: func(cmd *cobra.Command, args []string) {
		// Do some stuff in here
		configPath, exists := GetConfigPath()
		if !exists {
			log.Fatal("Kube config not found")
		}
		config, err := clientcmd.LoadFromFile(configPath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(
			`• Current Context: %s
• Cluster: %s (%s)
• Namespace: %s
• User: %s
   Client Certificate: %s
   Client Key: %s
`,
			config.CurrentContext,
			config.Contexts[config.CurrentContext].Cluster,
			config.Clusters[config.Contexts[config.CurrentContext].Cluster].Server,
			config.Contexts[config.CurrentContext].Namespace,
			config.Contexts[config.CurrentContext].AuthInfo,
			config.AuthInfos[config.Contexts[config.CurrentContext].AuthInfo].ClientCertificate,
			config.AuthInfos[config.Contexts[config.CurrentContext].AuthInfo].ClientKey,
		)
	},
}

func init() {
	RootCmd.AddCommand(getContextCmd)
}
