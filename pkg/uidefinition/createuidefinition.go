package uidefinition

import (
	"sort"
)

type OsPlatform int

const (
	Windows OsPlatform = iota
	Linux
)

func (i OsPlatform) String() string {
	return [...]string{"Windows", "Linux"}[i]
}

type ResourceFilter int

const (
	OnBasics ResourceFilter = iota
	All
)

func (i ResourceFilter) String() string {
	return [...]string{"onBasics", "all"}[i]
}

type InfoboxIcon int

const (
	None InfoboxIcon = iota
	Info
	Warning
	Error
)

func (i InfoboxIcon) String() string {
	return [...]string{"None", "Info", "Warning", "Error"}[i]
}

type CreateUIDefinition struct {
	Schema     string     `json:"$schema"`
	Handler    string     `json:"handler"`
	Version    string     `json:"version"`
	Parameters Parameters `json:"parameters"`
}

type Parameters struct {
	Config  Config            `json:"config,omitempty"`
	Basics  []Element         `json:"basics"`
	Steps   []Step            `json:"steps"`
	Outputs map[string]string `json:"outputs"`
}

type Step struct {
	Name     string    `json:"name"`
	Label    string    `json:"label"`
	Elements []Element `json:"elements"`
}

type Element struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Label        interface{}    `json:"label"`
	Visible      bool           `json:"visible"`
	Tooltip      string         `json:"toolTip,omitempty"`
	DefaultValue interface{}    `json:"defaultValue,omitempty"`
	Placeholder  string         `json:"placeholder,omitempty"`
	Options      interface{}    `json:"options,omitempty"`
	Constraints  interface{}    `json:"constraints,omitempty"`
	ResourceType string         `json:"resourceType,omitempty"`
	OsPlatform   string         `json:"osPlatform,omitempty"`
	Request      *ArmAPIRequest `json:"request,omitempty"`
}

type Config struct {
	IsWizard bool         `json:"isWizard,omitempty"`
	Basics   BasicsConfig `json:"basics,omitempty"`
}

type BasicsConfig struct {
	Description   string         `json:"description,omitempty"`
	Subscription  *Subscription  `json:"subscription,omitempty"`
	ResourceGroup *ResourceGroup `json:"resourceGroup,omitempty"`
	Location      *Location      `json:"location,omitempty"`
}

type Subscription struct {
	Description       string              `json:"description,omitempty"`
	Constraints       ResourceConstraints `json:"constraints,omitempty"`
	ResourceProviders []string            `json:"resourceProviders,omitempty"`
}

type ResourceGroup struct {
	Constraints   ResourceConstraints `json:"constraints,omitempty"`
	AllowExisting bool                `json:"allowExisting,omitempty"`
}

type ResourceConstraints struct {
	Validations []ResourceValidation `json:"validations,omitempty"`
}

type ResourceValidation struct {
	Permission string `json:"permission,omitempty"`
	Message    string `json:"message,omitempty"`
}

type Location struct {
	Label         string   `json:"label,omitempty"`
	Tooltip       string   `json:"toolTip,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
	Visible       bool     `json:"visible,omitempty"`
}

type InfoboxOptions struct {
	Icon string `json:"icon,omitempty"`
	Text string `json:"text,omitempty"`
	Uri  string `json:"uri,omitempty"`
}

type ArmAPIRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type ResourceSelectorOptions struct {
	Filter ResourceSelectorFilter `json:"filter,omitempty"`
}

type ResourceSelectorFilter struct {
	Subscription string `json:"subscription,omitempty"`
	Location     string `json:"location,omitempty"`
}

type UserPasswordConstraints struct {
	Required          bool   `json:"required,omitempty"`
	Regex             string `json:"regex,omitempty"`
	ValidationMessage string `json:"validationMessage,omitempty"`
}

type CheckBoxConstraints struct {
	Required          bool   `json:"required,omitempty"`
	ValidationMessage string `json:"validationMessage,omitempty"`
}

type TextBoxConstraints struct {
	Required    bool                 `json:"required,omitempty"`
	Validations []TextboxValidations `json:"validations,omitempty"`
}

type TextboxValidations struct {
	Regex   string `json:"regex,omitempty"`
	Message string `json:"message,omitempty"`
}

type PasswordLabel struct {
	Password        string `json:"password,omitempty"`
	ConfirmPassword string `json:"confirmPassword,omitempty"`
}

type PasswordOptions struct {
	HideConfirmation bool `json:"hideConfirmation,omitempty"`
}

type Blade struct {
	Label        string `json:"label,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
}

type DisplayElement struct {
	Name              string `json:"name,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	DisplayOrder      int    `json:"displayOrder,omitempty"`
	UIType            string `json:"uitype,omitempty"`
	ValidationRegex   string `json:"validationRegex,omitempty"`
	ValidationMessage string `json:"validationMessage,omitempty"`
	Tooltip           string `json:"toolTip,omitempty"`
	Bladename         string `json:"bladeName,omitempty"`
	Hide              bool   `json:"hide,omitempty"`
}

type DisplayElements struct {
	Elements []DisplayElement `json:"elements,omitempty"`
}

type CustomSettings struct {
	DisplayElements
	Blades map[string]Blade `json:"blades,omitempty"`
}

type by func(p1, p2 *DisplayElement) bool

func (by by) Sort(settings DisplayElements) {
	cs := &displayElementsSorter{
		settings: settings,
		by:       by,
	}
	sort.Sort(cs)
}

type displayElementsSorter struct {
	settings DisplayElements
	by       func(p1, p2 *DisplayElement) bool
}

func (cs *displayElementsSorter) Len() int {
	return len(cs.settings.Elements)
}

func (cs *displayElementsSorter) Swap(i, j int) {
	cs.settings.Elements[i], cs.settings.Elements[j] = cs.settings.Elements[j], cs.settings.Elements[i]
}

func (cs *displayElementsSorter) Less(i, j int) bool {
	return cs.by(&cs.settings.Elements[i], &cs.settings.Elements[j])
}

func (c DisplayElements) SortByName() {
	f := func(c1, c2 *DisplayElement) bool {
		return c1.Name < c2.Name
	}
	by(f).Sort(c)
}

func (c DisplayElements) SortByDisplayOrder() {
	f := func(c1, c2 *DisplayElement) bool {
		return c1.DisplayOrder < c2.DisplayOrder
	}
	by(f).Sort(c)
}
