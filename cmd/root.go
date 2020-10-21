package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"get.porter.sh/porter/pkg/porter"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/generator"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var bundleFileName string
var outputFileName string
var overwrite bool
var indent bool
var simplify bool
var customUI bool
var customRP bool
var includeCustomResource bool
var replaceKubeconfig bool
var timeout int
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

var getbundleCmd = &cobra.Command{
	Use:   "getbundle",
	Short: "Gets Bundle file for a tag",
	RunE: func(cmd *cobra.Command, args []string) error {

		bundle, _, err := common.PullBundle(&opts)
		if err != nil {
			return err
		}

		outputFile, err := getFile(bundleFileName, overwrite)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		err = common.WriteOutput(outputFile, bundle, indent)
		if err != nil {
			return fmt.Errorf("Error writing bundle file: %w", err)
		}

		err = outputFile.Sync()
		if err != nil {
			return fmt.Errorf("Error saving bundles file: %w", err)
		}

		return nil
	},
}

func getFile(fileName string, overwrite bool) (*os.File, error) {
	if err := checkFile(fileName, overwrite); err != nil {
		return nil, err
	}

	outputFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		return nil, fmt.Errorf("Error opening output file: %w", err)
	}
	return outputFile, nil
}

var rootCmd = &cobra.Command{
	Use:   "cnabtoarmtemplate",
	Short: "Generates an ARM template for executing a CNAB package using Azure driver",
	Long:  `Generates an ARM template which can be used to execute Porter in a deployment script, which in turn executes the CNAB Actions using the CNAB Azure Driver   `,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return common.ValidateTimeout(timeout)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		outputFile, err := getFile(outputFileName, overwrite)
		if err != nil {
			return err
		}

		defer outputFile.Close()

		var uiFile *os.File

		if customUI {
			uiFileName := path.Join(filepath.Dir(outputFileName), "createUIDefinition.json")
			uiFile, err = getFile(uiFileName, overwrite)
			if err != nil {
				return err
			}

			defer uiFile.Close()
		}

		options := common.BundleDetails{
			BundleLoc: bundleFileName,
			Options: common.Options{
				Indent:                indent,
				OutputWriter:          outputFile,
				Simplify:              simplify,
				Timeout:               timeout,
				GenerateUI:            customUI,
				CustomRPTemplate:      customRP,
				IncludeCustomResource: includeCustomResource,
				UIWriter:              uiFile,
				ReplaceKubeconfig:     replaceKubeconfig,
				BundlePullOptions:     &opts,
			},
		}
		err = generator.GenerateFiles(options)
		if err != nil {
			return fmt.Errorf("Error generating template: %w", err)
		}

		err = outputFile.Sync()
		if err != nil {
			return fmt.Errorf("Error saving output file: %w", err)
		}

		if customUI {
			err = uiFile.Sync()
			if err != nil {
				return fmt.Errorf("Error saving UI Definition file: %w", err)
			}
		}
		return nil
	},
}

func init() {

	rootCmd.Flags().StringVarP(&bundleFileName, "file", "f", "bundle.json", "name of bundle file to generate template for , default is bundle.json in the current directory")
	rootCmd.Flags().StringVarP(&outputFileName, "output", "o", "azuredeploy.json", "file name for generated template,default is azuredeploy.json")
	rootCmd.Flags().BoolVar(&overwrite, "overwrite", false, "specifies if to overwrite the output file if it already exists, default is false")
	rootCmd.Flags().BoolVarP(&indent, "indent", "i", false, "specifies if the json output should be indented")
	rootCmd.Flags().BoolVarP(&customUI, "customuidef", "c", false, "generates a custom createUIDefinition file called createUIdefinition.json in the same directory as the template")
	rootCmd.Flags().BoolVarP(&simplify, "simplify", "s", false, "specifies if the ARM template should be simplified, exposing less parameters and inferring default values")
	rootCmd.Flags().BoolVarP(&customRP, "customrp", "p", false, "generates a template to create a custom RP implemenation")
	rootCmd.Flags().BoolVarP(&includeCustomResource, "includeresource", "n", false, "causes the customRP template to include an instance of the type in addition to the resource and type definition")
	rootCmd.Flags().BoolVarP(&replaceKubeconfig, "replace", "r", false, "specifies if the ARM template generated should replace Kubeconfig Parameters with AKS references")
	rootCmd.Flags().IntVar(&timeout, "timeout", 15, "specifies the time in minutes that is allowed for execution of the CNAB Action in the generated template")
	rootCmd.Flags().StringVarP(&opts.Tag, "tag", "t", "", "Use a bundle specified by the given tag.")
	rootCmd.Flags().BoolVar(&opts.Force, "force", false, "Force a fresh pull of the bundle")
	rootCmd.Flags().BoolVar(&opts.InsecureRegistry, "insecure-registry", false, "Don't require TLS for the registry")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listenCmd)
	getbundleCmd.Flags().StringVarP(&bundleFileName, "file", "f", "bundle.json", "name of bundle file to write , default is bundle.json in the current directory")
	getbundleCmd.Flags().BoolVar(&overwrite, "overwrite", false, "specifies if to overwrite the output file if it already exists, default is false")
	getbundleCmd.Flags().StringVarP(&opts.Tag, "tag", "t", "", "Bundle tag to get bundle.json for.")
	getbundleCmd.Flags().BoolVar(&opts.Force, "force", false, "Force a fresh pull of the bundle")
	getbundleCmd.Flags().BoolVar(&opts.InsecureRegistry, "insecure-registry", false, "Don't require TLS for the registry")
	getbundleCmd.Flags().BoolVarP(&indent, "indent", "i", false, "specifies if the json output should be indented")
	err := getbundleCmd.MarkFlagRequired("tag")
	if err != nil {
		log.Infof("Error making Flag tag required: %v", err)
	}
	rootCmd.AddCommand(getbundleCmd)
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

func checkFile(dest string, overwrite bool) error {
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
