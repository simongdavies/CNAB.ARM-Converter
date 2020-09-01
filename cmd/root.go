package main

import (
	"fmt"
	"os"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/spf13/cobra"
)

var bundleloc string
var outputloc string
var overwrite bool
var indent bool
var simplify bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the cnabarmdriver version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cnabarmdriver-%v \n", Version())
	},
}

var rootCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates an ARM template for executing a CNAB package using Azure driver",
	Long:  `Generates an ARM template which can be used to execute Porter in a container using ACI to perform actions on a CNAB Package, which in turn executes the CNAB Actions using the CNAB Azure Driver   `,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		options := generator.GenerateTemplateOptions{
			BundleLoc:  bundleloc,
			Indent:     indent,
			OutputFile: outputloc,
			Overwrite:  overwrite,
			Version:    Version(),
			Simplify:   simplify,
		}

		return generator.GenerateTemplate(options)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&bundleloc, "bundle", "b", "bundle.json", "name of bundle file to generate template for , default is bundle.json")
	rootCmd.Flags().StringVarP(&outputloc, "file", "f", "azuredeploy.json", "file name for generated template,default is azuredeploy.json")
	rootCmd.Flags().BoolVarP(&overwrite, "overwrite", "o", false, "specifies if to overwrite the output file if it already exists, default is false")
	rootCmd.Flags().BoolVarP(&indent, "indent", "i", false, "specifies if the json output should be indented")
	rootCmd.Flags().BoolVarP(&simplify, "simplify", "s", false, "specifies if the ARM template should be simplified, exposing less parameters and inferring default values")

	rootCmd.AddCommand(versionCmd)
}

// Execute runs the template generator
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Version returns the version string
func Version() string {
	return fmt.Sprintf("%v-%v", pkg.Version, pkg.Commit)
}
