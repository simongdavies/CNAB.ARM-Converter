package template

import (
	"strings"
)

// NewCnabCustomRPTemplate creates a new instance of Template for running a CNAB bundle using cnab-azure-driver
func NewCnabMAnagedAppDefinitionTemplate(bundleName string, bundleDescription string, packageUri string) (*Template, error) {

	resources := []Resource{
		{
			Type:       "Microsoft.Solutions/applicationDefinitions",
			Name:       "[parameters('appdefname')]",
			APIVersion: "2019-07-01",
			Location:   "[resourceGroup().location]",
			Properties: ApplicationDefinitionProperties{
				LockLevel:      "none",
				Description:    bundleDescription,
				DisplayName:    strings.Title(strings.ReplaceAll(bundleName, "-", " ")),
				PackageFileUri: packageUri,
				ManagementPolicy: ManagementPolicy{
					Mode: "Managed",
				},
				DeploymentPolicy: DeploymentPolicy{
					DeploymentMode: "Complete",
				},
				LockingPolicy: LockingPolicy{
					AllowedActions: []string{},
				},
				NotificationPolicy: NotificationPolicy{
					NotificationEndpoints: []string{},
				},
			},
		},
	}

	parameters := map[string]Parameter{
		"appdefname": {
			Type: "string",
			Metadata: &Metadata{
				Description: "The name of the managed app to create.",
			},
		},
	}

	template := Template{
		Schema:         "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
		Parameters:     parameters,
	}

	return &template, nil
}
