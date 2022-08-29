// Package link provides functionality to manage go links.
package link

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

var (
	// ErrAlreadyExists means that a link name already exists in the database.
	ErrAlreadyExists = errors.New("already exists")
	// ErrInvalidLinkName means that a link name is not valid. See blockChars.
	ErrInvalidLinkName = errors.New("invalid name")
	// ErrNotFound means that the name was not found.
	ErrNotFound = errors.New("not found")
	// ErrInvalidAddress means that the address was not a parseable URL.
	ErrUnparseableAddress = errors.New("unparsable")
)

const BlockChars = "/<>"

// Record is an entry in the database for a name and an address to redirect to.
type Record struct {
	// Name is the name of the go link.
	Name string
	// Link is the address to redirect to.
	Link *url.URL
}

// Create inserts a new record into the database for name and address.
func Create(ctx context.Context, db *sql.DB, name, address string) error {
	if !ValidLinkName(name) {
		return ErrInvalidLinkName
	}
	u, err := url.Parse(address)
	if err != nil {
		return ErrUnparseableAddress
	}
	_, ok, err := linkByName(ctx, db, name)
	if err != nil {
		return err
	}
	if ok {
		return ErrAlreadyExists
	}
	query := "insert into links (name, url) values (?, ?);"
	if _, err := db.ExecContext(ctx, query, name, u.String()); err != nil {
		return fmt.Errorf("failed to create new record in the database: %w", err)
	}
	return nil
}

// Read returns a *Record for the link with the given name.
// Returns ErrNotFound when there's no corresponding record.
func Read(ctx context.Context, db *sql.DB, name string) (*Record, error) {
	if !ValidLinkName(name) {
		return nil, ErrInvalidLinkName
	}
	r, found, err := linkByName(ctx, db, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read link %q: %w", name, err)
	}
	if !found {
		return nil, ErrNotFound
	}
	return r, nil
}

// Update changes the record for oldName so that it's name is newName and the
// url it redirects to is address.
func Update(ctx context.Context, db *sql.DB, oldName, newName, address string) error {
	if !ValidLinkName(newName) {
		return ErrInvalidLinkName
	}
	_, err := url.Parse(address)
	if err != nil {
		return ErrUnparseableAddress
	}
	_, found, err := linkByName(ctx, db, oldName)
	if err != nil {
		return fmt.Errorf("failed to query the database for the old name: %w", err)
	}
	if !found {
		return ErrNotFound
	}
	// There is a race here between checking that the new name doesn't exist the
	// update, but the checks are really just for writing nicer messages for the
	// user. The database will enforce that names are unique as a constraint.
	_, found, err = linkByName(ctx, db, newName)
	if err != nil {
		return fmt.Errorf("failed to query the database for the new name: %w", err)
	}
	if found {
		return ErrAlreadyExists
	}
	const query = "update links set name = ?, url = ? where name = ?;"
	if _, err := db.ExecContext(ctx, query, newName, address, oldName); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	return nil
}

// Delete removes an entry from the database.
func Delete(ctx context.Context, db *sql.DB, name string) error {
	_, found, err := linkByName(ctx, db, name)
	if err != nil {
		return fmt.Errorf("failed to delete %q: %w", name, err)
	}
	if !found {
		return ErrNotFound
	}
	const query = "delete from links where name=?;"
	if _, err := db.ExecContext(ctx, query, name); err != nil {
		return fmt.Errorf("failed to execute delete statement: %w", err)
	}
	return nil
}

func linkByName(ctx context.Context, db *sql.DB, name string) (*Record, bool, error) {
	const query = "select (url) from links where name=?;"
	row := db.QueryRowContext(ctx, query, name)
	var link string
	if err := row.Scan(&link); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, false, fmt.Errorf("failed to lookup %q: %w", name, err)
	}
	return &Record{name, u}, true, nil
}

// ValidLinkName returns true if name is valid and false otherwise.
//
// A name is invalid if it contains whitespace, contains any of BlockChars, or is the empty string.
func ValidLinkName(name string) bool {
	for _, c := range name {
		if unicode.IsSpace(c) {
			return false
		}
	}
	return name != "" && !strings.ContainsAny(name, BlockChars)
}
