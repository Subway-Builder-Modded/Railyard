package constants

import (
	_ "embed"
	"strings"
)

const MOD_VERSION = "1.0.0"

// GameDependencyKey is the manifest dependency key used to declare the required Subway Builder version.
const GameDependencyKey = "subway-builder"

const MANIFEST_FILE_NAME = "manifest.json"

//go:embed mod_template.js
var modTemplate string

func ModTemplateWithConfig(configJSON string) string {
	return strings.Replace(modTemplate, "$CONFIG", configJSON, 1)
}
