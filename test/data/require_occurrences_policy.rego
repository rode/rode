package require_occurrences

pass {
    count(input.occurrences) > 0
}

violations[result] {
    occurrences_count := count(input.occurrences)
	result = {
		"pass": occurrences_count > 0,
		"id": "require_occurrences",
		"name": "Require Occurrences",
		"description": "Require at least one occurrence",
		"message": sprintf("Found %v occurrences", [occurrences_count]),
	}
}
