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
	Outputs        map[string]Output      `json:"outputs"`
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

// ContainerGroupsProperties properties defines the properties of the deployment script in the generated template
type ContainerGroupsProperties struct {
	Containers    []Container `json:"containers"`
	Volumes       []Volume    `json:"volumes"`
	OSType        string      `json:"osType"`
	RestartPolicy string      `json:"restartPolicy"`
	IPAddress     *IPAddress  `json:"ipAddress"`
}

// CustomProviderResourceProperties defines the properties of a custom RP type instance

type CustomProviderResourceProperties struct {
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ContainerProperties defines a container in a container group
type ContainerProperties struct {
	Image                string                `json:"image"`
	Ports                []ContainerPorts      `json:"ports"`
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables"`
	Command              []string              `json:"command"`
	Resources            *Resources            `json:"resources"`
	VolumeMounts         []VolumeMount         `json:"volumeMounts,omitempty"`
}

// Container defines the properties of a container in a container group
type Container struct {
	Name       string               `json:"name"`
	Properties *ContainerProperties `json:"properties"`
}

// ContainerPorts defines the port property for a container
type ContainerPorts struct {
	Port     interface{} `json:"port"`
	Protocol string      `json:"protocol"`
}

// Requests defines the requests property for a container
type Resources struct {
	Requests *Requests `json:"requests"`
}

// Requests defines the requests property for a container
type Requests struct {
	CPU        float64 `json:"cpu,omitempty"`
	MemoryInGB float64 `json:"memoryInGB,omitempty"`
}

// Volume defines the properties of a volume.
type Volume struct {
	Name      string            `json:"name,omitempty"`
	AzureFile *AzureFileVolume  `json:"azureFile,omitempty"`
	Secret    map[string]string `json:"secret,omitempty"`
}

// AzureFileVolume defines the properties of an Azure File share volume.
type AzureFileVolume struct {
	ShareName          string `json:"shareName,omitempty"`
	ReadOnly           bool   `json:"readOnly,omitempty"`
	StorageAccountName string `json:"storageAccountName,omitempty"`
	StorageAccountKey  string `json:"storageAccountKey,omitempty"`
}

// VolumeMount the properties of the volume mount.
type VolumeMount struct {
	Name      string `json:"name,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

// IPAddress defines the IP address property for the container group.
type IPAddress struct {
	Ports        *[]ContainerPorts `json:"ports,omitempty"`
	Type         string            `json:"type,omitempty"`
	DNSNameLabel string            `json:"dnsNameLabel,omitempty"`
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
	ManagedBy  string      `json:"managedBy,omitempty"`
	DependsOn  []string    `json:"dependsOn,omitempty"`
	Identity   *Identity   `json:"identity,omitempty"`
	Properties interface{} `json:"properties,omitempty"`
}

// CustomProviderProperties define the properties for a Custom RP
type CustomProviderProperties struct {
	Actions       []CustomProviderAction       `json:"actions,omitempty"`
	ResourceTypes []CustomProviderResourceType `json:"resourceTypes,omitempty"`
}

// ApplicationDefinitionProperties define the properties for an ApplicationDefinition
type ApplicationDefinitionProperties struct {
	LockLevel          string             `json:"locklevel"`
	Authorizations     []string           `json:"authorizations,omitempty"`
	Description        string             `json:"description"`
	DisplayName        string             `json:"displayName"`
	PackageFileUri     string             `json:"packageFileUri"`
	ManagementPolicy   ManagementPolicy   `json:"managementPolicy"`
	LockingPolicy      LockingPolicy      `json:"lockingPolicy"`
	NotificationPolicy NotificationPolicy `json:"notificationPolicy"`
	DeploymentPolicy   DeploymentPolicy   `json:"deploymentPolicy"`
}

type ManagementPolicy struct {
	Mode string `json:"mode"`
}

type LockingPolicy struct {
	AllowedActions []string `json:"allowedActions"`
}

type NotificationPolicy struct {
	NotificationEndpoints []string `json:"notificationEndpoints"`
}

type DeploymentPolicy struct {
	DeploymentMode string `json:"deploymentMode"`
}

// CustomProviderAction defines a custom provider action
type CustomProviderAction struct {
	Name        string `json:"name"`
	RoutingType string `json:"routingType"`
	Endpoint    string `json:"endpoint"`
}

// CustomProviderResourceType defines a custom provider action
type CustomProviderResourceType struct {
	Name        string `json:"name"`
	RoutingType string `json:"routingType"`
	Endpoint    string `json:"endpoint"`
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

// SetDeploymentScriptEnvironmentVariable sets an environment variable for the deployment script
func (template *Template) SetDeploymentScriptEnvironmentVariable(environmentVariable EnvironmentVariable) error {
	deploymentScript, err := template.FindResource(DeploymentScriptName)
	if err != nil {
		return errors.Wrap(err, "Failed to find deployment script resource")
	}

	deploymentScriptProperties, ok := deploymentScript.Properties.(DeploymentScriptProperties)

	if !ok {
		return errors.New("Failed to get deployment script resource properties")
	}

	deploymentScriptProperties.EnvironmentVariables = append(deploymentScriptProperties.EnvironmentVariables, environmentVariable)
	deploymentScript.Properties = deploymentScriptProperties

	return nil
}

// SetCustomRPAction sets an action for a CustomRP
func (template *Template) SetCustomRPAction(customRPAction CustomProviderAction) error {
	customRP, err := template.FindResource(CustomRPName)
	if err != nil {
		return errors.Wrap(err, "Failed to find customRP resource")
	}

	customRPProperties, ok := customRP.Properties.(CustomProviderProperties)

	if !ok {
		return errors.New("Failed to get customRP resource properties")
	}

	customRPProperties.Actions = append(customRPProperties.Actions, customRPAction)
	customRP.Properties = customRPProperties

	return nil
}

func (t *Template) FindResource(resourceName string) (*Resource, error) {
	for i := range t.Resources {
		resource := &t.Resources[i]
		if resource.Name == resourceName {
			return resource, nil
		}
	}

	return nil, fmt.Errorf("Resource not found in the template")
}
