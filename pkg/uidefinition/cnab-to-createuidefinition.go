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

func NewCreateUIDefinition(bundleName string, bundleDescription string, generatedTemplate *template.Template, simplyfy bool, useAKS bool, custom map[string]interface{}, customRPUI bool, includeResource bool, isARCResource bool, isDogfood bool) (*CreateUIDefinition, error) {

	if isARCResource {
		return NewArcCreateUIDefinition(bundleName, bundleDescription, generatedTemplate, simplyfy, custom, customRPUI, includeResource, isDogfood)
	}

	locationLabel := "CNAB Action Location"
	locationToolTip := "This is the location where the deployment to run the CNAB action will run"
	if customRPUI {
		locationLabel = "Application Location"
		locationToolTip = "This is the location for the application and all its resources"
	}

	UIDef := CreateUIDefinition{
		Schema:  "https://schema.management.azure.com/schemas/0.1.2-preview/CreateUIDefinition.MultiVm.json#",
		Handler: "Microsoft.Azure.CreateUIDef",
		Version: "0.1.2-preview",
		Parameters: Parameters{
			Config: Config{
				IsWizard: true,
				Basics: BasicsConfig{
					Description: bundleDescription,
					ResourceGroup: &ResourceGroup{
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
					Location: &Location{
						Label:   locationLabel,
						Tooltip: locationToolTip,
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

	if hasAKSParams(*generatedTemplate) && useAKS && !customRPUI {
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

	if _, hasKubeConfig := generatedTemplate.Parameters[common.KubeConfigParameterName]; hasKubeConfig && customRPUI && includeResource {
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

		elementsMap["basics"] = append(elementsMap["basics"], Element{
			Name: "aksKubeConfig",
			Type: "Microsoft.Solutions.ArmApiControl",
			Request: &ArmAPIRequest{
				Method: "POST",
				Path:   "[replace(concat(string(steps('basics').aksSelector.id),'/listClusterAdminCredential?api-version=2019-04-01'),'\"','')]",
			},
		})

		outputs[common.KubeConfigParameterName] = "[first(steps('basics').aksKubeConfig.kubeconfigs).value]"
	}

	return processParameters(generatedTemplate, custom, &UIDef, outputs, elementsMap, customRPUI)
}

func hasAKSParams(template template.Template) bool {
	_, aksResource := template.Parameters[common.AKSResourceParameterName]
	_, aksResourceGroup := template.Parameters[common.AKSResourceGroupParameterName]
	return aksResource && aksResourceGroup
}

func hasARCParams(template template.Template) bool {
	_, customResource := template.Parameters[common.CustomLocationResourceParameterName]
	_, customResourceGroup := template.Parameters[common.CustomLocationRGParameterName]
	return customResource && customResourceGroup
}

func NewArcCreateUIDefinition(bundleName string, bundleDescription string, generatedTemplate *template.Template, simplyfy bool, custom map[string]interface{}, customRPUI bool, includeResource bool, isDogfood bool) (*CreateUIDefinition, error) {

	locationLabel := "CNAB RP Location"
	locationToolTip := "This is the location where the CNAB RP will be located"
	if customRPUI {
		locationLabel = "Application Location"
		locationToolTip = "This is the location for the application and all its resources"
	}

	//TODO: set permission requests correctly for ARC template
	provider := "Microsoft.Contoso"

	if isDogfood {
		provider = "Microsoft.CNAB"
	}

	UIDef := CreateUIDefinition{
		Schema:  "https://schema.management.azure.com/schemas/0.1.2-preview/CreateUIDefinition.MultiVm.json#",
		Handler: "Microsoft.Azure.CreateUIDef",
		Version: "0.1.2-preview",
		Parameters: Parameters{
			Config: Config{
				IsWizard: true,
				Basics: BasicsConfig{
					Description: bundleDescription,
					Subscription: &Subscription{
						ResourceProviders: []string{
							provider,
						},
					},
					ResourceGroup: &ResourceGroup{
						Constraints: ResourceConstraints{
							Validations: []ResourceValidation{
								{
									Permission: fmt.Sprintf("%s/installations/write", provider),
									Message:    "Permission to create CNAB RP is needed in resource group ",
								},
							},
						},
						AllowExisting: true,
					},
					Location: &Location{
						Label:   locationLabel,
						Tooltip: locationToolTip,
						ResourceTypes: []string{
							fmt.Sprintf("%s/installations", provider),
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

	//TODO: Handle CustomRP and CustomLocation

	if hasARCParams(*generatedTemplate) && !customRPUI {
		elementsMap["basics"] = append(elementsMap["basics"], Element{
			Name:         "customLocationSelector",
			Type:         "Microsoft.Solutions.ResourceSelector",
			Label:        "Select Custom Location",
			ResourceType: "Microsoft.Extendedlocation/Customlocations",
			Visible:      true,
			Tooltip:      fmt.Sprintf("Select the Custom Location to deploy %s to", bundleName),
			Options: ResourceSelectorOptions{
				Filter: ResourceSelectorFilter{
					Subscription: OnBasics.String(),
					Location:     All.String(),
				},
			},
		})

		outputs[common.CustomLocationRGParameterName] = "[last(take(split(steps('basics').customLocationSelector.id,'/'),5))]"
		outputs[common.CustomLocationResourceParameterName] = "[steps('basics').customLocationSelector.name]"
	}

	// ARC in Dogfood requires location to be West US in prod requires eastus2canary

	if isDogfood {
		UIDef.Parameters.Config.Basics.Location.AllowedValues = []string{"westus"}
	} else {
		UIDef.Parameters.Config.Basics.Location.AllowedValues = []string{"eastus2euap"}
	}

	return processParameters(generatedTemplate, custom, &UIDef, outputs, elementsMap, customRPUI)
}

func processParameters(generatedTemplate *template.Template, custom map[string]interface{}, UIDef *CreateUIDefinition, outputs map[string]string, elementsMap map[string][]Element, customRPUI bool) (*CreateUIDefinition, error) {

	var settings CustomSettings
	if customSettings := custom["com.azure.creatuidef"]; customSettings != nil {
		jsonData, err := json.Marshal(custom["com.azure.creatuidef"])
		if err != nil {
			return nil, fmt.Errorf("Unable to serialise Custom UI settings to JSON %w", err)
		}
		err = json.Unmarshal(jsonData, &settings)
		if err != nil {
			return nil, fmt.Errorf("Unable to de-serialise Custom UI settings from JSON %w", err)
		}

		settings.DisplayElements.SortByDisplayOrder()

		allSettings := make([]DisplayElements, 2)

		for _, val := range settings.Elements {
			// Only process setting if the parameter is in the template and if hide is not set and if customRPUI then skip kubeconfig as UI has been generated to select this.
			if _, ok := generatedTemplate.Parameters[val.Name]; !ok || (val.Hide && !isRequired(generatedTemplate.Parameters[val.Name].DefaultValue)) || strings.ToLower(val.Name) == common.KubeConfigParameterName {
				continue
			}
			// two arrays first contains display ordered settings, second contains settings with no order
			if val.DisplayOrder > 0 {
				allSettings[0].Elements = append(allSettings[0].Elements, val)
			} else {
				allSettings[1].Elements = append(allSettings[1].Elements, val)
			}
		}

		for _, filteredSettings := range allSettings {

			for _, val := range filteredSettings.Elements {
				step := "basics"
				if len(val.Bladename) > 0 {
					step = val.Bladename
					if _, ok := elementsMap[step]; !ok {
						if _, ok := settings.Blades[val.Bladename]; ok {
							elementsMap[step] = make([]Element, 0)
						} else {
							return nil, fmt.Errorf("Bladename %s specified for element %s does not exist", val.Bladename, val.Name)
						}
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

				//TODO: add support for checkbox

				case strings.ToLower(val.UIType) == "microsoft.common.textbox":
					fallthrough

				default:
					elementsMap[step] = append(elementsMap[step], createTextBox(val.Name, val.DisplayName, tooltip, generatedTemplate.Parameters[val.Name].DefaultValue, isRequired(generatedTemplate.Parameters[val.Name].DefaultValue), "", ""))

				}
				outputs[val.Name] = fmt.Sprintf("[steps('%s').%s]", step, val.Name)
			}
		}
	}

	if _, ok := generatedTemplate.Parameters[common.LocationParameterName]; ok {
		outputs[common.LocationParameterName] = "[location()]"
	}

	settings.SortByName()

	for name, val := range generatedTemplate.Parameters {
		// Skip any parameters with custom ui , location or kubeconfig when processing CustomRP UI
		if shouldSkipParameter(settings, name, customRPUI) {
			continue
		}

		step := "basics"
		switch {
		case strings.Contains(strings.ToLower(name), "user"):
			elementsMap["basics"] = append(elementsMap["basics"], createUserNameTextBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, val.DefaultValue, "", ""))

		case strings.Contains(strings.ToLower(name), "password"):
			elementsMap["basics"] = append(elementsMap["basics"], createPasswordBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, val.DefaultValue, "", ""))

		case val.Type == "bool":
			defaultValue, ok := val.DefaultValue.(bool)
			if !ok {
				defaultValue = false
			}
			element := createCheckBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, defaultValue, "", "")
			if !isRequired(val.DefaultValue) {
				step = "Additional"
			}
			elementsMap[step] = append(elementsMap[step], element)

		default:
			element := createTextBox(name, trimLabel(val.Metadata.Description), val.Metadata.Description, getDefaultValue(val.DefaultValue), isRequired(val.DefaultValue), "", "")
			if !isRequired(val.DefaultValue) {
				step = "Additional"
			}
			elementsMap[step] = append(elementsMap[step], element)
		}
		outputs[name] = fmt.Sprintf("[steps('%s').%s]", step, name)

	}

	UIDef.Parameters.Basics = elementsMap["basics"]
	type bladeDetails struct {
		Name  string
		Order int
	}
	bladeDisplayOrder := make([]bladeDetails, 0)

	for k, v := range settings.Blades {
		if v.Label == "" {
			return nil, fmt.Errorf("No Label provided for blade %s", k)
		}
		b := bladeDetails{
			Name:  k,
			Order: v.DisplayOrder,
		}
		bladeDisplayOrder = append(bladeDisplayOrder, b)

	}

	sort.SliceStable(bladeDisplayOrder, func(i, j int) bool {
		return bladeDisplayOrder[i].Order < bladeDisplayOrder[j].Order
	})

	for _, v := range bladeDisplayOrder {

		if len(elementsMap[v.Name]) > 0 {
			step := Step{
				Name:     v.Name,
				Label:    settings.Blades[v.Name].Label,
				Elements: elementsMap[v.Name],
			}
			UIDef.Parameters.Steps = append(UIDef.Parameters.Steps, step)
		}
	}

	if len(elementsMap["Additional"]) > 0 {
		step := Step{
			Name:     "Additional",
			Label:    "Additional Parameters",
			Elements: elementsMap["Additional"],
		}
		UIDef.Parameters.Steps = append(UIDef.Parameters.Steps, step)
	}

	UIDef.Parameters.Outputs = outputs
	return UIDef, nil
}

func shouldSkipParameter(settings CustomSettings, name string, customRPUI bool) bool {
	return hasCustomSettings(settings, name) ||
		name == common.CustomLocationRGParameterName ||
		name == common.CustomLocationResourceParameterName ||
		name == common.AKSResourceGroupParameterName ||
		name == common.AKSResourceParameterName ||
		name == common.LocationParameterName ||
		(name == common.KubeConfigParameterName && customRPUI)
}

func hasCustomSettings(settings CustomSettings, name string) bool {
	index := sort.Search(len(settings.Elements), func(i int) bool { return settings.Elements[i].Name >= name })
	result := index < len(settings.Elements) && settings.Elements[index].Name == name
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

func createCheckBox(name string, label string, tooltip string, defaultValue bool, regex string, validationMessage string) Element {
	element := Element{
		Name:         name,
		Type:         "Microsoft.Common.CheckBox",
		Label:        label,
		Tooltip:      tooltip,
		Visible:      true,
		DefaultValue: defaultValue,
		Constraints: CheckBoxConstraints{
			Required:          isRequired(defaultValue),
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

	if label == "" {
		label = strings.ToTitle(name)
	}
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
