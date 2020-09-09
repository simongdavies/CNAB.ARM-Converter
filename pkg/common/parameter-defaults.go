package common

var ParameterDefaults = map[string]interface{}{
	"location":                              "[resourceGroup().Location]",
	"deployment_script_cleanup":             "Always",
	"cnab_azure_subscription_id":            "[subscription().subscriptionId]",
	"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
	"cnab_resource_group":                   "[resourceGroup().name]",
	"cnab_azure_verbose":                    false,
	"cnab_delete_outputs_from_fileshare":    true,
	"msi_name":                              "cnabinstall",
	"porter_version":                        "latest",
}
