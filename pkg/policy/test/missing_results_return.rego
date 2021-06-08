package harborfail

pass {
	3 == 3
}

violations {
	a := {
		"pass": true,
		"name": "Occurrences containing note names",
		"description": "Verify that all occurrences contain a note name",
		"message": sprintf("found %v occurrences with missing note names", ["hi"]),
	}
}
