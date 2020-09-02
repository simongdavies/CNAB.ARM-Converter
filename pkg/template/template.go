package template

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	//DeploymentScriptName is the value of the ContainerGroup Resource Name property in the generated template
	DeploymentScriptName = "[variables('deploymentScriptResourceName')]"
)

// Template defines an ARM Template that can run a CNAB Bundle
type Template struct {
	Schema         string                 `json:"$schema"`
	ContentVersion string                 `json:"contentVersion"`
	Parameters     map[string]Parameter   `json:"parameters"`
	Variables      map[string]interface{} `json:"variables"`
	Resources      []Resource             `json:"resources"`
	Outputs        Outputs                `json:"outputs"`
}

// Metadata defines the metadata for a template parameter
type Metadata struct {
	Description string `json:"description,omitempty"`
}

// Parameter defines a template parameter
type Parameter struct {
	Type          string      `json:"type"`
	DefaultValue  interface{} `json:"defaultValue,omitempty"`
	AllowedValues interface{} `json:"allowedValues,omitempty"`
	Metadata      *Metadata   `json:"metadata,omitempty"`
	MinValue      *int        `json:"minValue,omitempty"`
	MaxValue      *int        `json:"maxValue,omitempty"`
	MinLength     *int        `json:"minLength,omitempty"`
	MaxLength     *int        `json:"maxLength,omitempty"`
}

// Sku is defines a SKU for template resource
type Sku struct {
	Name string `json:"name,omitempty"`
}

// File defines if encryption is enabled for file shares in a storage account created by the template
type File struct {
	Enabled bool `json:"enabled"`
}

// Services defines Services that can be encrypted in a storage account
type Services struct {
	File File `json:"file"`
}

// Encryption defines the encryption properties for the storage account in the generated template
type Encryption struct {
	KeySource string   `json:"keySource"`
	Services  Services `json:"services"`
}

// StorageProperties defines the properties of the storage account in the generated template
type StorageProperties struct {
	Encryption Encryption `json:"encryption"`
}

// DeploymentScript properties defines the properties of the deployment script in the generated template
// TODO fix Retention Interval and Timeout types
type DeploymentScriptProperties struct {
	RetentionInterval      string                 `json:"retentionInterval"`
	Timeout                string                 `json:"timeout"`
	ForceUpdateTag         string                 `json:"forceUpdateTag"`
	AzCliVersion           string                 `json:"azCliVersion"`
	Arguments              string                 `json:"arguments"`
	ScriptContent          string                 `json:"scriptContent"`
	EnvironmentVariables   []EnvironmentVariable  `json:"environmentVariables"`
	StorageAccountSettings StorageAccountSettings `json:"storageAccountSettings"`
	CleanupPreference      string                 `json:"cleanupPreference"`
}

// StorageAccountSettings defines the storage account settings for the storage account settings property of the deployment script in the generated template

type StorageAccountSettings struct {
	StorageAccountKey  string `json:"storageAccountKey"`
	StorageAccountName string `json:"storageAccountName"`
}

// EnvironmentVariable defines the environment variables that are created for the deployment script in the generated template
type EnvironmentVariable struct {
	Name        string `json:"name"`
	SecureValue string `json:"secureValue,omitempty"`
	Value       string `json:"value,omitempty"`
}

// Resource defines a resource in the generated template
type Resource struct {
	Condition  string      `json:"condition,omitempty"`
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	APIVersion string      `json:"apiVersion"`
	Location   string      `json:"location"`
	Sku        *Sku        `json:"sku,omitempty"`
	Kind       string      `json:"kind,omitempty"`
	DependsOn  []string    `json:"dependsOn,omitempty"`
	Identity   *Identity   `json:"identity,omitempty"`
	Properties interface{} `json:"properties"`
}

// Identity defines managed identity
type Identity struct {
	Type                   string `json:"type"`
	UserAssignedIdentities map[string]interface{}
}

// IdentityType is the type of MSI
type IdentityType int

const (
	System IdentityType = iota
	User
	None
)

func (i IdentityType) String() string {
	return [...]string{"SystemAssigned", "UserAssigned", "None"}[i]
}

// RoleAssignment defines a role assignment in the generated template
type RoleAssignment struct {
	RoleDefinitionId string `json:"roleDefinitionId"`
	PrincipalId      string `json:"principalId"`
	Scope            string `json:"scope"`
	PrincipalType    string `json:"principalType"`
}

// Output defines an output in the generated template
type Output struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Outputs defines the outputs in the genreted template
type Outputs map[string]Output

// SetDeploymentScriptEnvironmentVariable sets an environment variable for the deployment script
func (template *Template) SetDeploymentScriptEnvironmentVariable(environmentVariable EnvironmentVariable) error {
	deploymentScriptProperties, err := findDeploymentsScript(template)
	if err != nil {
		return err
	}

	deploymentScriptProperties.EnvironmentVariables = append(deploymentScriptProperties.EnvironmentVariables, environmentVariable)

	return nil
}

func findDeploymentsScript(template *Template) (*DeploymentScriptProperties, error) {
	resource, err := template.FindResource(DeploymentScriptName)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find deployment script resource")
	}

	if deploymentScriptProperties, ok := resource.Properties.(DeploymentScriptProperties); ok {
		return &deploymentScriptProperties, nil
	}

	return nil, fmt.Errorf("Deployment Script not found in the template")
}

func (t *Template) FindResource(resourceName string) (*Resource, error) {
	for i := range t.Resources {
		resource := &t.Resources[i]
		if resource.Name == resourceName {
			return resource, nil
		}
	}

	return nil, fmt.Errorf("Deployment Script not found in the template")
}
