package uidefinition

type ViewDefinition struct {
	Schema         string `json:"$schema"`
	ContentVersion string `json:"contentVersion"`
	Views          []View `json:"views,omitempty"`
}

type View struct {
	Kind       string         `json:"kind"`
	Properties ViewProperties `json:"properties"`
}

type ViewProperties struct {
	Header      string   `json:"header"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
}
