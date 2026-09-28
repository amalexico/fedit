package main

import "flag"

// destFlag is registered at package init so it exists before flag.Parse in main().
var destFlag = flag.String("dest", "", "Destination file(s), comma-separated, for copy/move across files; each needs one of -after, -before, -aftermatch, -beforematch, -afterblock, -beforeblock")
