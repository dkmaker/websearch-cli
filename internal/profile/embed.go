package profile

import "embed"

//go:embed profiles/*.yaml
var builtinProfiles embed.FS
