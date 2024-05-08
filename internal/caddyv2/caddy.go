// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

// Package caddy .
// TODO: document
package caddy

import (
	"encoding/json"
)

// ModuleMap is a map that can contain multiple modules,
// where the map key is the module's name. (The namespace
// is usually read from an associated field's struct tag.)
// Because the module's name is given as the key in a
// module map, the name does not have to be given in the
// json.RawMessage.
type ModuleMap map[string]json.RawMessage
