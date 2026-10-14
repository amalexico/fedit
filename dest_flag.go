package main

import "flag"

// destFlag is registered at package init so it exists before flag.Parse in main().
var destFlag = flag.String("dest", "", "Destination file(s), comma-separated, for copy, move and replace across files; replace needs -destblock, copy and move each need one of -after, -before, -aftermatch, -beforematch, -afterblock, -beforeblock")
