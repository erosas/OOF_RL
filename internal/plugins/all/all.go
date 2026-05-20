// Package all imports every built-in plugin so their init() functions run and
// register into the global plugin factory. Remove a package from here when its
// plugin migrates to a WASM build.
package all

import (
	_ "OOF_RL/internal/plugins/debugassistant"
	_ "OOF_RL/internal/plugins/overlayhud"
)
