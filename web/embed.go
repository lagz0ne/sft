package web

import "io/fs"

// ClientFS is nil in source-only builds where the browser frontend has not been
// embedded yet. `sft view` must report that limitation explicitly instead of
// pretending the browser surface is available.
var ClientFS fs.FS
