package main

import "flag"

// destBlockFlag is registered at package init so it exists before flag.Parse in main().
var destBlockFlag = flag.String("destblock", "", "Block name in each -dest file to overwrite (replace -dest only)")
