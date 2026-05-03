package plugin

// SettingType controls which input control to render in the Settings page.
type SettingType string

const (
	SettingTypeText     SettingType = "text"
	SettingTypeNumber   SettingType = "number"
	SettingTypeCheckbox SettingType = "checkbox"
	SettingTypePassword SettingType = "password"
	SettingTypeSelect   SettingType = "select"
)

// Setting declares one user-configurable value a plugin exposes.
type Setting struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	Type        SettingType    `json:"type"`
	Default     string         `json:"default"`
	Options     []SelectOption `json:"options,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
}

// SelectOption is one item in a SettingTypeSelect control.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// PluginSettingsBlob is what the /api/settings/schema endpoint returns per plugin.
type PluginSettingsBlob struct {
	PluginID string    `json:"plugin_id"`
	NavTabID string    `json:"nav_tab_id"`
	Title    string    `json:"title"`
	Enabled  bool      `json:"enabled"`
	Requires []string  `json:"requires,omitempty"`
	Settings []Setting `json:"settings"`
}