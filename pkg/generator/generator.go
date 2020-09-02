package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"get.porter.sh/porter/pkg/porter"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-to-oci/relocation"
	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
)

// GenerateTemplateOptions is the set of options for configuring GenerateTemplate
type GenerateTemplateOptions struct {
	BundleLoc         string
	OutputFile        string
	Overwrite         bool
	Indent            bool
	Version           string
	Simplify          bool
	BundlePullOptions *porter.BundlePullOptions
}

// GenerateTemplate generates ARM template from bundle metadata
func GenerateTemplate(options GenerateTemplateOptions) error {

	useTag := false

	if options.BundlePullOptions.Tag != "" {
		useTag = true
	}

	// TODO support http uri and registry based bundle
	bundle, err := loadBundle(options.BundleLoc, useTag, options.BundlePullOptions)

	if err != nil {
		return err
	}

	if err = checkOutputFile(options.OutputFile, options.Overwrite); err != nil {
		return err
	}

	bundleName := bundle.Name
	bundleTag := options.BundlePullOptions.Tag
	if !useTag {
		var err error
		bundleTag, err = getBundleTag(bundle)
		if err != nil {
			return err
		}
	}
	bundleActions := make([]string, 0, len(bundle.Actions)+3)

	bundleActions = append(bundleActions, template.DefaultCNABActions...)
	for action := range bundle.Actions {
		bundleActions = append(bundleActions, action)
	}

	generatedTemplate, err := template.NewCnabArmDriverTemplate(
		bundleName,
		bundleTag,
		bundleActions,
		options.Simplify)

	if err != nil {
		return err
	}

	// Sort parameters, because Go randomizes order when iterating a map
	var parameterKeys []string
	for parameterKey := range bundle.Parameters {
		parameterKeys = append(parameterKeys, parameterKey)
	}
	sort.Strings(parameterKeys)

	for _, parameterKey := range parameterKeys {

		parameter := bundle.Parameters[parameterKey]
		definition := bundle.Definitions[parameter.Definition]

		// Parameter names cannot contain - as they are converted into environment variables set on ACI container

		// porter-debug is added automatically so can only be modified by updating porter
		if parameterKey == "porter-debug" {
			continue
		}

		if strings.Contains(parameterKey, "-") {
			return fmt.Errorf("Invalid Parameter name: %s.ARM template generation requires parameter names that can be used as environment variables", parameterKey)
		}

		var paramEnvVar template.EnvironmentVariable

		if cnabParam, ok := isCnabParam(parameterKey, *generatedTemplate); options.Simplify && ok {
			paramEnvVar = template.EnvironmentVariable{
				Name:  common.GetEnvironmentVariableNames().CnabParameterPrefix + parameterKey,
				Value: fmt.Sprintf("[variables('%s')]", cnabParam),
			}
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
					defaultValue = ""
				}
			}

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

			paramEnvVar = template.EnvironmentVariable{
				Name:  common.GetEnvironmentVariableNames().CnabParameterPrefix + parameterKey,
				Value: fmt.Sprintf("[parameters('%s')]", parameterKey),
			}
		}

		if err = generatedTemplate.SetDeploymentScriptEnvironmentVariable(paramEnvVar); err != nil {
			return err
		}
	}

	// Sort credentials, because Go randomizes order when iterating a map
	var credentialKeys []string
	for credentialKey := range bundle.Credentials {
		credentialKeys = append(credentialKeys, credentialKey)
	}
	sort.Strings(credentialKeys)

	for _, credentialKey := range credentialKeys {

		credential := bundle.Credentials[credentialKey]

		if strings.Contains(credentialKey, "-") {
			return fmt.Errorf("Invalid Credential name: %s.ARM template generation requires credential names that can be used as environment variables", credentialKey)
		}

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

		var credEnvVar template.EnvironmentVariable

		if cnabParam, ok := isCnabParam(credentialKey, *generatedTemplate); options.Simplify && ok {
			credEnvVar = template.EnvironmentVariable{
				Name:        envVarName,
				SecureValue: fmt.Sprintf("[variables('%s')]", cnabParam),
			}
		} else {
			generatedTemplate.Parameters[credentialKey] = template.Parameter{
				Type:         "securestring",
				Metadata:     &metadata,
				DefaultValue: defaultValue,
			}

			credEnvVar = template.EnvironmentVariable{
				Name:        envVarName,
				SecureValue: fmt.Sprintf("[parameters('%s')]", credentialKey),
			}
		}

		if err = generatedTemplate.SetDeploymentScriptEnvironmentVariable(credEnvVar); err != nil {
			return err
		}
	}

	var data []byte
	if options.Indent {
		data, _ = json.MarshalIndent(generatedTemplate, "", "\t")
	} else {
		data, _ = json.Marshal(generatedTemplate)
	}

	if err := ioutil.WriteFile(options.OutputFile, data, 0644); err != nil {
		return err
	}

	return nil
}

func isCnabParam(parameterKey string, template template.Template) (string, bool) {
	cnabKey := "cnab_" + parameterKey
	if _, ok := template.Variables[cnabKey]; ok {
		return cnabKey, true
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

func loadBundle(source string, useTag bool, bundleOptions *porter.BundlePullOptions) (*bundle.Bundle, error) {
	// TODO deal with relocationMap
	if useTag {
		bun, _, err := pullBundle(bundleOptions.Tag, bundleOptions.InsecureRegistry)
		if err != nil {
			return nil, fmt.Errorf("Unable to pull bundle with tag: %s. %w", bundleOptions.Tag, err)
		}
		return &bun, nil
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

func checkOutputFile(dest string, overwrite bool) error {
	if _, err := os.Stat(dest); err == nil {
		if !overwrite {
			return fmt.Errorf("File %s exists and overwrite not specified", dest)
		}
	} else {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to access output file: %s. %w", dest, err)
		}
	}

	return nil
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
