	texthex    := flag.Bool("texthex", false, "Decode -text as hex-encoded UTF-8 (use fwencode to produce)")
	cleanfirst := flag.Bool("cleanfirst", false, "Truncate -file to zero bytes before writing")
	x          := flag.Bool("x", false, "Machine-readable output: bare line numbers / counts, no labels")
		flag.Parse()

	// -texthex: decode -text from a hex string produced by fwencode.
	if *texthex && *text != "" {
		decoded, hErr := hex.DecodeString(*text)
		if hErr != nil {
			fmt.Fprintf(os.Stderr, "Error decoding -texthex: %v\n", hErr)
			os.Exit(1)
		}
		s := string(decoded)
		text = &s
	}

