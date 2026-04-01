package web

import "io/fs"

// ClientFS is nil — the viewer frontend is not bundled in this build.
// It will be rebuilt separately for v2.
var ClientFS fs.FS
