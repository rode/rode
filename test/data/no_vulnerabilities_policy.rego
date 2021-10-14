package no_vulns

pass {
    count(input.occurrences) > 0
    count(vulnerabilities) == 0
}

vulnerabilities[v] {
	input.occurrences[i].kind == "VULNERABILITY"
    v := input.occurrences[i]
}

violations[result] {
	result = {
		"pass": true,
		"id": "valid",
		"name": "name",
		"description": "description",
		"message": "message",
	}
}
