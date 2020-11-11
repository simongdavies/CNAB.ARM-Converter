package template

import (
	"fmt"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
)

const CustomRPContainerGroupName = "cnab-custom-resource"
const CustomRPName = "public"
const CustomRPAPIVersion = "2018-09-01-preview"
const CustomRPTypeName = "installs"

// NewCnabCustomRPTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabCustomRPTemplate(bundleName string, bundleImage string) (*Template, error) {

	resources := []Resource{
		{
			Type:       "Microsoft.ManagedIdentity/userAssignedIdentities",
			Name:       "[variables('msi_name')]",
			APIVersion: "2018-11-30",
			Location:   "[parameters('location')]",
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
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'),'-caddy')]",
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/tableServices/tables",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'),'/default/',variables('stateTableName'))]",
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/tableServices/tables",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'),'/default/',variables('aysncOpTableName'))]",
			APIVersion: "2019-06-01",
			Location:   "[variables('storage_location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.ContainerInstance/containerGroups",
			APIVersion: "2019-12-01",
			Name:       CustomRPContainerGroupName,
			Location:   "[parameters('location')]",
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts/blobServices/containers', variables('cnab_azure_state_storage_account_name'),'default', 'porter')]",
				"[resourceId('Microsoft.Storage/storageAccounts/fileServices/shares', variables('cnab_azure_state_storage_account_name'), 'default', variables('cnab_azure_state_fileshare'))]",
				"[resourceId('Microsoft.Storage/storageAccounts/fileServices/shares', variables('cnab_azure_state_storage_account_name'), 'default', concat(variables('cnab_azure_state_fileshare'),'-caddy'))]",
				"[resourceId('Microsoft.Storage/storageAccounts/tableServices/tables', variables('cnab_azure_state_storage_account_name'),'default',variables('stateTableName'))]",
				"[resourceId('Microsoft.Storage/storageAccounts/tableServices/tables', variables('cnab_azure_state_storage_account_name'),'default',variables('aysncOpTableName'))]",
			},
			Identity: &Identity{
				Type: User.String(),
			},
			Properties: ContainerGroupsProperties{
				Containers: []Container{
					{
						Name: "caddy",
						Properties: &ContainerProperties{
							Image: "caddy",
							Ports: []ContainerPorts{
								{
									Port:     80,
									Protocol: "tcp",
								},
								{
									Port:     443,
									Protocol: "tcp",
								},
							},
							EnvironmentVariables: []EnvironmentVariable{
								{
									Name:  "LISTENER_PORT",
									Value: "[variables('port')]",
								},
							},
							Command: []string{
								"caddy",
								"run",
								"--config",
								"/caddy/Caddyfile",
							},
							Resources: &Resources{
								&Requests{
									CPU:        1.0,
									MemoryInGB: 1.5,
								},
							},
							VolumeMounts: []VolumeMount{
								{
									Name:      "caddy-data",
									MountPath: "/data",
								},
								{
									Name:      "caddy-file",
									MountPath: "/caddy",
								},
							},
						},
					},
					{
						Name: "custom-resource-container",
						Properties: &ContainerProperties{
							Image: "cnabquickstarts.azurecr.io/cnabcustomrphandler:latest",
							Ports: []ContainerPorts{
								{
									Port:     "[variables('port')]",
									Protocol: "tcp",
								},
							},
							EnvironmentVariables: []EnvironmentVariable{
								{
									Name:  "LISTENER_PORT",
									Value: "[variables('port')]",
								},
								{
									Name:  "CNAB_AZURE_STATE_STORAGE_RESOURCE_GROUP",
									Value: "[resourceGroup().name]",
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
									Name:  "CNAB_AZURE_SUBSCRIPTION_ID",
									Value: "[subscription().subscriptionId]",
								},
								{
									Name:  "CNAB_BUNDLE_TAG",
									Value: bundleImage,
								},
								{
									Name:  "CNAB_AZURE_RESOURCE_GROUP",
									Value: "[resourceGroup().name]",
								},
								{
									Name:  "CNAB_AZURE_VERBOSE",
									Value: "[parameters('debug')]",
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
									Name:  "CUSTOM_RP_STATE_TABLE",
									Value: "[variables('stateTableName')]",
								},
								{
									Name:  "CUSTOM_RP_ASYNC_OP_TABLE",
									Value: "[variables('aysncOpTableName')]",
								},
								{
									Name:  "CUSTOM_RP_TYPE",
									Value: fmt.Sprintf("[concat(resourceId('Microsoft.CustomProviders/resourceProviders','%s'), '/%s')]", CustomRPName, CustomRPTypeName),
								},
							},
							Command: []string{
								"/cnabcustomrphandler",
								"[if(parameters('debug'),'--debug','')]",
							},
							Resources: &Resources{
								&Requests{
									CPU:        1.0,
									MemoryInGB: 1.5,
								},
							},
						},
					},
				},
				Volumes: []Volume{
					{
						Name: "caddy-data",
						AzureFile: &AzureFileVolume{
							ShareName:          "[variables('cnab_azure_state_fileshare')]",
							ReadOnly:           false,
							StorageAccountName: "[variables('cnab_azure_state_storage_account_name')]",
							StorageAccountKey:  "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('cnab_azure_state_storage_account_name')), '2019-04-01').keys[0].value]",
						},
					},
					{
						Name: "caddy-file",
						Secret: map[string]string{
							"Caddyfile": `[base64(concat('
								{
									debug
								}
								',variables('endPointDNSPrefix'),'.northeurope.azurecontainer.io {
									log {
											output stdout
											format console
											level debug
									}
									reverse_proxy {
											to :8080
									}
									tls {
										client_auth {
											mode require_and_verify
											trusted_leaf_cert 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
                      trusted_leaf_cert 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

										}
									}
								}'))]`,
						},
					},
				},
				OSType:        "Linux",
				RestartPolicy: "Always",
				IPAddress: &IPAddress{
					DNSNameLabel: "[variables('endPointDNSPrefix')]",
					Type:         "Public",
					Ports: &[]ContainerPorts{
						{
							Port:     80,
							Protocol: "tcp",
						},
						{
							Port:     443,
							Protocol: "tcp",
						},
					},
				},
			},
		},
		{
			Type:       "Microsoft.CustomProviders/resourceProviders",
			APIVersion: CustomRPAPIVersion,
			Name:       CustomRPName,
			Location:   "[parameters('location')]",
			DependsOn: []string{
				CustomRPContainerGroupName,
			},
			Properties: CustomProviderProperties{
				ResourceTypes: []CustomProviderResourceType{
					{
						Name:        CustomRPTypeName,
						Endpoint:    "[concat('https://',variables('endPointDNSName'),'/{requestPath}')]",
						RoutingType: "Proxy",
					},
				},
				Actions: []CustomProviderAction{},
			},
		},
	}

	parameters := map[string]Parameter{
		// TODO:The allowed values should be generated automatically based on ACI availability
		"location": {
			Type: "string",
			AllowedValues: []string{
				"australiaeast",
				"brazilsouth",
				"canadacentral",
				"centralindia",
				"centralus",
				"chinaeast2",
				"eastasia",
				"eastus",
				"eastus2",
				"francecentral",
				"japaneast",
				"koreacentral",
				"northcentralus",
				"northeurope",
				"southcentralus",
				"southeastasia",
				"southindia",
				"uksouth",
				"westeurope",
				"westcentralus",
				"westus",
				"westus2",
			},
			Metadata: &Metadata{
				Description: "The location in which the resources will be created.",
			},
			DefaultValue: common.ParameterDefaults["location"],
		},

		"debug": {
			Type: "bool",
			Metadata: &Metadata{
				Description: "Creates verbose output from cnab azure driver and custom RP",
			},
			DefaultValue: common.ParameterDefaults["customrp_debug"],
		},
	}

	variables := map[string]interface{}{
		"port":                                  8080,
		"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		"cnab_azure_state_fileshare":            "[Guid(variables('cnab_azure_state_storage_account_name'),'fileshare')]",
		"contributorRoleDefinitionId":           "[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]",
		"msi_name":                              "cnabcustomrp",
		"roleAssignmentId":                      "[guid(concat(resourceGroup().id,variables('msi_name'), 'contributor'))]",
		//TODO remove hardcoded storage location once blob index feature is available https://docs.microsoft.com/en-us/azure/storage/blobs/storage-manage-find-blobs?tabs=azure-portal#regional-availability-and-storage-account-support
		"storage_location":  "canadacentral",
		"endPointDNSPrefix": "[replace(variables('cnab_azure_state_fileshare'),'-','')]",
		"endPointDNSName":   "[concat(variables('endPointDNSPrefix'),'.',tolower(replace(parameters('location'),' ','')),'.azurecontainer.io')]",
		"stateTableName":    "installstate",
		"aysncOpTableName":  "asyncops",
	}

	template := Template{
		Schema:         "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
		Parameters:     parameters,
		Variables:      variables,
		Outputs:        make(map[string]Output),
	}

	resource, err := template.FindResource(CustomRPContainerGroupName)
	if err != nil {
		return nil, fmt.Errorf("Failed to find container group resource: %w", err)
	}
	var emptystruct struct{}
	userIdentity := make(map[string]interface{}, 1)
	userIdentity["[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities',variables('msi_name'))]"] = &emptystruct
	resource.Identity.UserAssignedIdentities = userIdentity
	return &template, nil
}
