package main

import "flag"

// sourceBlockFlag is registered at package init so it exists before flag.Parse in main().
var sourceBlockFlag = flag.String("sourceblock", "", "Block name in -sourcefile to use as content (requires -lang; insertafter/insertbefore only)")
