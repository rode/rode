package harborfail

pass {
	count(violation_count) == 0
}

violation_count[v] {
	violations[v].pass == false
}

#######################################################################################
note_name_dne[o] {
	m := input.occurrences

	o = count({x | m[x]; f(m[x])})
}

f(x) {
	x.note_name == "not this"
}

# No occurrence should be missing a note name
violations[result] {
	result = {
		"pass": note_name_dne[i] == 0,
		"id": "note_names_exist",
		"name": "Occurrences containing note names",
		"description": "Verify that all occurrences contain a note name",
		"message": sprintf("found %v occurrences with missing note names", [note_name_dne[i]]),
	}
}

###################################################################################
uses_gcr[o] {
	m := input.occurrences

	#trace(sprintf("this %v",[count({x | m[x]; f(m[x]) })]))
	o = count({x | m[x]; g(m[x])})
}

g(x) {
	contains(x.resource.uri, "gcr.io")
}

# All occurrences should have a resource uri containing a gcr endpoint
violations[result] {
	result = {
		"pass": uses_gcr[i] == 0,
		"id": "uses_gcr",
		"name": "Occurrences use GCR URIs",
		"description": "Verify that all occurrences contain a resource uri from gcr",
		"message": sprintf("found %v occurrences with non gcr resource uris", [uses_gcr[i]]),
	}
}
