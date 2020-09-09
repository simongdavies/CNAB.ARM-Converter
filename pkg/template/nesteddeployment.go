package template

// DeploymentResource defines a nested deployment resource
type DeploymentResource struct {
	Type       string                       `json:"type"`
	Name       string                       `json:"name"`
	APIVersion string                       `json:"apiVersion"`
	DependsOn  []string                     `json:"dependsOn,omitempty"`
	Properties DeploymentResourceProperties `json:"properties,omitempty"`
}

// DeploymentResource defines the properties of a nested deployment resource
type DeploymentResourceProperties struct {
	Mode         string       `json:"mode"`
	TemplateLink TemplateLink `json:"templatelink"`
	Parameters   map[string]ParameterValue
}

// TemplateLink defines the TemplateLink property of a nested deployment resource
type TemplateLink struct {
	Uri string `json:"uri"`
}

// ParameterValue defines the value property of a parameter in a nested deployment resource
type ParameterValue struct {
	Value interface{} `json:"value"`
}
