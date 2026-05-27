// Package all imports every native plugin so their init() functions run and
// register into the global plugin factory.
package all

import (
	_ "OOF_RL/internal/overlayhud"
)
