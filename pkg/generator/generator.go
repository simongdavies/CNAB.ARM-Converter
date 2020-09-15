package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"get.porter.sh/porter/pkg/porter"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/cnabio/cnab-to-oci/relocation"
	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
)

type GenerateOptions struct {
	Writer            io.Writer
	Indent            bool
	Simplify          bool
	ReplaceKubeconfig bool
	BundlePullOptions *porter.BundlePullOptions
}

// GenerateNestedDeploymentOptions is the set of options for configuring GenerateNestedDeployment
type GenerateNestedDeploymentOptions struct {
	Uri string
	GenerateOptions
}

// GenerateTemplateOptions is the set of options for configuring GenerateTemplate
type GenerateTemplateOptions struct {
	BundleLoc string
	GenerateOptions
}

// GenerateTemplate generates ARM template from bundle metadata
func GenerateNestedDeployment(options GenerateNestedDeploymentOptions) error {

	bundle, err := getBundleFromTag(options.BundlePullOptions)
	if err != nil {
		return err
	}

	generatedDeployment := template.NewCnabArmDeployment(bundle.Name, options.Uri, options.Simplify)

	parameterKeys, err := getParameterKeys(*bundle)
	if err != nil {
		return err
	}

	for _, parameterKey := range parameterKeys {

		parameter := bundle.Parameters[parameterKey]
		definition := bundle.Definitions[parameter.Definition]

		_, isCnabParam := isCnabParam(parameterKey)

		if !isCnabParam || (isCnabParam && !options.Simplify) {
			if options.ReplaceKubeconfig && strings.ToLower(parameterKey) == "kubeconfig" {
				generatedDeployment.Properties.Parameters["aksResourceGroupName"] = template.ParameterValue{
					Value: "TODO add value for aksResourceGroupName or delete this parameter to use default of current resource group",
				}
				generatedDeployment.Properties.Parameters["aksResourceName"] = template.ParameterValue{
					Value: "TODO add value for aksResourceName",
				}
			} else {
				defaultValue, required := getDefaultValue(definition, parameter)
				if reflect.TypeOf(defaultValue) == nil && required {
					defaultValue, _ = fmt.Printf("TODO Set Value for %s no default was provided and parameter is required", parameterKey)
				}
				generatedDeployment.Properties.Parameters[parameterKey] = template.ParameterValue{
					Value: defaultValue,
				}
			}
		}

	}

	credentialKeys, err := getCredentialKeys(*bundle)
	if err != nil {
		return err
	}

	for _, credentialKey := range credentialKeys {
		credential := bundle.Credentials[credentialKey]
		if options.ReplaceKubeconfig && strings.ToLower(credentialKey) == "kubeconfig" {
			generatedDeployment.Properties.Parameters["aksResourceGroupName"] = template.ParameterValue{
				Value: "TODO add value for aksResourceGroupName or delete this parameter to use default of current resource group",
			}
			generatedDeployment.Properties.Parameters["aksResourceName"] = template.ParameterValue{
				Value: "TODO add value for aksResourceName",
			}
		} else {
			defaultValue := fmt.Sprintf("TODO add value for %s", credentialKey)
			if !credential.Required {
				defaultValue = fmt.Sprintf("TODO add value or delete this entry as credential %s is optional", credentialKey)
			}
			generatedDeployment.Properties.Parameters[credentialKey] = template.ParameterValue{
				Value: defaultValue,
			}
		}
	}

	return writeResponseData(options.Writer, generatedDeployment, options.Indent)
}

// GenerateTemplate generates ARM template from bundle metadata
func GenerateTemplate(options GenerateTemplateOptions) error {

	bundle, bundleTag, err := getBundleDetails(options)
	if err != nil {
		return err
	}

	generatedTemplate, err := template.NewCnabArmDriverTemplate(
		bundle.Name,
		bundleTag,
		bundle.Outputs,
		options.Simplify,
	)

	if err != nil {
		return err
	}

	parameterKeys, err := getParameterKeys(*bundle)
	if err != nil {
		return err
	}

	for _, parameterKey := range parameterKeys {

		parameter := bundle.Parameters[parameterKey]
		definition := bundle.Definitions[parameter.Definition]
		paramEnvVar := template.EnvironmentVariable{
			Name: common.GetEnvironmentVariableNames().CnabParameterPrefix + parameterKey,
		}

		if options.ReplaceKubeconfig && strings.ToLower(parameterKey) == "kubeconfig" {
			paramEnvVar.SecureValue = "[listClusterAdminCredential(resourceId(subscription().subscriptionId,parameters('aksResourceGroupName'),'Microsoft.ContainerService/managedClusters',parameters('aksResourceName')), '2020-09-01').kubeconfigs[0].value]"
			setAKSParameters(generatedTemplate, bundle)
		} else if cnabParam, ok := isCnabParam(parameterKey); options.Simplify && ok {
			paramEnvVar.Value = fmt.Sprintf("[variables('%s')]", cnabParam)
		} else {

			var metadata template.Metadata
			if definition.Description != "" {
				metadata = template.Metadata{
					Description: definition.Description,
				}
			}

			var allowedValues interface{}
			if definition.Enum != nil {
				allowedValues = definition.Enum
			}

			defaultValue, _ := getDefaultValue(definition, parameter)

			var minValue *int
			if definition.Minimum != nil {
				minValue = definition.Minimum
			}
			if definition.ExclusiveMinimum != nil {
				min := *definition.ExclusiveMinimum + 1
				minValue = &min
			}

			var maxValue *int
			if definition.Maximum != nil {
				maxValue = definition.Maximum
			}
			if definition.ExclusiveMaximum != nil {
				max := *definition.ExclusiveMaximum - 1
				maxValue = &max
			}

			var minLength *int
			if definition.MinLength != nil {
				minLength = definition.MinLength
			}

			var maxLength *int
			if definition.MaxLength != nil {
				maxLength = definition.MaxLength
			}

			isSensitive := false
			if definition.WriteOnly != nil && *definition.WriteOnly {
				isSensitive = true
			}

			armType, err := toARMType(definition.Type.(string), isSensitive)
			if err != nil {
				return err
			}

			generatedTemplate.Parameters[parameterKey] = template.Parameter{
				Type:          armType,
				AllowedValues: allowedValues,
				DefaultValue:  defaultValue,
				Metadata:      &metadata,
				MinValue:      minValue,
				MaxValue:      maxValue,
				MinLength:     minLength,
				MaxLength:     maxLength,
			}

			if isSensitive {
				paramEnvVar.SecureValue = fmt.Sprintf("[parameters('%s')]", parameterKey)
			} else {
				paramEnvVar.Value = fmt.Sprintf("[parameters('%s')]", parameterKey)
			}

		}

		if err = generatedTemplate.SetDeploymentScriptEnvironmentVariable(paramEnvVar); err != nil {
			return err
		}
	}

	credentialKeys, err := getCredentialKeys(*bundle)
	if err != nil {
		return err
	}

	for _, credentialKey := range credentialKeys {

		credential := bundle.Credentials[credentialKey]

		var metadata template.Metadata
		var description string
		var defaultValue interface{}
		var envVarName string

		if credential.Description != "" {
			description = credential.Description
		}

		if credential.Path != "" {
			if description != "" {
				description += " "
			}
			description += "(Enter base64 encoded representation of file)"
			envVarName = common.GetEnvironmentVariableNames().CnabCredentialFilePrefix + credentialKey
		} else {
			envVarName = common.GetEnvironmentVariableNames().CnabCredentialPrefix + credentialKey
		}

		if description != "" {
			metadata = template.Metadata{
				Description: description,
			}
		}

		if !credential.Required {
			defaultValue = ""
		}

		credEnvVar := template.EnvironmentVariable{
			Name: envVarName,
		}

		if options.ReplaceKubeconfig && strings.ToLower(credentialKey) == "kubeconfig" {
			credEnvVar.SecureValue = "[listClusterAdminCredential(resourceId(subscription().subscriptionId,parameters('aksResourceGroupName'),'Microsoft.ContainerService/managedClusters',parameters('aksResourceName')), '2020-09-01').kubeconfigs[0].value]"
			setAKSParameters(generatedTemplate, bundle)
		} else {
			if cnabParam, ok := isCnabParam(credentialKey); options.Simplify && ok {
				credEnvVar.SecureValue = fmt.Sprintf("[variables('%s')]", cnabParam)
			} else {
				generatedTemplate.Parameters[credentialKey] = template.Parameter{
					Type:         "securestring",
					Metadata:     &metadata,
					DefaultValue: defaultValue,
				}
				credEnvVar.SecureValue = fmt.Sprintf("[parameters('%s')]", credentialKey)
			}
		}

		if err = generatedTemplate.SetDeploymentScriptEnvironmentVariable(credEnvVar); err != nil {
			return err
		}

	}

	return writeResponseData(options.Writer, generatedTemplate, options.Indent)

}

func setAKSParameters(generatedTemplate *template.Template, bundle *bundle.Bundle) {
	generatedTemplate.Parameters["aksResourceGroupName"] = template.Parameter{
		Type: "string",
		Metadata: &template.Metadata{
			Description: fmt.Sprintf("The resource group that contains the AKS Cluster to deploy bundle %s to", bundle.Name),
		},
		DefaultValue: "[resourceGroup().Name]",
	}
	generatedTemplate.Parameters["aksResourceName"] = template.Parameter{
		Type: "string",
		Metadata: &template.Metadata{
			Description: fmt.Sprintf("The name of the AKS Cluster to deploy bundle %s to", bundle.Name),
		},
	}
}

func getBundleDetails(options GenerateTemplateOptions) (*bundle.Bundle, string, error) {
	useTag := false

	if options.BundlePullOptions.Tag != "" {
		useTag = true
	}

	bundle, err := getBundle(options.BundleLoc, useTag, options.BundlePullOptions)
	if err != nil {
		return nil, "", err
	}

	bundleTag := options.BundlePullOptions.Tag
	if !useTag {
		var err error
		bundleTag, err = getBundleTag(bundle)
		if err != nil {
			return nil, "", err
		}
	}
	return bundle, bundleTag, nil
}

func isCnabParam(parameterKey string) (string, bool) {
	cnabKey := "cnab_" + parameterKey

	for _, name := range common.CNABParam {
		if name == cnabKey {
			return cnabKey, true
		}
	}
	return "", false

}

func getBundleTag(bundle *bundle.Bundle) (string, error) {
	for _, i := range bundle.InvocationImages {
		if i.ImageType == "docker" {
			ref, err := reference.ParseNamed(i.Image)
			if err != nil {
				return "", fmt.Errorf("Cannot parse invocationImage reference: %s %w", i.Image, err)
			}

			bundleTag := ref.Name() + "/bundle"

			if tagged, ok := ref.(reference.Tagged); ok {
				bundleTag += ":"
				bundleTag += tagged.Tag()
			}

			if digested, ok := ref.(reference.Digested); ok {
				bundleTag += "@"
				bundleTag += digested.Digest().String()
			}

			return bundleTag, nil
		}
	}

	return "", fmt.Errorf("Cannot get bundle name from invocationImages: %v", bundle.InvocationImages)
}

func toARMType(jsonType string, isSensitive bool) (string, error) {
	var armType string
	var err error

	switch jsonType {
	case "boolean":
		armType = "bool"
	case "integer":
		armType = "int"
	case "string":
		if isSensitive {
			armType = "securestring"
		} else {
			armType = "string"
		}
	case "object", "array":
		armType = jsonType
	default:
		err = fmt.Errorf("Unable to convert type '%s' to ARM template parameter type", jsonType)
	}

	return armType, err
}
func getBundleFromTag(bundleOptions *porter.BundlePullOptions) (*bundle.Bundle, error) {
	// TODO deal with relocationMap
	bun, _, err := pullBundle(bundleOptions.Tag, bundleOptions.InsecureRegistry)
	if err != nil {
		return nil, fmt.Errorf("Unable to pull bundle with tag: %s. %w", bundleOptions.Tag, err)
	}
	return &bun, nil
}

func getBundle(source string, useTag bool, bundleOptions *porter.BundlePullOptions) (*bundle.Bundle, error) {
	if useTag {
		return getBundleFromTag(bundleOptions)
	}

	return getBundleFromFile(source)
}

func getBundleFromFile(source string) (*bundle.Bundle, error) {
	_, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("Unable to access bundle file: %s. %w", source, err)
	}
	jsonFile, _ := os.Open(source)
	bun, err := bundle.ParseReader(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse bundle file: %s. %w", source, err)
	}
	return &bun, nil
}

func pullBundle(tag string, insecureRegistry bool) (bundle.Bundle, *relocation.ImageRelocationMap, error) {
	ref, err := reference.ParseNormalizedNamed(tag)
	if err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("Invalid bundle tag format, expected REGISTRY/name:tag %w", err)
	}

	var insecureRegistries []string
	if insecureRegistry {
		reg := reference.Domain(ref)
		insecureRegistries = append(insecureRegistries, reg)
	}

	bun, reloMap, err := remotes.Pull(context.Background(), ref, remotes.CreateResolver(config.LoadDefaultConfigFile(os.Stderr), insecureRegistries...))
	if err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("Unable to pull remote bundle %w", err)
	}

	if len(reloMap) == 0 {
		return *bun, nil, nil
	}
	return *bun, &reloMap, nil
}

func getParameterKeys(bundle bundle.Bundle) ([]string, error) {
	// Sort parameters, because Go randomizes order when iterating a map
	var parameterKeys []string
	for parameterKey := range bundle.Parameters {
		// porter-debug is added automatically so can only be modified by updating porter
		if parameterKey == "porter-debug" {
			continue
		}
		if strings.Contains(parameterKey, "-") {
			return nil, fmt.Errorf("Invalid Parameter name: %s.ARM template generation requires parameter names that can be used as environment variables", parameterKey)
		}
		parameterKeys = append(parameterKeys, parameterKey)
	}
	sort.Strings(parameterKeys)
	return parameterKeys, nil
}

func getCredentialKeys(bundle bundle.Bundle) ([]string, error) {
	// Sort credentials, because Go randomizes order when iterating a map
	var credentialKeys []string
	for credentialKey := range bundle.Credentials {

		if strings.Contains(credentialKey, "-") {
			return nil, fmt.Errorf("Invalid Credential name: %s.ARM template generation requires credential names that can be used as environment variables", credentialKey)
		}
		credentialKeys = append(credentialKeys, credentialKey)
	}
	sort.Strings(credentialKeys)
	return credentialKeys, nil
}

func getDefaultValue(definition *definition.Schema, parameter bundle.Parameter) (interface{}, bool) {
	var defaultValue interface{}
	if definition.Default != nil {
		defaultValue = definition.Default

		// If value is a string starting with square bracket, then we need to escape it
		// otherwise ARM thinks it is an expression
		if v, ok := defaultValue.(string); ok && strings.HasPrefix(v, "[") {
			v = "[" + v
			defaultValue = v
		}
	} else {
		if !parameter.Required {
			armType, err := toARMType(definition.Type.(string), false)
			if err == nil {
				switch armType {
				case "string":
					defaultValue = ""
				case "object":
					var o struct{}
					defaultValue = o
				case "array":
					defaultValue = make([]interface{}, 0)
				}
			}
		}
	}
	return defaultValue, parameter.Required
}

func writeResponseData(writer io.Writer, response interface{}, indent bool) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if indent {
		encoder.SetIndent("", "\t")
	}
	err := encoder.Encode(response)

	if err != nil {
		return fmt.Errorf("Error writing response: %w", err)
	}

	return nil
}
