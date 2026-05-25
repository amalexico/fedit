// doStreamFind is the streaming path for find (-stream). Outputs to stdout.
func doStreamFind(path, search string, x bool) {
	src, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	count, lineNum := 0, 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(line, search) {
			count++
			if x {
				fmt.Println(lineNum)
			} else {
				fmt.Printf("%d: %s\n", lineNum, line)
			}
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %v\n", scanner.Err())
		os.Exit(1)
	}
	if !x {
		fmt.Fprintf(os.Stderr, "Found %d match(es) across %d lines\n", count, lineNum)
	}
}
