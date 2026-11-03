// Package glossary contains the project's own dictionary, published by every instance.
package glossary

import _ "embed"

//go:embed project.json
var JSON []byte
