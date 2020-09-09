package template

import (
	"fmt"
	"strings"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
)

// NewCnabArmDriverTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabArmDriverTemplate(bundleName string, bundleTag string, outputs map[string]bundle.Output, simplify bool) (*Template, error) {

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
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
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
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'))]",
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
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
						Name:  "CNAB_INSTALLATION_NAME",
						Value: "[parameters('cnab_installation_name')]",
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
					{
						Name:        "AZURE_STORAGE_CONNECTION_STRING",
						SecureValue: "[format('AccountName={0};AccountKey={1}', variables('cnab_azure_state_storage_account_name'), listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-06-01').keys[0].value)]",
					},
				},
				Arguments:     "[format('{0} {1}',variables('porter_version'),parameters('cnab_installation_name'))]",
				ScriptContent: createScript(bundleTag),
			},
		},
	}

	parameters := map[string]Parameter{
		"deploymentTime": {
			Type:         "string",
			DefaultValue: "[utcNow()]",
			Metadata: &Metadata{
				Description: "The time of the delpoyment, used to force the script to run again",
			},
		},
	}

	parameters["cnab_installation_name"] = Parameter{
		Type:         "string",
		DefaultValue: bundleName,
		Metadata: &Metadata{
			Description: "The name of the installation.",
		},
	}

	if !simplify {
		// TODO:The allowed values should be generated automatically based on ACI availability
		parameters["location"] = Parameter{
			Type: "string",
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
				"francecentral",
				"brazilsouth",
				"koreacentral",
			},
			Metadata: &Metadata{
				Description: "The location in which the resources will be created.",
			},
			DefaultValue: common.ParameterDefaults["location"],
		}

		parameters["deployment_script_cleanup"] = Parameter{
			Type: "string",
			AllowedValues: []string{
				"Always",
				"OnSuccess",
				"OnExpiration",
			},
			Metadata: &Metadata{
				Description: "When to clean up deployment script resources see https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/deployment-script-template?tabs=CLI#clean-up-deployment-script-resources.",
			},
			DefaultValue: common.ParameterDefaults["deployment_script_cleanup"],
		}

		parameters["cnab_azure_subscription_id"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "Azure Subscription Id - this is the subscription to be used for ACI creation, if not specified the first (random) subscription is used.",
			},
			DefaultValue: common.ParameterDefaults["cnab_azure_subscription_id"],
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
			DefaultValue: common.ParameterDefaults["cnab_azure_state_storage_account_name"],
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
			DefaultValue: common.ParameterDefaults["cnab_resource_group"],
		}

		parameters["cnab_azure_verbose"] = Parameter{
			Type: "bool",
			Metadata: &Metadata{
				Description: "Creates verbose output from cnab azure driver",
			},
			DefaultValue: common.ParameterDefaults["cnab_azure_verbose"],
		}

		parameters["cnab_delete_outputs_from_fileshare"] = Parameter{
			Type: "bool",
			Metadata: &Metadata{
				Description: "Deletes any bundle outputs from temporary location in fileshare",
			},
			DefaultValue: common.ParameterDefaults["cnab_delete_outputs_from_fileshare"],
		}

		parameters["msi_name"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "resource name of the user msi to execute the azure aci driver and deployment script",
			},
			DefaultValue: common.ParameterDefaults["msi_name"],
		}

		parameters["porter_version"] = Parameter{
			Type: "string",
			Metadata: &Metadata{
				Description: "The version of porter to use",
			},
			DefaultValue: common.ParameterDefaults["porter_version"],
		}
	}

	output := map[string]Output{
		"BundleOutput": {
			Type:  "array",
			Value: "[reference(resourceId('Microsoft.Resources/deploymentScripts',variables('deploymentScriptResourceName')), '2019-10-01-preview').Outputs.BundleOutputs]",
		},
	}

	template := Template{
		Schema:         "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
		Parameters:     parameters,
		Outputs:        output,
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
		"porter_version":                        "[parameters('porter_version')]",
		//TODO remove hardcoded storage location once blob index feature is available https://docs.microsoft.com/en-us/azure/storage/blobs/storage-manage-find-blobs?tabs=azure-portal#regional-availability-and-storage-account-support
		"storage_location": "canadacentral",
	}

	template.Variables = variables
}

func (template *Template) addSimpleVariables(bundleName string, bundleTag string) {
	variables := map[string]interface{}{
		"cnab_resource_group":                   "[resourceGroup().name]",
		"cnab_azure_subscription_id":            "[subscription().subscriptionId]",
		"cnab_azure_state_fileshare":            fmt.Sprintf("[guid('%s')]", bundleName),
		"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		"location":                              "[resourceGroup().location]",
		"cleanup":                               "Always",
		"cnab_azure_verbose":                    "false",
		"cnab_delete_outputs_from_fileshare":    "true",
		"msi_name":                              "cnabinstall",
		"roleAssignmentId":                      "[guid(concat(resourceGroup().id,variables('msi_name'), 'contributor'))]",
		"deploymentScriptResourceName":          fmt.Sprintf("[concat('cnab-',uniqueString(resourceGroup().id, '%s'))]", bundleName),
		"contributorRoleDefinitionId":           "[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]",
		"porter_version":                        "latest",
		//TODO remove hardcoded storage location once blob index feature is available https://docs.microsoft.com/en-us/azure/storage/blobs/storage-manage-find-blobs?tabs=azure-portal#regional-availability-and-storage-account-support
		"storage_location": "canadacentral",
	}

	template.Variables = variables
}

func createScript(tag string) string {
	//TODO allow version specification for azure driver and plugin
	//TODO remove storage migrate
	installsteps := []string{
		"set -euxo pipefail",
		"PORTER_HOME=${HOME}/.porter",
		"PORTER_URL=https://cdn.porter.sh",
		"PORTER_VERSION=${1}",
		"mkdir -p ${PORTER_HOME}",
		"curl -fsSLo ${PORTER_HOME}/porter ${PORTER_URL}/${PORTER_VERSION}/porter-linux-amd64",
		"chmod +x ${PORTER_HOME}/porter",
		"export PATH=\"${PORTER_HOME}:${PATH}\"",
		"${PORTER_HOME}/porter plugin install azure --version $PORTER_VERSION",
		"echo 'default-storage-plugin = \"azure.blob\"' > ${PORTER_HOME}/config.toml",
		"cat ${PORTER_HOME}/config.toml",
		"DOWNLOAD_LOCATION=$( curl -sL https://api.github.com/repos/deislabs/cnab-azure-driver/releases/latest | jq '.assets[]|select(.name==\"cnab-azure-linux-amd64\").browser_download_url' -r)",
		"mkdir -p ${HOME}/.cnab-azure-driver",
		"curl -sSLo ${HOME}/.cnab-azure-driver/cnab-azure ${DOWNLOAD_LOCATION}",
		"chmod +x ${HOME}/.cnab-azure-driver/cnab-azure",
		"export PATH=${HOME}/.cnab-azure-driver:${PATH}",
		"porter storage migrate",
		"set +e",
		"INSTANCE=$(${HOME}/.porter/porter show \"${2}\" -o json)",
		"set -e",
		"ACTION='upgrade'",
		//TODO if the bundle exists then check that the installation is for the same bundle type
		"if [[ -z ${INSTANCE} ]]; then ACTION='install'; fi",
		"export CNAB_ACTION=${ACTION}",
		"PARAMS=",
		"CREDS=",
		"SUFFIX=",
		"for env_var in ${!CNAB_PARAM_@};do NAME=${env_var#CNAB_PARAM_};PARAMS=$(echo ${PARAMS} --param ${NAME}=\"'${!env_var}'\"); done",
		"for env_var in ${!CNAB_CRED_FILE@};do NAME=${env_var#CNAB_CRED_FILE_};echo ${!env_var}|base64 -d > /tmp/${NAME}; done",
		"if [[  ! -z  ${!CNAB_CRED_@} ]];then CREDSFILE=$(mktemp);CREDS=\" --cred ${CREDSFILE}\";echo {\\\"Name\\\": \\\"${2}\\\" , > ${CREDSFILE};echo \\\"Credentials\\\":[ >> ${CREDSFILE}; for env_var in ${!CNAB_CRED_@};do NAME=${env_var#CNAB_CRED_};echo ${SUFFIX};if [[ ${NAME} = FILE_* ]];then NAME=${NAME#FILE_};fi;echo {\\\"Name\\\":\\\"$NAME\\\" , >> ${CREDSFILE};echo \\\"Source\\\": { >> ${CREDSFILE};if [[ ${env_var} = CNAB_CRED_FILE_* ]];then echo \\\"Path\\\": \\\"/tmp/${NAME}\\\" >> ${CREDSFILE};else echo \\\"EnvVar\\\": \\\"${env_var}\\\" >> ${CREDSFILE};fi; echo }} >> ${CREDSFILE}; if [[ -z ${SUFFIX} ]];then SUFFIX=','; fi;  done;echo ]} >> ${CREDSFILE};fi",
		"cat ${CREDSFILE}",
		fmt.Sprintf("porter bundle ${ACTION} \"${2}\" ${PARAMS} ${CREDS} --tag %s -d azure", tag),
		"OUTPUTS=$(porter inst outputs list -i \"${2}\" -o json)",
		"if [[ -z ${OUTPUTS} ]]; then echo []|jq '{BundleOutputs: .}';  else echo $OUTPUTS|jq '{BundleOutputs: .}' > ${AZ_SCRIPTS_OUTPUT_PATH}; fi"}

	builder := strings.Builder{}
	for _, cmd := range installsteps {
		builder.WriteString(fmt.Sprintf("%s;", cmd))
	}

	return builder.String()
}
