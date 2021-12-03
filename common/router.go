package common

import "strings"

type StaticSiteConfig struct {
	Path  string
	Index string
	Error string
	ETag string
}

func Route(hostname string) StaticSiteConfig {
	hostname = strings.Split(hostname, ":")[0]
	hostname = strings.ToLower(hostname)
	routerMap := map[string]StaticSiteConfig{
		"bar.demo.com": {
			Path:  "sitebar",
			Index: "index.html",
			Error: "error.html",
			ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4",
		},
		"foo.demo.com": {
			Path:  "sitefoo",
			Index: "home.html",
			Error: "404.html",
			ETag: "33a64df551425fcc55e4d42a148795d9f25ffoo4",
		},
		"foobar.demo.com": {
			Path:  "sitefoo",
			Index: "index.html",
			Error: "error.html",
			ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4",
		},
		"engineering.demo.com": {
			Path:  "engineering",
			Index: "index.html",
			Error: "404.html",
			ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4",
		},
		"cloud.demo.com": {
			Path:  "cloud",
			Index: "index.html",
			Error: "404.html",
			ETag: "33a64df551425fcc55e4d42a148795d9f25f89d4",
		},
	}
	return routerMap[hostname]
}