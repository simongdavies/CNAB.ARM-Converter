package uidefinition

import "strings"

func NewViewDefinition(bundleName string, bundleDescription string) *ViewDefinition {

	return &ViewDefinition{
		Views: []View{
			{
				Kind: "Oveview",
				Properties: ViewProperties{
					Header:      strings.Title(strings.ReplaceAll(bundleName, "-", " ")),
					Description: bundleDescription,
				},
			},
		},
	}
}
