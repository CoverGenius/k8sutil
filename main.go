package main

import (
	"fmt"
	"github.com/CoverGenius/k8sutil/cmd"
	"github.com/spf13/pflag"
	"os"
	"strings"
)

// Function copied from github.com/spf13/cobra
func nonCompletableFlag(flag *pflag.Flag) bool {
	return flag.Hidden || len(flag.Deprecated) > 0
}

func main() {
	// handle bash autocomplete
	if len(os.Getenv("COMP_LINE")) != 0 {
		if len(os.Getenv("COMP_DEBUG")) != 0 {
			fmt.Printf("%#v\n", os.Getenv("COMP_LINE"))
		}
		compLine := strings.Split(os.Getenv("COMP_LINE"), " ")

		// we only handle auto complete, if we are the command
		// being invoked
		if compLine[0] == cmd.RootCmd.Name() {
			var cmdArgs []string
			if len(compLine) > 1 {
				cmdArgs = compLine[1:]
			} else {
				cmdArgs = compLine[0:]
			}
			c, _, _ := cmd.RootCmd.Find(cmdArgs)
			suggestions := c.SuggestionsFor(cmdArgs[len(cmdArgs)-1])
			for _, s := range suggestions {
				fmt.Printf("%s\n", s)
			}

			localNonPersistentFlags := c.LocalNonPersistentFlags()
			c.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
				if nonCompletableFlag(flag) {
					return
				}
				if strings.HasPrefix(fmt.Sprintf("--%s", flag.Name), cmdArgs[len(cmdArgs)-1]) {
					fmt.Printf("--%s\n", flag.Name)
				}
				if localNonPersistentFlags.Lookup(flag.Name) != nil {
					if strings.HasPrefix(fmt.Sprintf("--%s", flag.Name), cmdArgs[len(cmdArgs)-1]) {
						fmt.Printf("--%s\n", flag.Name)
					}
				}
			})

			c.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
				if nonCompletableFlag(flag) {
					return
				}
				if strings.HasPrefix(flag.Name, cmdArgs[len(cmdArgs)-1]) {
					fmt.Printf("--%s\n", flag.Name)
				}
				if len(flag.Shorthand) > 0 {
					if strings.HasPrefix(flag.Name, cmdArgs[len(cmdArgs)-1]) {
						fmt.Printf("--%s\n", flag.Name)
					}
				}
			})

		}

	} else {
		cmd.Execute()
	}
}
