package uidefinition

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
)

func NewCreateUIDefinition(bundleName string, bundleDescription string, generatedTemplate *template.Template, simplyfy bool, useAKS bool, custom map[string]interface{}) (*CreateUIDefinition, error) {
	UIDef := CreateUIDefinition{
		Schema:  "https://schema.management.azure.com/schemas/0.1.2-preview/CreateUIDefinition.MultiVm.json#",
		Handler: "Microsoft.Azure.CreateUIDef",
		Version: "0.1.2-preview",
		Parameters: Parameters{
			Config: Config{
				IsWizard: false,
				Basics: BasicsConfig{
					Description: bundleDescription,
					ResourceGroup: ResourceGroup{
						Constraints: ResourceConstraints{
							Validations: []ResourceValidation{
								{
									Permission: "Microsoft.ContainerInstance/containerGroups/write",
									Message:    "Permission to create Container Instance is needed in resource group ",
								},
								{
									Permission: "Microsoft.ManagedIdentity/userAssignedIdentities/write",
									Message:    "Permission to create User Assigned Identity is needed in resource group ",
								},
								{
									Permission: "Microsoft.Authorization/roleAssignments/write",
									Message:    "Permission to create Role Assignemnts is needed in resource group ",
								},
								{
									Permission: "Microsoft.Storage/storageAccounts/write",
									Message:    "Permission to create Storage Accounts is needed in resource group ",
								},
								{
									Permission: "Microsoft.Storage/storageAccounts/blobServices/containers/write",
									Message:    "Permission to create Storage Account Containers is needed in resource group ",
								},
								{
									Permission: "Microsoft.Storage/storageAccounts/fileServices/shares/write",
									Message:    "Permission to create Storage Account File Shares is needed in resource group ",
								},
								{
									Permission: "Microsoft.Resources/deploymentScripts/write",
									Message:    "Permission to create Deployment Scripts is needed in resource group ",
								},
							},
						},
						AllowExisting: true,
					},
					Location: Location{
						Label:   "CNAB Action Location",
						Tooltip: "This is the location where the deployment to run the CNAB action will run",
						ResourceTypes: []string{
							"Microsoft.ContainerInstance/containerGroups",
							"Microsoft.ManagedIdentity/userAssignedIdentities",
							"Microsoft.Authorization/roleAssignments",
							"Microsoft.Storage/storageAccounts",
							"Microsoft.Storage/storageAccounts/blobServices/containers",
							"Microsoft.Storage/storageAccounts/fileServices/shares",
							"Microsoft.Resources/deploymentScripts",
						},
						Visible: true,
					},
				},
			},
			Basics: []Element{},
			Steps:  make([]Step, 0),
		},
	}
	outputs := map[string]string{}

	elementsMap := map[string][]Element{}
	elementsMap["basics"] = []Element{}
	elementsMap["Additional"] = []Element{}

	if _, aks := generatedTemplate.Parameters[common.AKSResourceParameterName]; aks && useAKS {
		elementsMap["basics"] = append(elementsMap["basics"], Element{
			Name:         "aksSelector",
			Type:         "Microsoft.Solutions.ResourceSelector",
			Label:        "Select AKS Cluster",
			ResourceType: "Microsoft.ContainerService/managedClusters",
			Visible:      true,
			Tooltip:      fmt.Sprintf("Select the AKS Cluster to deploy %s to", bundleName),
			Options: ResourceSelectorOptions{
				Filter: ResourceSelectorFilter{
					Subscription: OnBasics.String(),
					Location:     All.String(),
				},
			},
		})

		outputs[common.AKSResourceGroupParameterName] = "[last(take(split(steps('basics').aksSelector.id,'/'),5))]"
		outputs[common.AKSResourceParameterName] = "[steps('basics').aksSelector.name]"
	}

	settings := make(CustomSettings, 0)
	if customSettings := custom["com.azure.creatuidef"]; customSettings != nil {
		jsonData, err := json.Marshal(custom["com.azure.creatuidef"])
		if err != nil {
			return nil, fmt.Errorf("Unable to serialise Custom UI settings to JSON %w", err)
		}
		err = json.Unmarshal(jsonData, &settings)
		if err != nil {
			return nil, fmt.Errorf("Unable to de-serialise Custom UI settings from JSON %w", err)
		}

		settings.SortByDisplayOrder()

		allSettings := []CustomSettings{
			// OrderedElements
			make(CustomSettings, 0),
			// UnorderedElements
			make(CustomSettings, 0),
		}

		for _, val := range settings {
			// Only process setting if the parameter is in the template and if hide is not set
			if _, ok := generatedTemplate.Parameters[val.Name]; !ok || (val.Hide && !isRequired(generatedTemplate.Parameters[val.Name].DefaultValue)) {
				continue
			}
			// two arrays first contains display ordered settings, second contains settings with no order
			if val.DisplayOrder > 0 {
				allSettings[0] = append(allSettings[0], val)
			} else {
				allSettings[1] = append(allSettings[1], val)
			}
		}

		for _, filteredSettings := range allSettings {

			for _, val := range filteredSettings {
				step := "basics"
				if len(val.Bladename) > 0 {
					step = val.Bladename
					if _, ok := elementsMap[step]; !ok {
						elementsMap[step] = make([]Element, 0)
					}
				}
				tooltip := trimLabel(generatedTemplate.Parameters[val.Name].Metadata.Description)
				if len(val.Tooltip) > 0 {
					tooltip = val.Tooltip
				}
				switch {
				case strings.Contains(strings.ToLower(val.Name), "user") || strings.ToLower(val.UIType) == "microsoft.compute.usernametextbox":
					elementsMap[step] = append(elementsMap[step], createUserNameTextBox(val.Name, val.DisplayName, tooltip, generatedTemplate.Parameters[val.Name].DefaultValue, val.ValidationRegex, val.ValidationMessage))

				case strings.Contains(strings.ToLower(val.Name), "password") || strings.ToLower(val.UIType) == "microsoft.common.passwordbox":
					elementsMap[step] = append(elementsMap[step], createPasswordBox(val.Name, val.DisplayName, tooltip, generatedTemplate.Parameters[val.Name].DefaultValue, val.ValidationRegex, val.ValidationMessage))

				case strings.ToLower(val.UIType) == "microsoft.common.textbox":
					fallthrough

				default:
					elementsMap[step] = append(elementsMap[step], createTextBox(val.Name, val.DisplayName, tooltip, generatedTemplate.Parameters[val.Name].DefaultValue, isRequired(generatedTemplate.Parameters[val.Name].DefaultValue), "", ""))

				}
				outputs[val.Name] = fmt.Sprintf("[steps('%s').%s]", step, val.Name)
			}
		}
	}

	settings.SortByName()

	for name, val := range generatedTemplate.Parameters {
		// Skip any parameters with custom ui
		if hasCustomSettings(settings, name) || name == common.AKSResourceGroupParameterName || name == common.AKSResourceParameterName {
			continue
		}

		step := "basics"
		switch {
		case strings.Contains(strings.ToLower(name), "user"):
			elementsMap["basics"] = append(elementsMap["basics"], createUserNameTextBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, val.DefaultValue, "", ""))

		case strings.Contains(strings.ToLower(name), "password"):
			elementsMap["basics"] = append(elementsMap["basics"], createPasswordBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, val.DefaultValue, "", ""))

		default:
			element := createTextBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, getDefaultValue(val.DefaultValue), isRequired(val.DefaultValue), "", "")
			if !isRequired(val.DefaultValue) {
				step = "Additional"
			}
			elementsMap[step] = append(elementsMap[step], element)
		}
		outputs[name] = fmt.Sprintf("[steps('%s').%s]", step, name)

	}

	for k, v := range elementsMap {
		if k == "basics" {
			UIDef.Parameters.Basics = v
		} else {
			if len(v) > 0 {
				step := Step{
					Name:     k,
					Label:    fmt.Sprintf("%s Parameters for %s", k, trimLabel(bundleName)),
					Elements: v,
				}
				UIDef.Parameters.Steps = append(UIDef.Parameters.Steps, step)
			}
		}
	}

	UIDef.Parameters.Outputs = outputs
	return &UIDef, nil
}

func hasCustomSettings(settings CustomSettings, name string) bool {
	index := sort.Search(len(settings), func(i int) bool { return settings[i].Name >= name })
	result := index < len(settings) && settings[index].Name == name
	return result
}

func trimLabel(label string) string {
	return strings.TrimSpace(strings.TrimSuffix(label, "(Required)"))
}

func isRequired(defaultValue interface{}) bool {
	return reflect.TypeOf(defaultValue) == nil
}

func getDefaultValue(defaultValue interface{}) interface{} {
	val := defaultValue
	if v, ok := defaultValue.(string); ok {
		if strings.HasPrefix(v, "[") && !strings.HasPrefix(v, "[[") {
			v = "[" + v
		}
		val = v
	}
	return val
}

func createUserNameTextBox(name string, label string, tooltip string, defaultValue interface{}, regex string, validationMessage string) Element {
	element := Element{
		Name:         name,
		Type:         "Microsoft.Compute.UserNameTextBox",
		Label:        label,
		Tooltip:      tooltip,
		Visible:      true,
		DefaultValue: defaultValue,
		Constraints: UserPasswordConstraints{
			Required:          isRequired(defaultValue),
			Regex:             regex,
			ValidationMessage: validationMessage,
		},
		OsPlatform: Linux.String(),
	}

	return element
}

func createPasswordBox(name string, label string, tooltip string, defaultValue interface{}, regex string, validationMessage string) Element {
	element := Element{
		Name: name,
		Type: "Microsoft.Common.PasswordBox",
		Label: PasswordLabel{
			Password:        label,
			ConfirmPassword: fmt.Sprintf("Confirm %s", label),
		},
		Tooltip: tooltip,
		Visible: true,
		Constraints: UserPasswordConstraints{
			Required:          isRequired(defaultValue),
			Regex:             regex,
			ValidationMessage: validationMessage,
		},
		Options: PasswordOptions{
			HideConfirmation: false,
		},
	}

	return element
}

func createTextBox(name string, label string, tooltip string, defaultValue interface{}, required bool, regex string, validationMessage string) Element {

	element := Element{
		Name:        name,
		Type:        "Microsoft.Common.TextBox",
		Label:       label,
		Tooltip:     tooltip,
		Visible:     true,
		Placeholder: fmt.Sprintf("Provide value for %s", label),
		Constraints: TextBoxConstraints{
			Required: required,
			Validations: []TextboxValidations{
				{
					Regex:   regex,
					Message: validationMessage,
				},
			},
		},
	}

	if !isRequired(defaultValue) {
		element.DefaultValue = defaultValue
	}

	return element
}
