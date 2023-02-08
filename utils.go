package main

// Filter empty strings from slice of strings
func filterBlanks(slice []string) []string {
	var filtered = []string{}
	for _, t := range slice {
		if t != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// Utility function for error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}
