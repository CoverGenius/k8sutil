package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/CoverGenius/k8sutil/utils/lint"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	directories        = []string{}
	standaloneLintMode bool
	fix                bool
	outPath            string
	report             bool
)

var lintCmd = &cobra.Command{
	Use:   "lint <file>*",
	Short: "Lint YAML file(s) against a set of predefined kubernetes best practices",
	Run: func(cmd *cobra.Command, args []string) {
		// check that the flags they set make sense
		if report && !fix {
			log.Error("You can't request a report without specifying fix mode")
			fmt.Println(cmd.Usage())
			os.Exit(1)
		}
		if outPath != "" && !fix {
			log.Error("You can't request an output location without specifying fix mode")
			fmt.Println(cmd.Usage())
			os.Exit(1)
		}

		var yamlObjects []*lint.YamlDerivedKubernetesResource
		if len(args) == 0 || args[0] != "-" {
			files, err := AggregateFiles(args, directories)
			if err != nil {
				log.Fatal(err)
			}

			// for each file, we get the buffer so that we can process it through the attach-metadata and validate stages.
			for _, yamlFileName := range files {
				// file -> bytes array
				yamlFilePath, _ := filepath.Abs(yamlFileName)
				yamlContent, err := ioutil.ReadFile(yamlFilePath)
				if err != nil {
					log.Fatal(err)
				}
				// bytes array -> bytes.Buffer
				buffer := bytes.NewBuffer(yamlContent)
				lint.KubevalLint(buffer, filepath.Base(yamlFileName))
				// get all the yaml derived kubernetes objects out
				// and store it in a slice
				yamlObjects = append(yamlObjects, lint.AttachMetaData(buffer, yamlFilePath)...)
			}
		} else {
			var data []byte
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			buffer := bytes.NewBuffer(data)
			lint.KubevalLint(buffer, filepath.Base("stdin"))
			yamlObjects = append(yamlObjects, lint.AttachMetaData(buffer, filepath.Base("stdin"))...)
		}
		outPathIsDirectory := false
		// check for now whether this metadata attaching is working correctly (just verify by hand)
		exitCode := lint.Lint(yamlObjects, standaloneLintMode, fix)
		if fix {
			s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
			var f *os.File = os.Stdout
			var err error
			if outPath != "" {
				if filepath.Ext(outPath) == ".yaml" || filepath.Ext(outPath) == ".yml" {
					filePath, err := filepath.Abs(outPath)
					if err != nil {
						log.Fatal(err)
					}
					f, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					// assume it's a directory and on every iteration of the loop, try to make a .fixed version of it.
					outPathIsDirectory = true
				}
			}
			// use yamlObjects..
			for i, resource := range yamlObjects {
				if outPathIsDirectory {
					outPath, err = filepath.Abs(outPath)
					if err != nil {
						log.Fatal(err)
					}
					fileName := nameFixed(filepath.Base(resource.FilePath))
					// check if file al:ready exists. if so, open it in append mode and write a ---\n to it.
					filePath := filepath.Join(outPath, fileName)
					if _, err := os.Stat(filePath); err == nil {
						f, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0666)
						f.WriteString("---\n")
					} else {
						f, err = os.Create(filePath)
					}
					if err != nil {
						log.Fatal(err)
					}
					defer f.Close()
				}
				// delete the pesky creationTimeStamp field (this is what StripAndWrite is for)
				//	err := s.Encode(resource.Resource, f)
				//	if err != nil {
				//		log.Fatal(err)
				//	}
				yamlBytes, err := StripAndWriteToBytesSlice(s, resource.Resource)
				if err != nil {
					log.Fatal(err)
				}
				// now write out the bytes
				_, err = f.Write(yamlBytes)
				if err != nil {
					log.Fatal(err)
				}
				if !outPathIsDirectory && len(yamlObjects) > 1 && i != (len(yamlObjects)-1) {
					f.WriteString("---\n")
				}
			}
		}
		// write out the report if they want it!
		if fix && report {
			ReportFixes()
		}
		os.Exit(exitCode)
	},
}

func ReportFixes() {
	green := color.New(color.FgHiGreen).SprintFunc()
	fmt.Fprintf(os.Stderr, "=====%s=====\n", bold("FIX SUMMARY"))
	for _, errorFix := range lint.GetErrorFixes() {
		fmt.Fprintf(os.Stderr, " %s %s\n", green("✓"), errorFix)
	}
}

// Remove unnecessary fields from the runtime Object so that we don't accidentally get fields that are empty becoming marshalled.
// One example is that creationTimestamp: null always appears because apparently encoding/json cannot handle the empty struct
// and prints null anyway. Very annoying.
func StripAndWriteToBytesSlice(s *json.Serializer, o runtime.Object) ([]byte, error) {
	// we will need to remove the field using regexes
	var b bytes.Buffer
	err := s.Encode(o, &b)
	if err != nil {
		return nil, err
	}
	// now clean up creationTimestamp with this handy regex
	// in order to do this I need to turn this into a string. A little bit annoying
	str := b.String()
	re := regexp.MustCompile(" *creationTimestamp: null\n")
	cleaned := re.ReplaceAllString(str, "")
	return []byte(cleaned), nil
}

func nameFixed(fileName string) string {
	// place the extension .fixed before the last extension in the thing
	ext := filepath.Ext(fileName) // might be .yml or .yaml
	return fileName[:len(fileName)-len(ext)] + ".fixed" + ext
}

func init() {
	RootCmd.AddCommand(lintCmd)
	lintCmd.Flags().StringSliceVarP(&directories, "directories", "d", []string{}, "A comma-separated list of directories to recursively search for YAML documents")
	lintCmd.Flags().BoolVarP(&standaloneLintMode, "standalone-mode", "", false, "Standalone mode - only run lint on the specified resources and skips any dependency checks")
	lintCmd.Flags().BoolVar(&fix, "fix", false, "apply fixes after identifying errors, where possible")
	lintCmd.Flags().StringVar(&outPath, "fix-output", "", "output fixed yaml to file or folder instead of stdout")
	lintCmd.Flags().BoolVar(&report, "fix-report", false, "report the successfully fixed errors")

}

// copied and pasted directly from kubeval/main.go. I realy just needed to process the command line arguments
// and send them off. Is this plagiarism
func AggregateFiles(args []string, directories []string) ([]string, error) {
	files := make([]string, len(args))
	copy(files, args)

	var allErrors *multierror.Error
	for _, directory := range directories {
		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			allErrors = multierror.Append(allErrors, err)
		}
	}
	return files, allErrors.ErrorOrNil()
}
