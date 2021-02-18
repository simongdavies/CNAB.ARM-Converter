package common

import (
	"encoding/json"
	"fmt"
	"io"
)

var CNABParam = []string{
	"cnab_resource_group",
	"cnab_azure_subscription_id",
	"cnab_azure_state_fileshare",
	"cnab_azure_state_storage_account_name",
	"cnab_azure_verbose",
	"cnab_delete_outputs_from_fileshare"}

// EnvironmentVariableNames defines environment variables names
type EnvironmentVariableNames struct {
	CnabParameterPrefix                       string
	CnabCredentialPrefix                      string
	CnabCredentialFilePrefix                  string
	CnabAction                                string
	CnabInstallationName                      string
	CnabBundleName                            string
	CnabBundleTag                             string
	CnabAzureLocation                         string
	CnabAzureClientID                         string
	CnabAzureClientSecret                     string
	CnabAzureSubscriptionID                   string
	CnabAzureTenantID                         string
	CnabAzureStateStorageAccountName          string
	CnabAzureStateStorageAccountKey           string
	CnabAzureStateStorageAccountResourceGroup string
	CnabAzureStateFileshare                   string
	Verbose                                   string
}

// GetEnvironmentVariableNames returns environment variable names
func GetEnvironmentVariableNames() EnvironmentVariableNames {
	return EnvironmentVariableNames{
		CnabParameterPrefix:                       "CNAB_PARAM_",
		CnabCredentialPrefix:                      "CNAB_CRED_",
		CnabCredentialFilePrefix:                  "CNAB_CRED_FILE_",
		CnabAction:                                "CNAB_ACTION",
		CnabInstallationName:                      "CNAB_INSTALLATION_NAME",
		CnabBundleName:                            "CNAB_BUNDLE_NAME",
		CnabBundleTag:                             "CNAB_BUNDLE_TAG",
		CnabAzureLocation:                         "CNAB_AZURE_LOCATION",
		CnabAzureClientID:                         "CNAB_AZURE_CLIENT_ID",
		CnabAzureClientSecret:                     "CNAB_AZURE_CLIENT_SECRET",
		CnabAzureSubscriptionID:                   "CNAB_AZURE_SUBSCRIPTION_ID",
		CnabAzureTenantID:                         "CNAB_AZURE_TENANT_ID",
		CnabAzureStateStorageAccountName:          "CNAB_AZURE_STATE_STORAGE_ACCOUNT_NAME",
		CnabAzureStateStorageAccountKey:           "CNAB_AZURE_STATE_STORAGE_ACCOUNT_KEY",
		CnabAzureStateStorageAccountResourceGroup: "CNAB_AZURE_STATE_STORAGE_ACCOUNT_RESOURCE_GROUP",
		CnabAzureStateFileshare:                   "CNAB_AZURE_STATE_FILESHARE",
		Verbose:                                   "VERBOSE",
	}
}

var ParameterDefaults = map[string]interface{}{
	"location":                              "[resourceGroup().Location]",
	"deployment_script_cleanup":             "Always",
	"cnab_azure_subscription_id":            "[subscription().subscriptionId]",
	"cnab_azure_state_storage_account_name": "[concat('cnabstate',uniqueString(resourceGroup().id))]",
	"cnab_resource_group":                   "[resourceGroup().name]",
	"cnab_azure_verbose":                    false,
	"debug":                                 false,
	"cnab_delete_outputs_from_fileshare":    true,
	"msi_name":                              "cnabinstall",
	"porter_version":                        "latest",
}

const AKSResourceParameterName = "aksClusterName"
const AKSResourceGroupParameterName = "aksClusterResourceGroupName"
const KubeConfigParameterName = "kubeconfig"
const LocationParameterName = "location"
const DebugParameterName = "debug"

func WriteOutput(writer io.Writer, data interface{}, indent bool) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if indent {
		encoder.SetIndent("", "\t")
	}
	err := encoder.Encode(data)

	if err != nil {
		return fmt.Errorf("Error writing response: %w", err)
	}

	return nil
}
