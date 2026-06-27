package server

import (
	"embed"
	"io/fs"
)

//go:embed assets/base.css assets/theme.js
var assetsFS embed.FS

// assetsSubFS 返回以 assets 为根的子文件系统，便于 http.FileServer 直接服务。
var assetsSubFS, _ = fs.Sub(assetsFS, "assets")
