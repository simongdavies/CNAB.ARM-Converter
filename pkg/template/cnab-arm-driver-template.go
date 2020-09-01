package template

import (
	"fmt"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
)

const (
	//CnabArmDriverImageName is the image name for the docker image that runs the ARM driver
	CnabArmDriverImageName = "cnabquickstarts.azurecr.io/cnabarmdriver"
)

// NewCnabArmDriverTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabArmDriverTemplate(bundleName string, bundleTag string, bundleActions []string, containerImageName string, containerImageVersion string, simplify bool) Template {

	resources := []Resource{
		{
			Type:       "Microsoft.Storage/storageAccounts",
			Name:       "[variables('cnab_azure_state_storage_account_name')]",
			APIVersion: "2019-04-01",
			Location:   "[variables('aci_location')]",
			Sku: &Sku{
				Name: "Standard_LRS",
			},
			Kind: "StorageV2",
			Properties: StorageProperties{
				Encryption: Encryption{
					KeySource: "Microsoft.Storage",
					Services: Services{
						File: File{
							Enabled: true,
						},
					},
				},
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/blobServices/containers",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/porter')]",
			APIVersion: "2019-04-01",
			Location:   "[variables('aci_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'))]",
			APIVersion: "2019-04-01",
			Location:   "[variables('aci_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Name:       ContainerGroupName,
			Type:       "Microsoft.ContainerInstance/containerGroups",
			APIVersion: "2018-10-01",
			Location:   "[variables('aci_location')]",
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts/fileServices/shares', variables('cnab_azure_state_storage_account_name'), 'default', variables('cnab_azure_state_fileshare'))]",
			},
			Properties: ContainerGroupProperties{
				Containers: []Container{
					{
						Name: ContainerName,
						Properties: ContainerProperties{
							Resources: Resources{
								Requests: Requests{
									CPU:        "1.0",
									MemoryInGb: "1.5",
								},
							},
							EnvironmentVariables: []EnvironmentVariable{
								{
									Name:  common.GetEnvironmentVariableNames().CnabAction,
									Value: "[variables('cnab_action')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabInstallationName,
									Value: "[variables('cnab_installation_name')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureLocation,
									Value: "[variables('cnab_azure_location')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureClientID,
									Value: "[variables('cnab_azure_client_id')]",
								},
								{
									Name:        common.GetEnvironmentVariableNames().CnabAzureClientSecret,
									SecureValue: "[variables('cnab_azure_client_secret')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureSubscriptionID,
									Value: "[variables('cnab_azure_subscription_id')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureTenantID,
									Value: "[variables('cnab_azure_tenant_id')]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureStateStorageAccountName,
									Value: "[variables('cnab_azure_state_storage_account_name')]",
								},
								{
									Name:        common.GetEnvironmentVariableNames().CnabAzureStateStorageAccountKey,
									SecureValue: "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value]",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabAzureStateFileshare,
									Value: "[variables('cnab_azure_state_fileshare')]",
								},
								{
									Name:  "VERBOSE",
									Value: "false",
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabBundleName,
									Value: bundleName,
								},
								{
									Name:  common.GetEnvironmentVariableNames().CnabBundleTag,
									Value: bundleTag,
								},
								{
									Name:        "AZURE_STORAGE_CONNECTION_STRING",
									SecureValue: "[concat('AccountName=', variables('cnab_azure_state_storage_account_name'), ';AccountKey=', listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value)]",
								},
							},
						},
					},
				},
				OsType:        "Linux",
				RestartPolicy: "Never",
			},
		},
	}

	parameters := map[string]Parameter{
		"cnab_action": {
			Type:          "string",
			DefaultValue:  bundleActions[0],
			AllowedValues: bundleActions,
			Metadata: &Metadata{
				Description: "The name of the action to be performed on the application instance.",
			},
		},
		"cnab_azure_client_id": {
			Type: "string",
			Metadata: &Metadata{
				Description: "AAD Client ID for Azure account authentication - used to authenticate to Azure using Service Principal for ACI creation.",
			},
		},
		"cnab_azure_client_secret": {
			Type: "securestring",
			Metadata: &Metadata{
				Description: "AAD Client Secret for Azure account authentication - used to authenticate to Azure using Service Principal for ACI creation.",
			},
		},
	}

	if !simplify {
		// TODO:The allowed values should be generated automatically based on ACI availability
		parameters["aci_location"] = Parameter{
			Type:         "string",
			DefaultValue: "[resourceGroup().Location]",
			AllowedValues: []string{
				"westus",
				"eastus",
				"westeurope",
				"westus2",
				"northeurope",
				"southeastasia",
				"eastus2",
				"centralus",
				"australiaeast",
				"uksouth",
				"southcentralus",
				"centralindia",
				"southindia",
				"northcentralus",
				"eastasia",
				"canadacentral",
				"japaneast",
			},
			Metadata: &Metadata{
				Description: "The location in which the bootstrapper ACI resources will be created.",
			},
		}

		// TODO:The allowed values should be generated automatically based on ACI availability
		parameters["cnab_azure_location"] = Parameter{
			Type:         "string",
			DefaultValue: "[resourceGroup().Location]",
			AllowedValues: []string{
				"westus",
				"eastus",
				"westeurope",
				"westus2",
				"northeurope",
				"southeastasia",
				"eastus2",
				"centralus",
				"australiaeast",
				"uksouth",
				"southcentralus",
				"centralindia",
				"southindia",
				"northcentralus",
				"eastasia",
				"canadacentral",
				"japaneast",
			},
			Metadata: &Metadata{
				Description: "The location which the cnab-azure driver will use to create ACI.",
			},
		}

		parameters["cnab_azure_subscription_id"] = Parameter{
			Type:         "string",
			DefaultValue: "[subscription().subscriptionId]",
			Metadata: &Metadata{
				Description: "Azure Subscription Id - this is the subscription to be used for ACI creation, if not specified the first (random) subscription is used.",
			},
		}

		parameters["cnab_azure_tenant_id"] = Parameter{
			Type:         "string",
			DefaultValue: "[subscription().tenantId]",
			Metadata: &Metadata{
				Description: "Azure AAD Tenant Id Azure account authentication - used to authenticate to Azure using Service Principal or Device Code for ACI creation.",
			},
		}

		parameters["cnab_installation_name"] = Parameter{
			Type:         "string",
			DefaultValue: bundleName,
			Metadata: &Metadata{
				Description: "The name of the application instance.",
			},
		}

		parameters["containerGroupName"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "Name for the container group",
			},
			DefaultValue: "[concat('cg-',uniqueString(resourceGroup().id, newGuid()))]",
		}

		parameters["containerName"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "Name for the container",
			},
			DefaultValue: "[concat('cn-',uniqueString(resourceGroup().id, newGuid()))]",
		}

		parameters["cnab_azure_state_storage_account_name"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "The storage account name for the account for the CNAB state to be stored in, by default this will be in the current resource group and will be created if it does not exist",
			},
			DefaultValue: "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		}

		parameters["cnab_azure_state_fileshare"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "The file share name in the storage account for the CNAB state to be stored in",
			},
			DefaultValue: bundleName,
		}

	}

	output := Outputs{
		Output{
			Type:  "string",
			Value: "[concat('az container logs -g ',resourceGroup().name,' -n ',variables('containerGroupName'),'  --container-name ',variables('containerName'), ' --follow')]",
		},
	}

	template := Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
		Parameters:     parameters,
		Outputs:        output,
	}

	_ = template.setContainerImage(containerImageName, containerImageVersion)

	if simplify {
		template.addSimpleVariables(bundleName, bundleTag)
	} else {
		template.addAdvancedVariables()
	}

	return template
}

func (template *Template) addAdvancedVariables() {
	variables := map[string]string{
		"cnab_action":                           "[parameters('cnab_action')]",
		"cnab_azure_client_id":                  "[parameters('cnab_azure_client_id')]",
		"cnab_azure_client_secret":              "[parameters('cnab_azure_client_secret')]",
		"cnab_azure_location":                   "[parameters('cnab_azure_location')]",
		"cnab_azure_subscription_id":            "[parameters('cnab_azure_subscription_id')]",
		"cnab_azure_tenant_id":                  "[parameters('cnab_azure_tenant_id')]",
		"cnab_installation_name":                "[parameters('cnab_installation_name')]",
		"cnab_azure_state_fileshare":            "[parameters('cnab_azure_state_fileshare')]",
		"cnab_azure_state_storage_account_name": "[parameters('cnab_azure_state_storage_account_name')]",
		"containerGroupName":                    "[parameters('containerGroupName')]",
		"containerName":                         "[parameters('containerName')]",
		"aci_location":                          "[parameters('aci_location')]",
	}

	template.Variables = variables
}

func (template *Template) addSimpleVariables(bundleName string, bundleTag string) {
	variables := map[string]string{
		"cnab_action":                           "[parameters('cnab_action')]",
		"cnab_azure_client_id":                  "[parameters('cnab_azure_client_id')]",
		"cnab_azure_client_secret":              "[parameters('cnab_azure_client_secret')]",
		"cnab_azure_location":                   "[resourceGroup().Location]",
		"cnab_azure_subscription_id":            "[subscription().subscriptionId]",
		"cnab_azure_tenant_id":                  "[subscription().tenantId]",
		"cnab_installation_name":                bundleName,
		"cnab_azure_state_fileshare":            bundleName,
		"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		"containerGroupName":                    fmt.Sprintf("[concat('cg-',uniqueString(resourceGroup().id, '%s', '%s'))]", bundleName, bundleTag),
		"containerName":                         fmt.Sprintf("[concat('cn-',uniqueString(resourceGroup().id, '%s', '%s'))]", bundleName, bundleTag),
		"aci_location":                          "[resourceGroup().Location]",
	}

	template.Variables = variables
}
