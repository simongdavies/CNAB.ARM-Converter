package template

import (
	"fmt"
)

// NewCnabArmDriverTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabArmDriverTemplate(bundleName string, bundleTag string, bundleActions []string, simplify bool) (*Template, error) {

	resources := []Resource{
		{
			Type:       "Microsoft.ManagedIdentity/userAssignedIdentities",
			Name:       "[variables('msiName')]",
			APIVersion: "2018-11-30",
			Location:   "[variables('location')]",
		},
		{
			Type:       "Microsoft.Authorization/roleAssignments",
			APIVersion: "2018-09-01-preview",
			Name:       "[variables('roleAssignmentId')]",
			DependsOn: []string{
				"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', variables('msiName'))]",
			},
			Properties: RoleAssignment{
				RoleDefinitionId: "[variables('contributorRoleDefinitionId')]",
				PrincipalId:      "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msiName')), '2018-11-30').principalId]",
				Scope:            "[resourceGroup().id]",
				PrincipalType:    "ServicePrincipal",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts",
			Name:       "[variables('cnab_azure_state_storage_account_name')]",
			APIVersion: "2019-04-01",
			Location:   "[variables('location')]",
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
			Location:   "[variables('location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'))]",
			APIVersion: "2019-04-01",
			Location:   "[variables('location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Resources/deploymentScripts",
			APIVersion: "2019-10-01-preview",
			Name:       DeploymentScriptName,
			Location:   "[variables('location')]",
			DependsOn: []string{
				"[variables('msiName')]",
				"[resourceId('Microsoft.Storage/storageAccounts/fileServices/shares', variables('cnab_azure_state_storage_account_name'), 'default', variables('cnab_azure_state_fileshare'))]",
			},
			Identity: &Identity{
				Type: User,
			},
			Kind: "AzureCLI",
			Properties: DeploymentScriptProperties{
				RetentionInterval: "P1D",
				StorageAccountSettings: StorageAccountSettings{
					StorageAccountKey:  "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value]",
					StorageAccountName: "[variables('cnab_azure_state_storage_account_name')]",
				},
				ForceUpdateTag: "[parameters('deploymentTime')]",
				AzCliVersion:   "2.9.1",
				Timeout:        "PT5M",
				EnvironmentVariables: []EnvironmentVariable{
					{
						Name:  "CNAB_AZURE_LOCATION",
						Value: "[resourceGroup().location]",
					},
					{
						Name:  "CNAB_AZURE_RESOURCE_GROUP",
						Value: "[resourceGroup().name]",
					},
					{
						Name:  "CNAB_AZURE_SUBSCRIPTION_ID",
						Value: "[subscription().subscriptionId]",
					},
					{
						Name:  "CNAB_AZURE_VERBOSE",
						Value: "false",
					},
					{
						Name:  "CNAB_AZURE_MSI_TYPE",
						Value: "user",
					},
					{
						Name:  "CNAB_AZURE_USER_MSI_RESOURCE_ID",
						Value: "[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msiName'))]",
					},
					{
						Name:  "CNAB_AZURE_STATE_STORAGE_ACCOUNT_NAME",
						Value: "[variables('cnab_azure_state_storage_account_name')]",
					},
					{
						Name:        "CNAB_AZURE_STATE_STORAGE_ACCOUNT_KEY",
						SecureValue: "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value]",
					},
					{
						Name:  "CNAB_AZURE_STATE_FILESHARE",
						Value: "[variables('cnab_azure_state_fileshare')]",
					},
					{
						Name:  "CNAB_AZURE_DELETE_OUTPUTS_FROM_FILESHARE",
						Value: "true",
					},
				},
				Arguments: "[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msiName'))]",
				ScriptContent: `
          STDERR=$(mktemp)
          STDOUT=$(mktemp)
          exec > $STDOUT
          exec 2> $STDERR
          echo Installing Porter
          curl https://cdn.deislabs.io/porter/latest/install-linux.sh|/bin/bash
          export PATH=\"${HOME}/.porter:${PATH}\"
          echo Installed  $(${HOME}/.porter/porter version)
          echo Installing CNAB azure driver
          DOWNLOAD_LOCATION=$( curl -sL https://api.github.com/repos/deislabs/cnab-azure-driver/releases/latest | jq '.assets[]|select(.name==\"cnab-azure-linux-amd64\").browser_download_url' -r)
          mkdir -p ${HOME}/.cnab-azure-driver
          curl -sSLo ${HOME}/.cnab-azure-driver/cnab-azure ${DOWNLOAD_LOCATION}
          chmod +x ${HOME}/.cnab-azure-driver/cnab-azure
          export PATH=${HOME}/.cnab-azure-driver:${PATH}
          echo Installed  $(${HOME}/.cnab-azure-driver/cnab-azure version)
          porter install test --tag deislabs/porter-example-exec-outputs-bundle:0.1.0 -d azure
          jq -n --arg stdout \"$(cat $STDOUT)\" --arg stderr  \"$(cat $STDERR)\" '{\"stdout\":$stdout,\"stderr\": $stderr}' > $AZ_SCRIPTS_OUTPUT_PATH`,
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

	resource, err := template.FindResource(DeploymentScriptName)
	if err != nil {
		return nil, fmt.Errorf("Failed to find deployment script resource: %w", err)
	}

	userIdentity := make(map[string]interface{}, 1)
	userIdentity["resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msiName'))]"] = nil
	resource.Identity.UserAssignedIdentities = userIdentity

	if simplify {
		template.addSimpleVariables(bundleName, bundleTag)
	} else {
		template.addAdvancedVariables()
	}

	return &template, nil
}

func (template *Template) addAdvancedVariables() {
	variables := map[string]string{
		"cnab_action":                           "[parameters('cnab_action')]",
		"cnab_azure_location":                   "[parameters('cnab_azure_location')]",
		"cnab_azure_subscription_id":            "[parameters('cnab_azure_subscription_id')]",
		"cnab_azure_tenant_id":                  "[parameters('cnab_azure_tenant_id')]",
		"cnab_installation_name":                "[parameters('cnab_installation_name')]",
		"cnab_azure_state_fileshare":            "[parameters('cnab_azure_state_fileshare')]",
		"cnab_azure_state_storage_account_name": "[parameters('cnab_azure_state_storage_account_name')]",
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
