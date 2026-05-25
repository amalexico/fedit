	if *op == "write" || *op == "writeraw" || *op == "writelines" {
		var content []string
		switch *op {
		case "writelines":
			scanner := bufio.NewScanner(os.Stdin)
			for {
				fmt.Fprint(os.Stderr, "> ")
				if !scanner.Scan() {
					break
				}
				content = append(content, scanner.Text())
			}
		case "writeraw":
			if *textFile != "" {
				var err error
				content, err = readLines(*textFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
					os.Exit(1)
				}
			} else {
				// raw: split on actual newlines only -- no \n->newline expansion
				content = strings.Split(*text, "\n")
			}
		default: // "write"
			content = expandText(*text)
			if *textFile != "" {
				var err error
				content, err = readLines(*textFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
					os.Exit(1)
				}
			}
		}
		if len(content) == 0 {
			fmt.Fprintln(os.Stderr, "Nothing to write")
			os.Exit(1)
		}
		if err := writeLines(*file, content); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d line(s) to %s\n", len(content), *file)
		return
	}
