package lib

import "embed"

//go:embed _c_files/*.c
var CFiles embed.FS
