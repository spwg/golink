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
		wantErr  bool
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
		{
			name:     "escaped name invalid",
			linkName: "<alert>console.log('hello');</alert>",
			address:  "http://example.com",
			wantErr:  true,
		},
	}
	ctx := context.Background()
	db := golinktest.NewDatabase(ctx, t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Create(ctx, db, tc.linkName, tc.address)
			if !tc.wantErr && err != nil {
				t.Errorf("Create(%v, %v) returned err=%v, want nil", tc.linkName, tc.address, err)
			}
			if tc.wantErr && err == nil {
				t.Errorf("Create(%v, %v) returned err=nil, want not nil", tc.linkName, tc.address)
			}
		})
	}
}
