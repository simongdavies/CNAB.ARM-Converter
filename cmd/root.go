package main

import (
	"fmt"
	"os"

	"get.porter.sh/porter/pkg/porter"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	"github.com/spf13/cobra"
)

var fileloc string
var outputloc string
var overwrite bool
var indent bool
var simplify bool
var opts porter.BundlePullOptions

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the cnabtoarmtemplate version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cnabtoarmtemplate-%v \n", Version())
	},
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Starts an http server to listen for request for template generation",
	Run: func(cmd *cobra.Command, args []string) {
		Listen()
	},
}

var rootCmd = &cobra.Command{
	Use:   "cnabtoarmtemplate",
	Short: "Generates an ARM template for executing a CNAB package using Azure driver",
	Long:  `Generates an ARM template which can be used to execute Porter in a deployment script, which in turn executes the CNAB Actions using the CNAB Azure Driver   `,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		if err := checkOutputFile(outputloc, overwrite); err != nil {
			return err
		}

		file, err := os.OpenFile(outputloc, os.O_RDWR|os.O_CREATE, 0644)

		if err != nil {
			return fmt.Errorf("Error opening output file: %w", err)
		}

		defer file.Close()

		options := generator.GenerateTemplateOptions{
			BundleLoc: fileloc,
			GenerateOptions: generator.GenerateOptions{
				Indent:            indent,
				Writer:            file,
				Simplify:          simplify,
				BundlePullOptions: &opts,
			},
		}

		err = generator.GenerateTemplate(options)
		if err != nil {
			return fmt.Errorf("Error generating template: %w", err)
		}

		err = file.Sync()
		if err != nil {
			return fmt.Errorf("Error saving output file: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&fileloc, "file", "f", "bundle.json", "name of bundle file to generate template for , default is bundle.json in the current directory")
	rootCmd.Flags().StringVarP(&outputloc, "output", "o", "azuredeploy.json", "file name for generated template,default is azuredeploy.json")
	rootCmd.Flags().BoolVar(&overwrite, "overwrite", false, "specifies if to overwrite the output file if it already exists, default is false")
	rootCmd.Flags().BoolVarP(&indent, "indent", "i", false, "specifies if the json output should be indented")
	rootCmd.Flags().BoolVarP(&simplify, "simplify", "s", false, "specifies if the ARM template should be simplified, exposing less parameters and inferring default values")
	rootCmd.Flags().StringVarP(&opts.Tag, "tag", "t", "", "Use a bundle specified by the given tag.")
	rootCmd.Flags().BoolVar(&opts.Force, "force", false, "Force a fresh pull of the bundle")
	rootCmd.Flags().BoolVar(&opts.InsecureRegistry, "insecure-registry", false, "Don't require TLS for the registry")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listenCmd)
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

func checkOutputFile(dest string, overwrite bool) error {
	if _, err := os.Stat(dest); err == nil {
		if !overwrite {
			return fmt.Errorf("File %s exists and overwrite not specified", dest)
		}
		if err := os.Truncate(dest, 0); err != nil {
			return fmt.Errorf("File %s exists and truncate failed with error:%w", dest, err)
		}
	} else {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to access output file: %s. %w", dest, err)
		}
	}
	return nil
}
