package uidefinition

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/template"
)

func NewCreateUIDefinition(bundleName string, bundleDescription string, generatedTemplate *template.Template, simplyfy bool, useAKS bool) *CreateUIDefinition {
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
	elements := []Element{}
	optionalElements := []Element{}
	if _, aks := generatedTemplate.Parameters[common.AKSResourceParameterName]; aks && useAKS {
		elements = append(elements, Element{
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

	for name, val := range generatedTemplate.Parameters {
		if name == common.AKSResourceGroupParameterName || name == common.AKSResourceParameterName {
			continue
		}

		switch {
		case strings.Contains(strings.ToLower(name), "user"):
			elements = append(elements, Element{
				Name:         name,
				Type:         "Microsoft.Compute.UserNameTextBox",
				Label:        trimLabel(val.Metadata.Description),
				Tooltip:      val.Metadata.Description,
				Visible:      true,
				DefaultValue: val.DefaultValue,
				Constraints: UserPasswordConstraints{
					Required: isRequired(val.DefaultValue),
				},
				OsPlatform: Linux.String(),
			})
			outputs[name] = fmt.Sprintf("[steps('basics').%s]", name)

		case strings.Contains(strings.ToLower(name), "password"):
			elements = append(elements, Element{
				Name: name,
				Type: "Microsoft.Common.PasswordBox",
				Label: PasswordLabel{
					Password:        trimLabel(val.Metadata.Description),
					ConfirmPassword: fmt.Sprintf("Confirm %s", trimLabel(val.Metadata.Description)),
				},
				Tooltip: val.Metadata.Description,
				Visible: true,
				Constraints: UserPasswordConstraints{
					Required: isRequired(val.DefaultValue),
				},
				Options: PasswordOptions{
					HideConfirmation: false,
				},
			})
			outputs[name] = fmt.Sprintf("[steps('basics').%s]", name)

		default:
			if isRequired(val.DefaultValue) {
				elements = append(elements, Element{
					Name:        name,
					Type:        "Microsoft.Common.TextBox",
					Label:       trimLabel(val.Metadata.Description),
					Tooltip:     val.Metadata.Description,
					Visible:     true,
					Placeholder: fmt.Sprintf("Provide value for %s", trimLabel(val.Metadata.Description)),
					Constraints: TextBoxConstraints{
						Required: true,
					},
				})
				outputs[name] = fmt.Sprintf("[steps('basics').%s]", name)
			} else {
				optionalElements = append(optionalElements, Element{
					Name:         name,
					Type:         "Microsoft.Common.TextBox",
					Label:        trimLabel(val.Metadata.Description),
					Tooltip:      val.Metadata.Description,
					Visible:      true,
					DefaultValue: getDefaultValue(val.DefaultValue),
					Placeholder:  fmt.Sprintf("Provide value for %s", trimLabel(val.Metadata.Description)),
					Constraints: TextBoxConstraints{
						Required: false,
					},
				})
				outputs[name] = fmt.Sprintf("[steps('AdditionalParameters').%s]", name)
			}
		}
	}

	if len(optionalElements) > 0 {
		UIDef.Parameters.Steps = []Step{
			{
				Name:     "AdditionalParameters",
				Label:    fmt.Sprintf("Addition Parameters for %s", trimLabel(bundleName)),
				Elements: optionalElements,
			},
		}
	}

	UIDef.Parameters.Basics = elements
	UIDef.Parameters.Outputs = outputs
	return &UIDef
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
