package provider

import "net/url"

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func makeQuery() url.Values {
	return url.Values{}
}
