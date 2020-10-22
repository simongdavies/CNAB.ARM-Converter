package uidefinition

import "strings"

func NewViewDefinition(bundleName string, bundleDescription string) *ViewDefinition {

	return &ViewDefinition{
		Schema:         "https://schema.management.azure.com/schemas/viewdefinition/0.0.1-preview/ViewDefinition.json#",
		ContentVersion: "0.0.0.1",
		Views: []View{
			{
				Kind: "Oveview",
				Properties: ViewProperties{
					Header:      strings.Title(strings.ReplaceAll(bundleName, "-", " ")),
					Description: bundleDescription,
				},
			},
		},
		Commands: []string{},
	}
}
