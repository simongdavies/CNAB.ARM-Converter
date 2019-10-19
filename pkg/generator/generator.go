package generator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution/reference"
	"github.com/endjin/CNAB.ARM-Converter/pkg/template"
)

const (
	bundlecontainerregistry = "cnabquickstartstest.azurecr.io/"
)

// GenerateTemplate generates ARM template from bundle metadata
func GenerateTemplate(bundleloc string, outputfile string, overwrite bool, indent bool) error {

	// TODO support http uri and registry based bundle

	bundle, err := loadBundle(bundleloc)

	if err != nil {
		return err
	}

	if err = checkOutputFile(outputfile, overwrite); err != nil {
		return err
	}

	generatedTemplate := template.NewCnabArmDriverTemplate()

	bundleName := bundle.Name
	bundleNameEnvironmentVariable := template.EnvironmentVariable{
		Name:  template.CnabBundleNameEnvVar,
		Value: bundleName,
	}

	if err = generatedTemplate.SetContainerEnvironmentVariable(bundleNameEnvironmentVariable); err != nil {
		return err
	}

	bundleTag, err := getBundleTag(bundle)
	if err != nil {
		return err
	}
	bundleTagEnvironmentVariable := template.EnvironmentVariable{
		Name:  template.CnabBundleTagEnvVar,
		Value: bundleTag,
	}

	if err = generatedTemplate.SetContainerEnvironmentVariable(bundleTagEnvironmentVariable); err != nil {
		return err
	}

	// Set the default installation name to be the bundle name
	installationName := bundle.Name
	generatedTemplate.Parameters["cnab_installation_name"] = template.Parameter{
		Type:         "string",
		DefaultValue: installationName,
		Metadata: &template.Metadata{
			Description: "The name of the application instance.",
		},
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

		// Location parameter is added to template definition automatically as ACI uses it
		if parameterKey != "location" {

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

			armType, err := toARMType(definition.Type.(string))
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
		}

		environmentVariable := template.EnvironmentVariable{
			Name:  "CNAB_PARAM_" + parameterKey,
			Value: fmt.Sprintf("[parameters('%s')]", parameterKey),
		}

		if err = generatedTemplate.SetContainerEnvironmentVariable(environmentVariable); err != nil {
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

		var environmentVariable template.EnvironmentVariable
		var metadata template.Metadata
		var defaultValue interface{}

		if credential.Description != "" {
			metadata = template.Metadata{
				Description: credential.Description,
			}
		}

		if !credential.Required {
			defaultValue = ""
		}

		generatedTemplate.Parameters[credentialKey] = template.Parameter{
			Type:         "securestring",
			Metadata:     &metadata,
			DefaultValue: defaultValue,
		}
		environmentVariable = template.EnvironmentVariable{
			Name:        "CNAB_CRED_" + credentialKey,
			SecureValue: fmt.Sprintf("[parameters('%s')]", credentialKey),
		}

		if err = generatedTemplate.SetContainerEnvironmentVariable(environmentVariable); err != nil {
			return err
		}
	}

	var data []byte
	if indent {
		data, _ = json.MarshalIndent(generatedTemplate, "", "\t")
	} else {
		data, _ = json.Marshal(generatedTemplate)
	}

	if err := ioutil.WriteFile(outputfile, data, 0644); err != nil {
		return err
	}

	return nil
}

func getBundleTag(bundle *bundle.Bundle) (string, error) {
	for _, i := range bundle.InvocationImages {
		if i.ImageType == "docker" {
			ref, err := reference.ParseNamed(i.Image)
			if err != nil {
				return "", fmt.Errorf("Cannot parse invocationImage reference: %s", i.Image)
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

func toARMType(jsonType string) (string, error) {
	var armType string
	var err error

	switch jsonType {
	case "boolean":
		armType = "bool"
	case "integer":
		armType = "int"
	case "object", "array", "string":
		armType = jsonType
	default:
		err = fmt.Errorf("Unable to convert type '%s' to ARM template parameter type", jsonType)
	}

	return armType, err
}

func loadBundle(source string) (*bundle.Bundle, error) {
	_, err := os.Stat(source)
	if err == nil {
		jsonFile, _ := os.Open(source)
		bundle, err := bundle.ParseReader(jsonFile)
		return &bundle, err
	}
	return nil, err
}

func checkOutputFile(dest string, overwrite bool) error {
	if _, err := os.Stat(dest); err == nil {
		if !overwrite {
			return fmt.Errorf("File %s exists and overwrite not specified", dest)
		}
	} else {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
