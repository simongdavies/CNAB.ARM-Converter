package template

import (
	"fmt"
	"strings"
)

// DefaultCNABActions are the default actions for a CNAB Bundle
var DefaultCNABActions = []string{"install", "upgrade", "uninstall"}

// IsDefaultAction checks if an bundle action is a default or custom action
func IsDefaultAction(action string) bool {
	for _, a := range DefaultCNABActions {
		if action == a {
			return true
		}
	}
	return false
}

// NewCnabArmDriverTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabArmDriverTemplate(bundleName string, bundleTag string, bundleActions []string, simplify bool) (*Template, error) {

	resources := []Resource{
		{
			Type:       "Microsoft.ManagedIdentity/userAssignedIdentities",
			Name:       "[variables('msi_name')]",
			APIVersion: "2018-11-30",
			Location:   "[variables('location')]",
		},
		{
			Type:       "Microsoft.Authorization/roleAssignments",
			APIVersion: "2018-09-01-preview",
			Name:       "[variables('roleAssignmentId')]",
			DependsOn: []string{
				"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', variables('msi_name'))]",
			},
			Properties: RoleAssignment{
				RoleDefinitionId: "[variables('contributorRoleDefinitionId')]",
				PrincipalId:      "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msi_name')), '2018-11-30').principalId]",
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
			DependsOn: []string{
				"[variables('roleAssignmentId')]",
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
			Name:       "[variables('deploymentScriptResourceName')]",
			Location:   "[variables('location')]",
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts/blobServices/containers', variables('cnab_azure_state_storage_account_name'),'default', 'porter')]",
				"[resourceId('Microsoft.Storage/storageAccounts/fileServices/shares', variables('cnab_azure_state_storage_account_name'), 'default', variables('cnab_azure_state_fileshare'))]",
			},
			Identity: &Identity{
				Type: User.String(),
			},
			Kind: "AzureCLI",
			Properties: DeploymentScriptProperties{
				RetentionInterval: "P1D",
				CleanupPreference: "[variables('cleanup')]",
				StorageAccountSettings: StorageAccountSettings{
					StorageAccountKey:  "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value]",
					StorageAccountName: "[variables('cnab_azure_state_storage_account_name')]",
				},
				ForceUpdateTag: "[parameters('deploymentTime')]",
				AzCliVersion:   "2.9.1",
				Timeout:        "PT5M",
				EnvironmentVariables: []EnvironmentVariable{
					{
						Name:  "CNAB_ACTION",
						Value: "[parameters('cnab_action')]",
					},
					{
						Name:  "CNAB_INSTALLATION_NAME",
						Value: "[variables('cnab_installation_name')]",
					},
					{
						Name:  "CNAB_AZURE_LOCATION",
						Value: "[variables('location')]",
					},
					{
						Name:  "CNAB_AZURE_RESOURCE_GROUP",
						Value: "[variables('cnab_resource_group')]",
					},
					{
						Name:  "CNAB_AZURE_SUBSCRIPTION_ID",
						Value: "[variables('cnab_azure_subscription_id')]",
					},
					{
						Name:  "CNAB_AZURE_VERBOSE",
						Value: "[variables('cnab_azure_verbose')]",
					},
					{
						Name:  "CNAB_AZURE_MSI_TYPE",
						Value: "user",
					},
					{
						Name:  "CNAB_AZURE_USER_MSI_RESOURCE_ID",
						Value: "[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msi_name'))]",
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
						Value: "[variables('cnab_delete_outputs_from_fileshare')]",
					},
				},
				Arguments:     "[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msi_name'))]",
				ScriptContent: createScript(bundleTag),
			},
		},
	}

	parameters := map[string]Parameter{
		"cnab_action": {
			Type:          "string",
			DefaultValue:  bundleActions[0],
			AllowedValues: bundleActions,
			Metadata: &Metadata{
				Description: "The name of the action to be performed on the bundle instance.",
			},
		},
	}

	parameters["deploymentTime"] = Parameter{
		Type:         "string",
		DefaultValue: "[utcNow()]",
		Metadata: &Metadata{
			Description: "The time of the delpoyment, used to force the script to run again",
		},
	}

	if !simplify {
		// TODO:The allowed values should be generated automatically based on ACI availability
		parameters["location"] = Parameter{
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
				Description: "The location in which the resources will be created.",
			},
		}

		parameters["deployment_script_cleanup"] = Parameter{
			Type:         "string",
			DefaultValue: "Always",
			AllowedValues: []string{
				"Always",
				"OnSuccess",
				"OnExpiration",
			},
			Metadata: &Metadata{
				Description: "When to clean up deployment script resources see https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/deployment-script-template?tabs=CLI#clean-up-deployment-script-resources.",
			},
		}

		parameters["cnab_azure_subscription_id"] = Parameter{
			Type:         "string",
			DefaultValue: "[subscription().subscriptionId]",
			Metadata: &Metadata{
				Description: "Azure Subscription Id - this is the subscription to be used for ACI creation, if not specified the first (random) subscription is used.",
			},
		}

		parameters["cnab_installation_name"] = Parameter{
			Type:         "string",
			DefaultValue: bundleName,
			Metadata: &Metadata{
				Description: "The name of the application instance.",
			},
		}

		parameters["deploymentScriptResourceName"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "Name for the container",
			},
			DefaultValue: "[concat('cnab-',uniqueString(resourceGroup().id, newGuid()))]",
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
			DefaultValue: fmt.Sprintf("[guid('%s')]", bundleName),
		}

		parameters["cnab_resource_group"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "The resource group for the cnab azure driver to create ACI container group in",
			},
			DefaultValue: "[resourceGroup().name]",
		}

		parameters["cnab_azure_verbose"] = Parameter{
			Type: "bool",
			Metadata: &Metadata{
				Description: "Creates verbose output from cnab azure driver",
			},
			DefaultValue: false,
		}

		parameters["cnab_delete_outputs_from_fileshare"] = Parameter{
			Type: "bool",
			Metadata: &Metadata{
				Description: "Deletes any bundle outputs from temporary location in fileshare",
			},
			DefaultValue: true,
		}

		parameters["msi_name"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "resource name of the user msi to execute the azure aci driver and deployment script",
			},
			DefaultValue: "cnabinstall",
		}

	}

	// output := Outputs{
	// 	Output{
	// 		Type:  "string",
	// 		Value: "[concat('az container logs -g ',resourceGroup().name,' -n ',variables('containerGroupName'),'  --container-name ',variables('containerName'), ' --follow')]",
	// 	},
	// }

	template := Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
		Parameters:     parameters,
		Outputs:        Outputs{},
	}

	resource, err := template.FindResource(DeploymentScriptName)
	if err != nil {
		return nil, fmt.Errorf("Failed to find deployment script resource: %w", err)
	}
	var emptystruct struct{}
	userIdentity := make(map[string]interface{}, 1)
	userIdentity["[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msi_name'))]"] = &emptystruct
	resource.Identity.UserAssignedIdentities = userIdentity

	if simplify {
		template.addSimpleVariables(bundleName, bundleTag)
	} else {
		template.addAdvancedVariables()
	}

	return &template, nil
}

func (template *Template) addAdvancedVariables() {
	variables := map[string]interface{}{
		"cnab_resource_group":                   "[parameters('cnab_resource_group')]",
		"cnab_azure_subscription_id":            "[parameters('cnab_azure_subscription_id')]",
		"cnab_installation_name":                "[parameters('cnab_installation_name')]",
		"cnab_azure_state_fileshare":            "[parameters('cnab_azure_state_fileshare')]",
		"cnab_azure_state_storage_account_name": "[parameters('cnab_azure_state_storage_account_name')]",
		"location":                              "[parameters('location')]",
		"cleanup":                               "[parameters('deployment_script_cleanup')]",
		"cnab_azure_verbose":                    "[parameters('cnab_azure_verbose')]",
		"cnab_delete_outputs_from_fileshare":    "[parameters('cnab_delete_outputs_from_fileshare')]",
		"msi_name":                              "[parameters('msi_name')]",
		"roleAssignmentId":                      "[guid(concat(resourceGroup().id,parameters('msi_name'), 'contributor'))]",
		"deploymentScriptResourceName":          "[parameters('deploymentScriptResourceName')]",
		"contributorRoleDefinitionId":           "[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]",
		"default_actions":                       []string{"install", "uninstall", "upgrade"},
	}

	template.Variables = variables
}

func (template *Template) addSimpleVariables(bundleName string, bundleTag string) {
	variables := map[string]interface{}{
		"cnab_resource_group":                   "[resourceGroup().name]",
		"cnab_azure_subscription_id":            "[subscription().subscriptionId]",
		"cnab_installation_name":                bundleName,
		"cnab_azure_state_fileshare":            fmt.Sprintf("[guid('%s')]", bundleName),
		"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		"location":                              "[resourceGroup().location]",
		"cleanup":                               "always",
		"cnab_azure_verbose":                    "false",
		"cnab_delete_outputs_from_fileshare":    "true",
		"msi_name":                              "cnabinstall",
		"roleAssignmentId":                      "[guid(concat(resourceGroup().id,variables('msi_name'), 'contributor'))]",
		"deploymentScriptResourceName":          fmt.Sprintf("[concat('cnab-',uniqueString(resourceGroup().id, '%s'))]", bundleName),
		"contributorRoleDefinitionId":           "[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]",
		"default_actions":                       []string{"install", "uninstall", "upgrade"},
	}

	template.Variables = variables
}

func createScript(tag string) string {
	scriptPrefix := "[format('"
	scriptSuffix := "',if(contains(variables('default_actions'),tolower(parameters('cnab_action'))),parameters('cnab_action'),'invoke'),variables('cnab_installation_name'),if(contains(variables('default_actions'),tolower(parameters('cnab_action'))),'',format('--action {0}',parameters('cnab_action'))))]"
	installsteps := []string{
		"STDERR=$(mktemp)",
		"STDOUT=$(mktemp)",
		"exec > $STDOUT",
		"exec 2> $STDERR",
		"set -euo pipefail",
		"echo Installing Porter",
		"curl https://cdn.deislabs.io/porter/latest/install-linux.sh|/bin/bash",
		"export PATH=\"$HOME/.porter:$PATH\"",
		"echo Installed  $($HOME/.porter/porter version)",
		"echo Installing CNAB azure driver",
		"DOWNLOAD_LOCATION=$( curl -sL https://api.github.com/repos/deislabs/cnab-azure-driver/releases/latest | jq ''.assets[]|select(.name==\"cnab-azure-linux-amd64\").browser_download_url'' -r)",
		"mkdir -p $HOME/.cnab-azure-driver",
		"curl -sSLo $HOME/.cnab-azure-driver/cnab-azure $DOWNLOAD_LOCATION",
		"chmod +x $HOME/.cnab-azure-driver/cnab-azure",
		"export PATH=$HOME/.cnab-azure-driver:$PATH",
		"echo Installed  $($HOME/.cnab-azure-driver/cnab-azure version)"}
	portercmdBuilder := strings.Builder{}
	portercmdBuilder.WriteString("porter bundle {0} ")
	portercmdBuilder.WriteString("{1} ")
	portercmdBuilder.WriteString("{2} ")
	// if !IsDefaultAction(action) {
	// 	portercmdBuilder.WriteString(fmt.Sprintf("--action %s ", action))
	// }
	portercmdBuilder.WriteString(fmt.Sprintf("--tag %s -d azure ", tag))
	//portercmd := fmt.Sprintf("porter bundle install test --tag %s -d azure", tag)
	outputcmd := "jq -n --arg stdout \"$(cat $STDOUT)\" --arg stderr  \"$(cat $STDERR)\" ''{{\"stdout\":$stdout,\"stderr\": $stderr}}'' > $AZ_SCRIPTS_OUTPUT_PATH"

	builder := strings.Builder{}
	builder.WriteString(scriptPrefix)
	for _, cmd := range installsteps {
		builder.WriteString(fmt.Sprintf("%s;", cmd))
	}
	builder.WriteString(fmt.Sprintf("%s;", portercmdBuilder.String()))
	builder.WriteString(fmt.Sprintf("%s;", outputcmd))
	builder.WriteString(scriptSuffix)
	return builder.String()
}
