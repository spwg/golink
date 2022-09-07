package link

import (
	"context"
	"testing"

	"github.com/spwg/golink/internal/golinktest"
)

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
			name:     "whitespace",
			linkName: "foo foo",
			want:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := validLinkName(tc.linkName)
			if got != tc.want {
				t.Errorf("ValidLinkName(%q) returned %v, want %v", tc.linkName, got, tc.want)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type testCase struct {
		name     string
		linkName string
		address  string
	}
	testCases := []testCase{
		{
			name:     "ok",
			linkName: "foo",
			address:  "http://example.com",
		},
		{
			name:     "dashes ok",
			linkName: "foo-foo",
			address:  "http://example.com",
		},
		{
			name:     "colons ok",
			linkName: "foo:foo",
			address:  "http://example.com",
		},
	}
	ctx := context.Background()
	db := golinktest.NewDatabase(ctx, t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Create(ctx, db, tc.linkName, tc.address)
			if err != nil {
				t.Errorf("Create(%v, %v) returned err=%v, want nil", tc.linkName, tc.address, err)
			}
		})
	}
}
