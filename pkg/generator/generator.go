package generator

import (
	"encoding/json"
	"errors"
	"fmt"

	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/uidefinition"
)

// GenerateNestedDeploymentOptions is the set of options for configuring GenerateNestedDeployment
type GenerateNestedDeploymentOptions struct {
	Uri string
	common.Options
}

// GenerateNestedDeployment generates ARM deployment resource from bundle metadata
func GenerateNestedDeployment(options GenerateNestedDeploymentOptions) error {

	bundle, err := common.GetBundleFromTag(options.BundlePullOptions)
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
			if options.ReplaceKubeconfig && strings.ToLower(parameterKey) == common.KubeConfigParameterName {
				generatedDeployment.Properties.Parameters[common.AKSResourceGroupParameterName] = template.ParameterValue{
					Value: fmt.Sprintf("TODO add value for %s or delete this parameter to use default of current resource group", common.AKSResourceGroupParameterName),
				}
				generatedDeployment.Properties.Parameters[common.AKSResourceParameterName] = template.ParameterValue{
					Value: fmt.Sprintf("TODO add value for %s", common.AKSResourceParameterName),
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
		if options.ReplaceKubeconfig && strings.ToLower(credentialKey) == common.KubeConfigParameterName {
			generatedDeployment.Properties.Parameters[common.AKSResourceGroupParameterName] = template.ParameterValue{
				Value: fmt.Sprintf("TODO add value for %s or delete this parameter to use default of current resource group", common.AKSResourceGroupParameterName),
			}
			generatedDeployment.Properties.Parameters[common.AKSResourceParameterName] = template.ParameterValue{
				Value: fmt.Sprintf("TODO add value for %s", common.AKSResourceParameterName),
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

	return common.WriteOutput(options.OutputWriter, generatedDeployment, options.Indent)
}

// GenerateTemplate generates ARM template from bundle metadata
func GenerateTemplate(options common.BundleDetails) (*template.Template, *bundle.Bundle, error) {

	bundle, bundleTag, err := common.GetBundleDetails(options)
	if err != nil {
		return nil, nil, err
	}

	generatedTemplate, err := template.NewCnabArmDriverTemplate(
		bundle.Name,
		bundleTag,
		bundle.Outputs,
		options.Simplify,
		options.Timeout)

	if err != nil {
		return nil, nil, err
	}

	parameterKeys, err := getParameterKeys(*bundle)
	if err != nil {
		return nil, nil, err
	}

	for _, parameterKey := range parameterKeys {

		parameter := bundle.Parameters[parameterKey]
		definition := bundle.Definitions[parameter.Definition]
		paramEnvVar := template.EnvironmentVariable{
			Name: common.GetEnvironmentVariableNames().CnabParameterPrefix + parameterKey,
		}

		if options.ReplaceKubeconfig && strings.ToLower(parameterKey) == common.KubeConfigParameterName {
			paramEnvVar.SecureValue = fmt.Sprintf("[listClusterAdminCredential(resourceId(subscription().subscriptionId,parameters('%s'),'Microsoft.ContainerService/managedClusters',parameters('%s')), '2020-09-01').kubeconfigs[0].value]", common.AKSResourceGroupParameterName, common.AKSResourceParameterName)
			setAKSParameters(generatedTemplate, bundle)
		} else if cnabParam, ok := isCnabParam(parameterKey); options.Simplify && ok {
			paramEnvVar.Value = fmt.Sprintf("[variables('%s')]", cnabParam)
		} else if parameter.AppliesTo("install") || parameter.AppliesTo("upgrade") {
			templateParameter, isSensitive, err := genParameter(parameter, definition)
			if err != nil {
				return nil, nil, err
			}

			generatedTemplate.Parameters[parameterKey] = *templateParameter

			if isSensitive {
				paramEnvVar.SecureValue = fmt.Sprintf("[parameters('%s')]", parameterKey)
			} else {
				paramEnvVar.Value = fmt.Sprintf("[parameters('%s')]", parameterKey)
			}

		} else {
			continue
		}

		if err = generatedTemplate.SetDeploymentScriptEnvironmentVariable(paramEnvVar); err != nil {
			return nil, nil, err
		}
	}

	credentialKeys, err := getCredentialKeys(*bundle)
	if err != nil {
		return nil, nil, err
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

		if options.ReplaceKubeconfig && strings.ToLower(credentialKey) == common.KubeConfigParameterName {
			credEnvVar.SecureValue = fmt.Sprintf("[listClusterAdminCredential(resourceId(subscription().subscriptionId,parameters('%s'),'Microsoft.ContainerService/managedClusters',parameters('%s')), '2020-09-01').kubeconfigs[0].value]", common.AKSResourceGroupParameterName, common.AKSResourceParameterName)
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
			return nil, nil, err
		}

	}
	return generatedTemplate, bundle, nil
}

func genParameter(parameter bundle.Parameter, definition *definition.Schema) (*template.Parameter, bool, error) {

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
		return nil, false, err
	}

	return &template.Parameter{
		Type:          armType,
		AllowedValues: allowedValues,
		DefaultValue:  defaultValue,
		Metadata:      &metadata,
		MinValue:      minValue,
		MaxValue:      maxValue,
		MinLength:     minLength,
		MaxLength:     maxLength,
	}, isSensitive, nil
}

func GenerateCustomRP(options common.BundleDetails) (*template.Template, *bundle.Bundle, error) {
	bundle, bundleTag, err := common.GetBundleDetails(options)
	if err != nil {
		return nil, nil, err
	}

	var customTypeInfo *template.Type
	customTypeInfo, err = getCustomTypeInfo(bundle)
	if err != nil {
		return nil, nil, err
	}

	typeName := template.CustomRPTypeName

	if customTypeInfo != nil {
		typeName = customTypeInfo.Type
	}

	customRPTemplate, err := template.NewCnabCustomRPTemplate(
		bundle.Name,
		bundleTag,
		customTypeInfo)

	if err != nil {
		return nil, nil, err
	}

	customActions := getCustomActions(bundle, customTypeInfo)

	for i := range customActions {

		customProviderAction := template.CustomProviderAction{
			Name:        customActions[i],
			Endpoint:    "[concat('https://',variables('endPointDNSName'),'/{requestPath}')]",
			RoutingType: "Proxy",
		}

		if err = customRPTemplate.SetCustomRPAction(customProviderAction); err != nil {
			return nil, nil, err
		}

	}

	if options.IncludeCustomResource {

		customResourceName := fmt.Sprintf("concat('%s/',deployment().name)", typeName)
		customResourceProperties := template.CustomProviderResourceProperties{
			Credentials: make(map[string]interface{}),
			Parameters:  make(map[string]interface{}),
		}
		customResource := template.Resource{
			Type:       fmt.Sprintf("Microsoft.CustomProviders/resourceProviders/%s", typeName),
			APIVersion: template.CustomRPAPIVersion,
			Name:       fmt.Sprintf("[%s]", customResourceName),
			Location:   "[parameters('location')]",
			DependsOn:  []string{template.CustomRPName},
			Properties: customResourceProperties,
		}

		parameterKeys, err := getParameterKeys(*bundle)
		if err != nil {
			return nil, nil, err
		}

		for _, parameterKey := range parameterKeys {
			parameter := bundle.Parameters[parameterKey]
			if _, isCnabParam := isCnabParam(parameterKey); !isCnabParam && (parameter.AppliesTo("install") || parameter.AppliesTo("upgrade")) && (customTypeInfo != nil && customTypeInfo.Id != parameterKey) {

				definition := bundle.Definitions[parameter.Definition]
				templateParameter, _, err := genParameter(parameter, definition)
				if err != nil {
					return nil, nil, err
				}
				customRPTemplate.Parameters[parameterKey] = *templateParameter
				customResourceProperties.Parameters[parameterKey] = fmt.Sprintf("[parameters('%s')]", parameterKey)
			}
		}

		credentialKeys, err := getCredentialKeys(*bundle)
		if err != nil {
			return nil, nil, err
		}

		for _, credentialKey := range credentialKeys {
			credential := bundle.Credentials[credentialKey]

			var metadata template.Metadata
			var description string
			var defaultValue interface{}

			if credential.Description != "" {
				description = credential.Description
			}

			if credential.Path != "" {
				if description != "" {
					description += " "
				}
				description += "(Enter base64 encoded representation of file)"
			}

			if description != "" {
				metadata = template.Metadata{
					Description: description,
				}
			}

			if !credential.Required {
				defaultValue = ""
			}

			customRPTemplate.Parameters[credentialKey] = template.Parameter{
				Type:         "securestring",
				Metadata:     &metadata,
				DefaultValue: defaultValue,
			}
			customResourceProperties.Credentials[credentialKey] = fmt.Sprintf("[parameters('%s')]", credentialKey)
		}
		customRPTemplate.Resources = append(customRPTemplate.Resources, customResource)

		customRPTemplate.Outputs["Installation"] = template.Output{
			Type:  "string",
			Value: fmt.Sprintf("[reference(concat(resourceId('Microsoft.CustomProviders/resourceProviders','%s'),'/%s/',deployment().name)).Installation]", template.CustomRPName, typeName),
		}

		for k, v := range bundle.Outputs {
			if v.AppliesTo("install") || v.AppliesTo("upgrade") {
				sensitive, err := bundle.IsOutputSensitive(k)
				if err != nil {
					return nil, nil, fmt.Errorf("Failed to check of output %s is sensitive: %w", k, err)
				}
				if !sensitive {
					armType, err := toARMType(bundle.Definitions[v.Definition].Type.(string), false)
					if err != nil {
						return nil, nil, fmt.Errorf("Failed to get ARM type of output %s: %w", k, err)
					}
					customRPTemplate.Outputs[k] = template.Output{
						Type:  armType,
						Value: fmt.Sprintf("[reference(concat(resourceId('Microsoft.CustomProviders/resourceProviders','%s'),'/%s/',deployment().name)).%s]", template.CustomRPName, typeName, k),
					}
				}
			}
		}
	}
	return customRPTemplate, bundle, nil
}

func GenerateFiles(options common.BundleDetails) error {
	var generatedTemplate *template.Template
	var bundle *bundle.Bundle
	var err error

	if options.CustomRPTemplate {
		generatedTemplate, bundle, err = GenerateCustomRP(options)
	} else {
		generatedTemplate, bundle, err = GenerateTemplate(options)
	}
	if err != nil {
		return fmt.Errorf("Error generating template: %w", err)
	}

	err = common.WriteOutput(options.OutputWriter, generatedTemplate, options.Indent)
	if err != nil {
		return fmt.Errorf("Error writing output file: %w", err)
	}

	if options.GenerateUI {
		ui, err := uidefinition.NewCreateUIDefinition(bundle.Name, bundle.Description, generatedTemplate, options.Simplify, options.ReplaceKubeconfig, bundle.Custom, options.CustomRPTemplate, options.IncludeCustomResource)
		if err != nil {
			return fmt.Errorf("Failed to gernerate UI definition, %w", err)
		}

		if err = common.WriteOutput(options.UIWriter, ui, options.Indent); err != nil {
			return fmt.Errorf("Failed to write ui definition output, %w", err)
		}
	}
	return nil
}

func GenerateManagedAppDefinitionTemplate(options common.BundleDetails, packageUri string) (*template.Template, *bundle.Bundle, error) {

	bundle, _, err := common.GetBundleDetails(options)
	if err != nil {
		return nil, nil, err
	}
	generatedTemplate, err := template.NewCnabMAnagedAppDefinitionTemplate(
		bundle.Name,
		bundle.Description,
		packageUri,
	)

	if err != nil {
		return nil, nil, err
	}

	return generatedTemplate, bundle, nil
}

func setAKSParameters(generatedTemplate *template.Template, bundle *bundle.Bundle) {
	generatedTemplate.Parameters[common.AKSResourceGroupParameterName] = template.Parameter{
		Type: "string",
		Metadata: &template.Metadata{
			Description: fmt.Sprintf("The resource group that contains the AKS Cluster to deploy bundle %s to", bundle.Name),
		},
		DefaultValue: "[resourceGroup().Name]",
	}
	generatedTemplate.Parameters[common.AKSResourceParameterName] = template.Parameter{
		Type: "string",
		Metadata: &template.Metadata{
			Description: fmt.Sprintf("The name of the AKS Cluster to deploy bundle %s to", bundle.Name),
		},
	}
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

func getCustomActions(bundle *bundle.Bundle, customTypeInfo *template.Type) []string {
	var actions []string
	for name := range bundle.Actions {
		if isCustomAction(name) {
			typedName := getCustomActionTypedName(customTypeInfo, name)
			if typedName != "" {
				actions = append(actions, typedName)
			}
		}
	}
	return actions
}

func isCustomAction(name string) bool {
	for i := range common.BuiltInActions {
		if strings.EqualFold(common.BuiltInActions[i], name) {
			return false
		}
	}
	return true
}

func getCustomActionTypedName(customTypeInfo *template.Type, name string) string {

	if customTypeInfo == nil {
		return name
	}
	for action, actionName := range customTypeInfo.Actions {
		if strings.EqualFold(actionName, name) {
			return fmt.Sprintf("%s/%s", customTypeInfo.Type, action)
		}
	}
	for childTypeName, childType := range customTypeInfo.ChildTypes {
		for action, actionName := range childType.Actions {
			if strings.EqualFold(actionName, name) {
				return fmt.Sprintf("%s/%s/%s", customTypeInfo.Type, childTypeName, action)
			}
		}
	}
	return ""
}

func getCustomTypeInfo(bundle *bundle.Bundle) (*template.Type, error) {
	var typeInfo interface{}
	if typeInfo = bundle.Custom["com.azure.arm"]; typeInfo == nil {
		return nil, nil
	}
	var customType template.Type
	jsonData, err := json.Marshal(bundle.Custom["com.azure.arm"])
	if err != nil {
		return nil, fmt.Errorf("Unable to serialise Custom Type settings to JSON %w", err)
	}
	err = json.Unmarshal(jsonData, &customType)
	if err != nil {
		return nil, fmt.Errorf("Unable to de-serialise Custom Type settings from JSON %w", err)
	}
	if customType.Type == "" {
		return nil, errors.New("Custom Type specified with no type property")
	}
	if customType.Id == "" {
		return nil, fmt.Errorf("Id not specified for custom type %s", customType.Type)
	}
	if _, ok := bundle.Parameters[customType.Id]; !ok {
		return nil, fmt.Errorf("Bundle Parameter %s specified as Id for Type %s does not exist", customType.Id, customType.Type)
	}
	for childTypeName, childType := range customType.ChildTypes {
		actions := []string{"CreateUpdateAction", "DeleteAction", "GetAction", "ListAction"}
		for _, childAction := range actions {
			fieldValue := reflect.ValueOf(&childType).Elem().FieldByName(childAction).String()
			if fieldValue == "" {
				return nil, fmt.Errorf("Action %s for Operation %s for Child Type %s is not set", fieldValue, childAction, childTypeName)
			} else {
				if _, ok := bundle.Actions[fieldValue]; !ok {
					return nil, fmt.Errorf("Action %s for Operation %s for Child Type %s does not exist", fieldValue, childAction, childTypeName)
				} else {
					if customAction := isCustomAction(fieldValue); !customAction {
						return nil, fmt.Errorf("Action %s for for Operation %s Child Type %s is not a custom action", fieldValue, childAction, childTypeName)
					}
				}
			}
		}
		for actionName, childTypeAction := range childType.Actions {
			if _, ok := bundle.Actions[childTypeAction]; !ok {
				return nil, fmt.Errorf("Custom action %s for action name %s for Child Type %s does not exist", childTypeAction, actionName, childTypeName)
			} else {
				if customAction := isCustomAction(childTypeAction); !customAction {
					return nil, fmt.Errorf("Custom action %s for action name %s for Child Type %s is not a custom action", childTypeAction, actionName, childTypeName)
				}
			}
		}
	}

	return &customType, nil
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

func WriteOutput(writer io.Writer, data interface{}, indent bool) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if indent {
		encoder.SetIndent("", "\t")
	}
	err := encoder.Encode(data)

	if err != nil {
		return fmt.Errorf("Error writing response: %w", err)
	}

	return nil
}
