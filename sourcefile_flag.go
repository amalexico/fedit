package main

import "flag"

// sourceFileFlag is registered at package init so it exists before flag.Parse in main().
var sourceFileFlag = flag.String("sourcefile", "", "The other file to pull -sourceblock content from (insertafter/insertbefore only); -file may then be a comma list or glob")
