package template

import (
	"errors"
	"fmt"

	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
)

const CustomRPContainerGroupName = "cnab-custom-resource"
const CustomRPName = "public"
const CustomRPAPIVersion = "2018-09-01-preview"
const CustomRPTypeName = "installs"

// NewCnabCustomRPTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabCustomRPTemplate(bundleName string, bundleImage string, customTypeInfo *Type) (*Template, error) {
	typeName := CustomRPTypeName
	if customTypeInfo != nil {
		typeName = customTypeInfo.Type
	}

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
			Location:   "[parameters('location')]",
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
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'))]",
			APIVersion: "2019-06-01",
			Location:   "[parameters('location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/fileServices/shares",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'), '/default/', variables('cnab_azure_state_fileshare'),'-caddy')]",
			APIVersion: "2019-06-01",
			Location:   "[parameters('location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/tableServices/tables",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'),'/default/',variables('stateTableName'))]",
			APIVersion: "2019-06-01",
			Location:   "[parameters('location')]",
			DependsOn: []string{
				"[variables('cnab_azure_state_storage_account_name')]",
			},
		},
		{
			Type:       "Microsoft.Storage/storageAccounts/tableServices/tables",
			Name:       "[concat(variables('cnab_azure_state_storage_account_name'),'/default/',variables('aysncOpTableName'))]",
			APIVersion: "2019-06-01",
			Location:   "[parameters('location')]",
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
									Name:  "RESOURCE_TYPE",
									Value: CustomRPName,
								},
								{
									Name:  "LOG_REQUEST_BODY",
									Value: "[(parameters('debug')]",
								},
								{
									Name:  "LOG_RESPONSE_BODY",
									Value: "[(parameters('debug')]",
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
								',variables('endPointDNSPrefix'),'.',parameters('location'),'.azurecontainer.io {
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
						Name:        typeName,
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
				Description: "Creates debug output from cnab azure driver and custom RP",
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

	resource, err = template.FindResource(CustomRPName)
	if err != nil {
		return nil, fmt.Errorf("Failed to find custom resource: %w", err)
	}

	if customTypeInfo != nil {
		customProviderProperties, ok := resource.Properties.(CustomProviderProperties)
		if !ok {
			return nil, errors.New("Failed to get custom resource properties")
		}

		for nestedTypeName := range customTypeInfo.ChildTypes {

			customProviderProperties.ResourceTypes = append(customProviderProperties.ResourceTypes, CustomProviderResourceType{
				Name:        fmt.Sprintf("%s/%s", typeName, nestedTypeName),
				Endpoint:    "[concat('https://',variables('endPointDNSName'),'/{requestPath}')]",
				RoutingType: "Proxy",
			})
		}

		resource.Properties = customProviderProperties
	}

	return &template, nil

}
