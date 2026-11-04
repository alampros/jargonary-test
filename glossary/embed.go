// Package glossary contains the bundled project dictionary.
package glossary

import _ "embed"

//go:embed project.json
var JSON []byte
