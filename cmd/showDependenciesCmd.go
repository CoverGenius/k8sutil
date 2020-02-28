package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/CoverGenius/k8sutil/utils"
	"github.com/CoverGenius/k8sutil/utils/lint"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// these global variables are made necessary by the command's flags
var (
	showDirectories []string
)

var showDependenciesCmd = &cobra.Command{
	Use:   "show-dependencies <file>*|-",
	Short: "Show the dependencies implied by the given files and directories",
	// Long: w
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Collect all the filenames we need to read objects from
		filenames, err := AggregateFiles(args, showDirectories)
		if err != nil {
			log.Fatal(err)
		}
		// 2. Get corresponding file pointers for the filenames
		files, err := GetFiles(filenames, args)
		if err != nil {
			log.Fatal(err)
		}
		// 3. Turn the bytes into metav1.Object interface conformant things
		var yamlObjects []metav1.Object
		for _, file := range files {
			rawBytes, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatal(err)
			}
			buffer := bytes.NewBuffer(rawBytes)
			yamlObjects = append(yamlObjects, lint.ConvertToMetaV1Objects(buffer)...)
		}
		// 4. Turn the metav1.Objects into DependencyInformations so we can print the things it told us
		var dependencies []*utils.DependencyInformation
		for _, resource := range yamlObjects {
			// dodgy, i am sorry, but I want to get some type information into this dependencyInfo struct
			typed, err := meta.TypeAccessor(resource)
			if err != nil {
				continue
			}
			d := utils.GetDependencyInformation(resource, typed)
			dependencies = append(dependencies, d)
		}
		// 5. Pretty Print the Objects and some dependency information about them
		PrintDependencyInformation(dependencies)
	},
}

func PrintDependencyInformation(dependencies []*utils.DependencyInformation) {
	for _, d := range dependencies {
		if len(d.Requirements) == 0 {
			continue
		}
		fmt.Printf("%s %s (%s):\n", d.Type.GetKind(), nameStyle(d.Object.GetName()), cyan(d.Type.GetAPIVersion()))
		for _, message := range d.Requirements {
			fmt.Printf("\t%s %s\n", magenta("•"), message)
		}
		fmt.Println()
	}
}

func GetFiles(filenames []string, args []string) ([]*os.File, error) {
	var files []*os.File
	// consider stdin
	if len(args) == 1 && args[0] == "-" {
		files = append(files, os.Stdin)
	}
	// consider regular files
	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func init() {
	RootCmd.AddCommand(showDependenciesCmd)
	showDependenciesCmd.Flags().StringSliceVarP(&showDirectories, "directories", "d", []string{}, "A comma-separated list of directories to recursively search for YAML documents")
}
