package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/Muxcore-Media/core/pkg/contracts"
)

func (m *Module) Settings() []contracts.SettingDef {
	return m.settingsDefs()
}

func (m *Module) UpdateSetting(key, value string) error {
	return m.updateSetting(key, value)
}

func (m *Module) settingsDefs() []contracts.SettingDef {
	m.mu.RLock()
	dir := m.dataDir
	m.mu.RUnlock()
	return []contracts.SettingDef{
		{
			Key:         "data_dir",
			Label:       "Schema Data Directory",
			Type:        contracts.SettingTypeString,
			Value:       dir,
			Default:     "./schemas",
			Description: "Directory for named JSON Schema files (VALIDATE_DATA_DIR)",
			Group:       "Schemas",
		},
	}
}

func (m *Module) updateSetting(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "data_dir", "VALIDATE_DATA_DIR":
		if value == "" {
			return fmt.Errorf("data_dir must not be empty")
		}
		fi, err := os.Stat(value)
		if err != nil {
			return fmt.Errorf("data_dir must be an existing directory: %w", err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("data_dir must be a directory")
		}
		m.mu.Lock()
		m.dataDir = value
		m.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
}
