package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Selection struct {
	Context   string
	Namespace string
}

var selections []*Selection

var workonCmd = &cobra.Command{
	Use:   "workon",
	Short: "Set the current context (cluster and namespace) for kubectl",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, configExists := GetConfigPath()
		if !configExists {
			log.Fatal("Config not found")
		}
		// this is NON API stuff
		config, _ := clientcmd.LoadFromFile(configPath)
		// type clientcmdapi.Config

		// for each context that is defined, it specifies a cluster,
		// and we want to find out all the namespaces under that cluster.
		// We should be able to retrieve them if the user specified by the context has those permissions.
		for name, c := range config.Contexts {
			// modify the config object in place
			// set the current context to the one we're looking at right now
			config.CurrentContext = name
			restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
			if err != nil {
				log.Fatal(err)
			}
			clientset, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				log.Fatal(err)
			}
			namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			if err != nil {
				// If we get any kind of error, we show the current namespace set in the
				// context and if one is not, we specfy default
				var namespace string
				if len(c.Namespace) != 0 {
					namespace = c.Namespace
				} else {
					namespace = "default"
				}

				selections = append(selections, &Selection{Context: name, Namespace: namespace})
			} else {
				for _, namespace := range namespaces.Items {
					// this will be one of the possible selections
					selections = append(selections, &Selection{Context: name, Namespace: namespace.Name})
				}
			}
		}
		// by this point, the selections slice is completely instantiated
		chosen, err := fuzzyfinder.Find(
			selections,
			func(i int) string {
				return fmt.Sprintf("%s %s", selections[i].Context, selections[i].Namespace)
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				if i == -1 {
					return ""
				}
				return fmt.Sprintf("🏡 Context: %s\n🥑 Namespace: %s\n✨ Cluster: %s\n🤖 Cluster Server: %s",
					selections[i].Context, selections[i].Namespace, config.Contexts[selections[i].Context].Cluster,
					config.Clusters[config.Contexts[selections[i].Context].Cluster].Server,
				)
			}))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Switching current context to %s and context %s's namespace to %s in the file %s...\n",
			selections[chosen].Context, selections[chosen].Context, selections[chosen].Namespace, configPath)
		// Write that change out to the config!
		// We have changed the clientcmdapi.Config and
		config.CurrentContext = selections[chosen].Context
		config.Contexts[selections[chosen].Context].Namespace = selections[chosen].Namespace
		// I want to write this out to the config path please. I look inside clientcmd and they have a lot of utils for this.
		err = clientcmd.WriteToFile(*config, configPath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(workonCmd)
}

func GetConfigPath() (string, bool) {
	// 1. check the path specified by the environment variable KUBECONFIG
	path, exists := os.LookupEnv("KUBECONFIG")
	// 2. default to ~/.kube/config
	if !exists {
		path = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	// 3. Check that that path points to a file that does exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", false
	}
	return path, true
}
