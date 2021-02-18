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
									Value: "[parameters('debug')]",
								},
								{
									Name:  "LOG_RESPONSE_BODY",
									Value: "[parameters('debug')]",
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
			DefaultValue: common.ParameterDefaults["debug"],
		},
	}

	variables := map[string]interface{}{
		"port":                                  8080,
		"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
		"cnab_azure_state_fileshare":            "[Guid(variables('cnab_azure_state_storage_account_name'),'fileshare')]",
		"contributorRoleDefinitionId":           "[concat('/subscriptions/', subscription().subscriptionId, '/providers/Microsoft.Authorization/roleDefinitions/', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]",
		"msi_name":                              "cnabcustomrp",
		"roleAssignmentId":                      "[guid(concat(resourceGroup().id,variables('msi_name'), 'contributor'))]",
		"endPointDNSPrefix":                     "[replace(variables('cnab_azure_state_fileshare'),'-','')]",
		"endPointDNSName":                       "[concat(variables('endPointDNSPrefix'),'.',tolower(replace(parameters('location'),' ','')),'.azurecontainer.io')]",
		"stateTableName":                        "installstate",
		"aysncOpTableName":                      "asyncops",
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
