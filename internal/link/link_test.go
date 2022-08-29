package link

import "testing"

func TestValidLinkName(t *testing.T) {
	type testCase struct {
		name     string
		linkName string
		want     bool
	}
	testCases := []testCase{
		{
			name:     "ok",
			linkName: "foo",
			want:     true,
		},
		{
			name:     "bad char",
			linkName: "<alert>console.log('here')</alert>",
			want:     false,
		},
		{
			name:     "whitespace",
			linkName: "foo foo",
			want:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidLinkName(tc.linkName)
			if got != tc.want {
				t.Errorf("ValidLinkName(%q) returned %v, want %v", tc.linkName, got, tc.want)
			}
		})
	}
}
