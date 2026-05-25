	// -cleanfirst: truncate before reading so mutations start from an empty file.
	if *cleanfirst && *file != "" {
		if err := os.WriteFile(*file, []byte{}, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error truncating file: %v\n", err)
			os.Exit(1)
		}
	}

