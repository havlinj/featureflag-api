package main

import "testing"

func TestIsGeneratedSourcePath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want bool
	}{
		{"graph/generated.go", true},
		{"graph/model/models_gen.go", true},
		{"graph/foo.go", true},
		{"transport/graphql/flags.resolvers.go", false},
		{"internal/flags/service.go", false},
		{"graph/schema.graphqls", false},
		{"prefix/graph/generated.go", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isGeneratedSourcePath(tc.path); got != tc.want {
			t.Errorf("isGeneratedSourcePath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
