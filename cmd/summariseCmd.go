package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/rdowavic/k8sutil/utils"
	"github.com/rdowavic/k8sutil/utils/kubeapi"
	"github.com/rdowavic/k8sutil/utils/lint"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type Colour func(...interface{}) string

// these are populated by flags to the command line tool
var (
	kind                string
	groupByResourceKind bool
	showFilePath        bool
	remote              bool
	namespace           string
	showLabels          bool
)

var (
	cyan      Colour = color.New(color.FgHiCyan).SprintFunc()
	purple    Colour = color.New(color.FgHiBlue, color.FgHiRed).SprintFunc()
	blue      Colour = color.New(color.FgHiBlue).SprintFunc()
	green     Colour = color.New(color.FgHiGreen).SprintFunc()
	magenta   Colour = color.New(color.FgHiMagenta).SprintFunc()
	bold      Colour = color.New(color.Bold).SprintFunc()
	nameStyle Colour = color.New(color.Bold, color.FgHiYellow, color.Italic).SprintFunc()
	yellow    Colour = color.New(color.FgHiYellow).SprintFunc()
	boldRed   Colour = color.New(color.FgRed, color.Bold).SprintFunc()
)

var resourcesGroupedByKind map[string][]*utils.ResourceInfo
var resourcesGroupedByLabel map[string][]*utils.ResourceInfo

var summariseCmd = &cobra.Command{
	Use:   "summarise <file>*|-",
	Short: "Provide summary of all kubernetes resources in the given files and directories (also understands stdin)",
	Long: `An example of the default output format of this command is
$ xops service summarise-k8s -d ../xcover-kubernetes/xcover-batch-app/
Resource xcover-batch-production:
	Name: xcover-batch-production
	Kind: Namespace

Resource xcover-batch:
	Name: xcover-batch
	Namespace: xcover-batch-production
	Kind: ServiceAccount
...
With the --group-by-resource-kind flag set, the output will be grouped by resource kind. For example,
$ xops service summarise-k8s -d ../xcover-kubernetes/xcover-batch-app/ --group-by-resource-kind
Role:

	- xcover-batch in xcover-batch-production
	- jenkins-xcover-batch in xcover-batch-production
	- db in xcover-batch-production
	- minio in xcover-batch-production
	- redis in xcover-batch-production

Ingress:

	- xcover-batch-ingress in xcover-batch-production

PersistentVolumeClaim:

	- db-data in xcover-batch-production
	- minio-data in xcover-batch-production
	- redis-data in xcover-batch-production

`,
	Aliases: []string{"summarize"},
	Run: func(cmd *cobra.Command, args []string) {
		// try to do something random with the kubernetes API
		if remote && namespace == "" {
			// then this isn't a valid calling of the function
			fmt.Println("You need to provide a namespace if you want to retrieve remote resources")
			cmd.Usage()
			os.Exit(1)
		}

		var resources []*utils.ResourceInfo
		// parse args and directories and get a []runtime.Object
		if remote && namespace != "" {
			// create a config and use the kubeapi package to retrieve the resources in that namespace
			config, _ := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
			clientset, _ := kubernetes.NewForConfig(config)
			r, err := kubeapi.GetResources(clientset, namespace)
			if err != nil {
				log.Fatal(err)
			}
			resources = kubeapi.Convert(r)
			// put them into the group by kind
		}

		if len(args) == 0 || args[0] != "-" {
			fileNames, err := AggregateFiles(args, directories)
			if err != nil {
				log.Fatal(err)
			}
			resources, err = deserialise(fileNames)
			if err != nil {
				log.Fatal(err)
			}
		} else if len(args) == 1 && args[0] == "-" {
			var data []byte
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			resources, err = deserialiseBytes(data, "stdin")
			if err != nil {
				log.Fatal(err)
			}

		} else {
			cmd.Usage()
			os.Exit(1)
		}
		// loop through slice and print relevant information
		if groupByResourceKind {
			PrintGroupByKind(resources)
		} else if kind != "" {
			// if that kind is not in the global map thing, then log a fatal error
			if _, ok := GetResourcesGroupedByKind(resources)[kind]; !ok {
				// then no resources of this kind exist
				os.Exit(1)
			}
			PrintFilteredByKind(resources, kind)
		} else if showLabels {
			PrintGroupByLabel(resources)
		} else {
			PrintDefault(resources)
		}
	},
}

func aggregateLabels(resources []*utils.ResourceInfo) map[string][]string {
	m := make(map[string][]string)
	for _, resource := range resources {
		if resource.Labels == nil {
			continue
		}
		for label, labelValue := range resource.Labels {
			if _, exists := m[label]; !exists {
				m[label] = nil
			}
			m[label] = append(m[label], labelValue)
		}
	}
	return m
}

func PrintLabels(labels map[string][]string) {
	for label, labelValues := range labels {
		fmt.Printf("Label %s:\n%v\n", nameStyle(label), labelValues)
	}
}

func PrintFilteredByKind(resources []*utils.ResourceInfo, kind string) {
	filtered := GetResourcesGroupedByKind(resources)[kind]
	for _, resource := range filtered {
		fmt.Printf("%s", yellow(resource.Name))
		if resource.Namespace != "" {
			fmt.Printf(" in %s", green(resource.Namespace))
		}
		if showFilePath {
			fmt.Printf(" from %s", magenta(resource.FileName))
		}
		fmt.Println()
	}
}

func PrintDefault(resources []*utils.ResourceInfo) {
	for i, resource := range resources {
		if resource.Name == "" && resource.Namespace == "" && resource.Kind == "" {
			fmt.Printf("Resource %d: %s\n\n", i, boldRed("No Information Available"))
			continue
		}
		if resource.Name != "" {
			fmt.Printf("%s %s:\n", bold("Resource"), nameStyle(resource.Name))
		} else {
			fmt.Printf("%s %s:\n", bold("Resource"), nameStyle(strconv.Itoa(i)))
		}
		// print the contents of the resourceInfo struct
		if resource.Name != "" {
			fmt.Printf("\t%s: %s\n", cyan("Name"), resource.Name)
		}
		if resource.Namespace != "" {
			fmt.Printf("\t%s: %s\n", green("Namespace"), resource.Namespace)
		}
		if resource.Kind != "" {
			fmt.Printf("\t%s: %s\n", magenta("Kind"), resource.Kind)
		}
		if showFilePath && resource.FileName != "" {
			fmt.Printf("\t%s: %s\n", yellow("Filepath"), resource.FileName)
		}
		if showLabels && resource.Labels != nil {
			fmt.Printf("\t%s:\n", blue("Labels"))
			for k, v := range resource.Labels {
				fmt.Printf("\t\t%s: %s\n", purple(k), v)
			}
		}
		fmt.Println()
	}
}

func PrintGroupByKind(resources []*utils.ResourceInfo) {
	for kind, list := range GetResourcesGroupedByKind(resources) {
		if kind == "" {
			kind = "Missing Kind"
		}
		fmt.Printf("%s:\n\n", bold(kind))
		for _, resource := range list {
			fmt.Printf("\t- %s", yellow(resource.Name))
			if resource.Namespace != "" {
				fmt.Printf(" in %s", green(resource.Namespace))
			}
			if showFilePath {
				fmt.Printf(" from %s", magenta(resource.FileName))
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func PrintGroupByLabel(resources []*utils.ResourceInfo) {
	labelMap := GetResourcesGroupedByLabel(resources)
	for label, list := range labelMap {
		fmt.Printf("Label %s:\n", nameStyle(label))
		for _, resource := range list {
			if resource.Name != "" {
				fmt.Printf("\t%s: %s\n", cyan("Name"), resource.Name)
			}
			if resource.Namespace != "" {
				fmt.Printf("\t%s: %s\n", green("Namespace"), resource.Namespace)
			}
			if resource.Kind != "" {
				fmt.Printf("\t%s: %s\n", magenta("Kind"), resource.Kind)
			}
			if showFilePath && resource.FileName != "" {
				fmt.Printf("\t%s: %s\n", yellow("Filepath"), resource.FileName)
			}
			if showLabels && resource.Labels != nil {
				fmt.Printf("\t%s:\n", blue("Labels"))
				for k, v := range resource.Labels {
					fmt.Printf("\t\t%s: %s\n", purple(k), v)
				}
			}
		}
	}
	fmt.Println()
}

func GetResourcesGroupedByLabel(resources []*utils.ResourceInfo) map[string][]*utils.ResourceInfo {
	if len(resourcesGroupedByLabel) != 0 {
		return resourcesGroupedByLabel
	}
	// just to make the logic less ugly, and maps are references so this modifies the original groupedByLabel variable.
	m := resourcesGroupedByLabel
	// now start actually adding keys to the map
	for _, resource := range resources {
		if resource.Labels == nil {
			continue
		}
		for label := range resource.Labels {
			_, ok := m[label]
			if !ok {
				m[label] = nil
			}
			m[label] = append(m[label], resource)
		}
	}
	return resourcesGroupedByLabel
}

func GetResourcesGroupedByKind(resources []*utils.ResourceInfo) map[string][]*utils.ResourceInfo {
	if len(resourcesGroupedByKind) != 0 {
		// then it's already been populated
		return resourcesGroupedByKind
	}
	m := make(map[string][]*utils.ResourceInfo)
	for _, resource := range resources {
		kind := resource.Kind
		// if m[kind] doesn't exist, add an empty thing
		_, ok := m[kind]
		if !ok {
			m[kind] = nil
		}
		// add the resource to the nil slice
		// now it defos exists
		m[kind] = append(m[kind], resource)
	}
	resourcesGroupedByKind = m
	return m
}

func Convert(resource runtime.Object, b []byte, fileName string) (*utils.ResourceInfo, error) {
	r := &utils.ResourceInfo{FileName: fileName}
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(b, &m)
	r.Origin = m
	if err != nil {
		return nil, err
	}
	if object, conformsToMetaV1Object := resource.(metav1.Object); conformsToMetaV1Object {
		r.Name = object.GetName()
		r.Namespace = object.GetNamespace()
	}
	//Resource kind
	typed, err := meta.TypeAccessor(resource)

	if err == nil {
		r.Kind = typed.GetKind()
	} else {
		return nil, err
	}
	return r, nil
}

func init() {
	resourcesGroupedByKind = make(map[string][]*utils.ResourceInfo)
	resourcesGroupedByLabel = make(map[string][]*utils.ResourceInfo)
	RootCmd.AddCommand(summariseCmd)
	summariseCmd.Flags().StringSliceVarP(&directories, "directories", "d", nil, "A comma-separated list of directories to recursively search for YAML documents")
	summariseCmd.Flags().BoolVarP(&groupByResourceKind, "group-by-resource-kind", "g", false, "Group output by resource kind")
	summariseCmd.Flags().BoolVarP(&showFilePath, "show-file", "f", false, "Show which file this resource was read from")
	summariseCmd.Flags().StringVarP(&kind, "kind", "k", "", "Only show resources of a certain kind, eg Deployment.")
	summariseCmd.Flags().BoolVarP(&remote, "remote", "r", false, "Get resources from remote cluster")
	summariseCmd.Flags().BoolVarP(&showLabels, "show-labels", "l", false, "Show labels associated with the resource")
	summariseCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Show resources from only this namespace")
}

// This takes a list of filenames and returns a list of kubernetes runtime Objects
func deserialise(fileNames []string) ([]*utils.ResourceInfo, error) {
	var resources []*utils.ResourceInfo
	// turn the file into bytes, split by ---
	for _, yamlFileName := range fileNames {
		yamlFilePath, _ := filepath.Abs(yamlFileName)
		yamlContent, err := ioutil.ReadFile(yamlFilePath)
		if err != nil {
			return nil, err
		}
		r, err := deserialiseBytes(yamlContent, yamlFileName)
		if err != nil {
			continue
		} else {
			resources = append(resources, r...)
		}
	}
	// return the list
	return resources, nil
}

// This takes a byte array and returns a list of kubernetes runtime objects
func deserialiseBytes(yamlContent []byte, fileName string) ([]*utils.ResourceInfo, error) {
	// this is the resultant resource list
	var resources []*utils.ResourceInfo
	lineBreak := lint.DetectLineBreak(yamlContent)
	serialisedResources := bytes.Split(yamlContent, []byte(lineBreak+"---"+lineBreak))
	for _, resource := range serialisedResources {
		if strings.Trim(string(resource), lineBreak) == "" {
			continue
		}
		deserialised, _, err := scheme.Codecs.UniversalDeserializer().Decode(resource, nil, nil)
		var result *utils.ResourceInfo
		if err != nil {
			result, err = MakeResourceInformation(resource, fileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't read resource in %s: %v\n", fileName, err)
				continue
			}
		} else {
			result, err = Convert(deserialised, resource, fileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't read resource in %s: %v\n", fileName, err)
				continue
			}
		}
		// check throughout the map for a deep equality
		isDuplicate := false
		for _, other := range resourcesGroupedByKind[result.Kind] {
			if reflect.DeepEqual(other.Origin, result.Origin) {
				// don't actually add this one to the list OR the map! pls!
				isDuplicate = true
			}
		}
		if !isDuplicate {
			resources = append(resources, result)
			// also add to the map
			if _, ok := resourcesGroupedByKind[result.Kind]; !ok {
				resourcesGroupedByKind[result.Kind] = nil
			}
			resourcesGroupedByKind[result.Kind] = append(resourcesGroupedByKind[result.Kind], result)
		}
	}
	// return the list
	return resources, nil
}

func MakeResourceInformation(b []byte, fileName string) (*utils.ResourceInfo, error) {
	// this will be a single yaml like
	//apiVersion: apiextensions.k8s.io/v1beta1
	//kind: CustomResourceDefinition
	//metadata:
	//  name: hostendpoints.crd.projectcalico.org
	//spec:
	//  scope: Cluster
	//  group: crd.projectcalico.org
	//  versions:
	//    - name: v1
	//      served: true
	//      storage: true
	//  names:
	//    kind: HostEndpoint
	//    plural: hostendpoints
	//    singular: hostendpoint
	r := &utils.ResourceInfo{FileName: fileName}
	m := make(map[interface{}]interface{})
	err := yaml.Unmarshal(b, &m)
	r.Origin = m
	if err != nil {
		return nil, err
	}
	if kind, ok := m["kind"]; ok {
		r.Kind = kind.(string)
	} else {
		// something without a kind is just unacceptable
		return nil, errors.New("Resource missing 'kind' key")
	}
	if metadata, ok := m["metadata"]; ok {
		if metMap, ok := metadata.(map[interface{}]interface{}); ok {
			if name, ok := metMap["name"]; ok {
				r.Name = name.(string)
			}
			if namespace, ok := metMap["namespace"]; ok {
				r.Namespace = namespace.(string)
			}
		}
	}
	return r, nil
}
