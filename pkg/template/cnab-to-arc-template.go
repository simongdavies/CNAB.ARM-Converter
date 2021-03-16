package template

// NewCnabarcTemplate creates a new instance of Template for running a CNAB bundle via the porter operator and arc
func NewCnabArcTemplate(bundleName string, bundleTag string) (*Template, error) {

	// TODO: does not handle parameters or credentials yet

	resources := []Resource{
		{
			Type:       "Microsoft.CNAB/installations",
			Name:       "[parameters('installation_name')]",
			APIVersion: "2021-02-12-preview",
			Location:   "westUS",
			ExtendedLocation: &ExtendedLocationProperties{
				Type: "customLocation",
				Name: "[concat('/subscriptions/', subscription().subscriptionId, 'resourceGroups/',parameters('customLocationRG'),'/providers/Microsoft.CNAB/installations/',parameters('customLocationResource'))]",
			},
			Properties: CNABInstallation{
				Reference: bundleTag,
				Action:    "[parameters('action')]",
			},
		},
	}

	parameters := map[string]Parameter{
		"installation_name": {
			Type:         "string",
			DefaultValue: bundleName,
			Metadata: &Metadata{
				Description: "The name of the installation.",
			},
		},
	}

	parameters["installation_name"] = Parameter{
		Type:         "string",
		DefaultValue: bundleName,
		Metadata: &Metadata{
			Description: "The name of the installation.",
		},
	}

	parameters["action"] = Parameter{
		Type:         "string",
		DefaultValue: "install",
		Metadata: &Metadata{
			Description: "The CNAB Action to perform.",
		},
	}

	parameters["customLocationRG"] = Parameter{
		Type: "string",
		Metadata: &Metadata{
			Description: "The resource group containing the Custom Location",
		},
	}

	parameters["customLocationResource"] = Parameter{
		Type: "string",
		Metadata: &Metadata{
			Description: "The Resource Name of the Custom Location.",
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
