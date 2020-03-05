package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/rdowavic/kubelint"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var (
	Directories        = []string{}
	StandaloneLintMode bool
	Fix                bool
	OutPath            string
	Report             bool
)

var lintCmd = &cobra.Command{
	Use:   "lint <file>*",
	Short: "Lint YAML file(s) against a set of predefined kubernetes best practices",
	Run: func(cmd *cobra.Command, args []string) {
		// check that the flags they set make sense
		if Report && !Fix {
			log.Error("You can't request a report without specifying fix mode")
			fmt.Println(cmd.Usage())
			os.Exit(1)
		}
		if OutPath != "" && !Fix {
			log.Error("You can't request an output location without specifying fix mode")
			fmt.Println(cmd.Usage())
			os.Exit(1)
		}
		fmt.Println("HELLO, NEW LINTER!!!")
		// Prepare the linter.
		linter := kubelint.NewDefaultLinter()
		linter.AddAppsV1DeploymentRule(
			kubelint.APPSV1_DEPLOYMENT_EXISTS_PROJECT_LABEL,
			kubelint.APPSV1_DEPLOYMENT_EXISTS_APP_K8S_LABEL,
			kubelint.APPSV1_DEPLOYMENT_WITHIN_NAMESPACE,
			kubelint.APPSV1_DEPLOYMENT_CONTAINER_EXISTS_LIVENESS,
			kubelint.APPSV1_DEPLOYMENT_CONTAINER_EXISTS_LIVENESS,
			kubelint.APPSV1_DEPLOYMENT_LIVENESS_READINESS_NONMATCHING,
		)
		linter.AddV1PodSpecRule(
			kubelint.V1_PODSPEC_NON_NIL_SECURITY_CONTEXT,
			kubelint.V1_PODSPEC_RUN_AS_NON_ROOT,
			kubelint.V1_PODSPEC_CORRECT_USER_GROUP_ID,
			kubelint.V1_PODSPEC_EXACTLY_1_CONTAINER,
			kubelint.V1_PODSPEC_NON_ZERO_CONTAINERS,
		)
		linter.AddV1ContainerRule(
			kubelint.V1_CONTAINER_EXISTS_SECURITY_CONTEXT,
			kubelint.V1_CONTAINER_ALLOW_PRIVILEGE_ESCALATION_FALSE,
			kubelint.V1_CONTAINER_VALID_IMAGE,
			kubelint.V1_CONTAINER_PRIVILEGED_FALSE,
			kubelint.V1_CONTAINER_EXISTS_RESOURCE_LIMITS_AND_REQUESTS,
			kubelint.V1_CONTAINER_REQUESTS_CPU_REASONABLE,
		)
		linter.AddBatchV1Beta1CronJobRule(
			kubelint.BATCHV1_BETA1_CRONJOB_WITHIN_NAMESPACE,
			kubelint.BATCHV1_BETA1_CRONJOB_FORBID_CONCURRENT,
		)
		linter.AddBatchV1JobRule(
			kubelint.BATCHV1_JOB_WITHIN_NAMESPACE,
			kubelint.BATCHV1_JOB_RESTART_NEVER,
			kubelint.BATCHV1_JOB_EXISTS_TTL,
		)
		linter.AddV1NamespaceRule(
			kubelint.V1_NAMESPACE_VALID_DNS,
		)
		linter.AddV1ServiceRule(
			kubelint.V1_SERVICE_NAME_VALID_DNS,
			kubelint.V1_SERVICE_WITHIN_NAMESPACE,
			kubelint.V1_SERVICE_NAME_VALID_DNS,
		)
		// finished preparing the linter
		var results []*kubelint.Result
		var errs []error
		if len(args) == 1 && args[0] == "-" {
			r, e := linter.LintFile(os.Stdin)
			results = append(results, r...)
			errs = append(errs, e...)
		}
		filepaths, err := AggregateFiles(args, Directories)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%#v\n", filepaths)
		r, e := linter.Lint(filepaths...)
		results = append(results, r...)
		errs = append(errs, e...)
		// Now all our results are in the results array
		if len(errs) != 0 {
			for err := range errs {
				log.Error(err)
			}
			os.Exit(1)
		}
		logger := log.New()
		logger.SetOutput(os.Stdout)
		fmt.Println(len(results))
		for _, result := range results {
			logger.WithFields(log.Fields{
				"line number":   result.Resources[0].LineNumber,
				"filepath":      result.Resources[0].Filepath,
				"resource name": result.Resources[0].Resource.Object.GetName(),
			}).Log(result.Level, result.Message)
		}

		// write out the report if they want it!
		if Fix {
			resources, fixDescriptions := linter.ApplyFixes()
			byteRepresentation, errs := kubelint.Write(resources...)
			if len(errs) != 0 {
				for err := range errs {
					log.Error(err)
				}
				os.Exit(1)
			}
			// output to stdout by default
			fmt.Printf(string(byteRepresentation))
			if Report {
				ReportFixes(fixDescriptions)
			}
		}
	},
}

func ReportFixes(errorFixes []string) {
	green := color.New(color.FgHiGreen).SprintFunc()
	fmt.Fprintf(os.Stderr, "=====%s=====\n", bold("FIX SUMMARY"))
	for _, errorFix := range errorFixes {
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
	lintCmd.Flags().StringSliceVarP(&Directories, "directories", "d", []string{}, "A comma-separated list of directories to recursively search for YAML documents")
	lintCmd.Flags().BoolVarP(&StandaloneLintMode, "standalone-mode", "", false, "Standalone mode - only run lint on the specified resources and skips any dependency checks")
	lintCmd.Flags().BoolVar(&Fix, "fix", false, "apply fixes after identifying errors, where possible")
	lintCmd.Flags().StringVar(&OutPath, "fix-output", "", "output fixed yaml to file or folder instead of stdout")
	lintCmd.Flags().BoolVar(&Report, "fix-report", false, "report the successfully fixed errors")

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
