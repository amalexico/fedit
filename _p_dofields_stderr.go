	if !x {
		fmt.Fprintf(os.Stderr, "Extracted col %d from %d/%d lines (delim=%q)\n",
			col, extracted, lineNum, delim)
	}
