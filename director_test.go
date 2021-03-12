package main

import (
	"testing"
)

func TestDirector(t *testing.T) {
	paths := map[string]string{
		"git.andyleap.dev":        "git.git:8080",
		"git.andyleap.dev/about":  "about.git:8080",
		"blog.andyleap.dev":       "blog.blog:8080",
		"blog.andyleap.dev/about": "about.blog:8080",
	}
	tests := []struct {
		url      string
		endpoint string
		eppath   string
	}{
		{
			"git.andyleap.dev",
			"git.git:8080",
			"",
		},
		{
			"git.andyleap.dev/about",
			"about.git:8080",
			"",
		},
		{
			"git.andyleap.dev/test",
			"git.git:8080",
			"test",
		},
		{
			"blog.andyleap.dev/test",
			"blog.blog:8080",
			"test",
		},
		{
			"blog.andyleap.dev/about",
			"about.blog:8080",
			"",
		},
		{
			"blag.andyleap.dev/",
			"",
			"",
		},
	}
	for _, test := range tests {
		endpoint, eppath := Direct(paths, test.url)
		if endpoint != test.endpoint {
			t.Errorf("Got endpoint %s, expected %s", endpoint, test.endpoint)
		}
		if eppath != test.eppath {
			t.Errorf("Got eppath %s, expected %s", eppath, test.eppath)
		}
	}
}
