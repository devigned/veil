package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/devigned/veil/core"
	"github.com/marstr/collection"
	"github.com/spf13/cobra"
)

const defaultTarget = "py3"

var (
	generateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Generate the binding for a Golang package",
		Long: `Give a set of target language / platforms generate bindings
for a Golang package in each of the targets`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			done := make(chan struct{})
			allGood := collection.AsEnumerable(targets, pkgPath, outDir).Enumerate(done).
				All(func(a interface{}) bool {
					if t, ok := a.([]string); ok {
						return len(t) > 0
					}
					return len(a.(string)) > 0
				})
			close(done)
			if !allGood {
				return core.NewUserError("Please provide --targets, --outdir and --pkg")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return NewGenerator(pkgPath, outDir, libName, targets).Execute()
		},
	}
	supportedTargets = []string{defaultTarget, "java"}

	targets []string
	pkgPath string
	outDir  string
	libName string
)

func init() {
	cwd, _ := os.Getwd()
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringSliceVarP(
		&targets,
		"targets",
		"t",
		[]string{defaultTarget},
		fmt.Sprintf("Targets for binding generation %s", supportedTargets))

	generateCmd.Flags().StringVarP(
		&pkgPath,
		"pkg",
		"p",
		"",
		"Path to Golang package to generate bindings (example github.com/devigned/veil/_examples/helloworld)")

	generateCmd.Flags().StringVarP(
		&outDir,
		"outdir",
		"o",
		path.Join(cwd, "output"),
		"Output directory to drop generated binding")

	generateCmd.Flags().StringVarP(
		&libName,
		"name",
		"n",
		"libgen",
		"Name of the CGo library to be generated in the output directory")
}
