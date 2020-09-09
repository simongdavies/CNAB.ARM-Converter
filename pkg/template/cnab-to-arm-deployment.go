package template

import (
	"github.com/simongdavies/CNAB.ARM-Converter/pkg/common"
)

// NewCnabArmDriverTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabArmDeployment(bundleName string, uri string, simplyfy bool) *DeploymentResource {
	resource := DeploymentResource{
		Type:       "Microsoft.Resources/deployments",
		Name:       bundleName,
		APIVersion: "2020-06-01",
		Properties: DeploymentResourceProperties{
			Mode: "Incremental",
			TemplateLink: TemplateLink{
				Uri: uri,
			},
		},
	}

	parameters := map[string]ParameterValue{
		"cnab_installation_name": {
			Value: "TODO: update with install name or delete this parameter to use default of bundle name",
		},
	}

	if !simplyfy {
		parameters["location"] = ParameterValue{
			Value: common.ParameterDefaults["location"],
		}

		parameters["deployment_script_cleanup"] = ParameterValue{
			Value: common.ParameterDefaults["deployment_script_cleanup"],
		}

		parameters["cnab_azure_subscription_id"] = ParameterValue{
			Value: common.ParameterDefaults["cnab_azure_subscription_id"],
		}

		parameters["deploymentScriptResourceName"] = ParameterValue{
			Value: "TODO: update to a resource name or delete this parameter to use default deploymentScript resource name",
		}

		parameters["cnab_azure_state_storage_account_name"] = ParameterValue{
			Value: common.ParameterDefaults["cnab_azure_state_storage_account_name"],
		}

		parameters["cnab_azure_state_fileshare"] = ParameterValue{
			Value: "TODO: update to a file share name or delete this parameter to use default file share name",
		}

		parameters["cnab_resource_group"] = ParameterValue{
			Value: common.ParameterDefaults["cnab_resource_group"],
		}

		parameters["cnab_azure_verbose"] = ParameterValue{
			Value: common.ParameterDefaults["cnab_azure_verbose"],
		}

		parameters["cnab_delete_outputs_from_fileshare"] = ParameterValue{
			Value: common.ParameterDefaults["cnab_delete_outputs_from_fileshare"],
		}

		parameters["msi_name"] = ParameterValue{
			Value: common.ParameterDefaults["msi_name"],
		}

		parameters["porter_version"] = ParameterValue{
			Value: common.ParameterDefaults["porter_version"],
		}

	}

	resource.Properties.Parameters = parameters
	return &resource
}
