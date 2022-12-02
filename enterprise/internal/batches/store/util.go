package store

import (
	"context"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// recordScanner is a callback that knows how to scan the results of the query
// that has been executed into a target object.
type recordScanner[T any] func(target *T, sc dbutil.Scanner) error

// createOrUpdateRecord executes the given query, scans the results back into
// the given record, and returns any error.
func createOrUpdateRecord[T any](ctx context.Context, s *Store, q *sqlf.Query, rs recordScanner[T], record *T) error {
	return s.query(ctx, q, func(sc dbutil.Scanner) error {
		return rs(record, sc)
	})
}

// getRecord returns a single record, if any, from the given query, and return
// ErrNoResults if no record was found.
func getRecord[T any](ctx context.Context, s *Store, q *sqlf.Query, rs recordScanner[T]) (*T, error) {
	var (
		record T
		exists bool
	)

	if err := s.query(ctx, q, func(sc dbutil.Scanner) error {
		exists = true
		return rs(&record, sc)
	}); err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrNoResults
	}

	return &record, nil
}

// listRecords returns a page of records from the given query, using
// CursorResultset internally to construct the return values. Note that the
// query must include the WHERE and LIMIT clauses generated by invoking WhereDB
// and LimitDB on the CursorOpts, respectively.
func listRecords[T any, PT interface {
	// OK, so let's explain the type definition here. T is the concrete record
	// type we're going to be hydrating. PT is, essentially, *T, but _also_ needs to
	// implement Cursor so that CursorResultset works as expected.
	//
	// Practically, that just means that your concrete type (btypes.Whatever) needs
	// to have a Cursor method that receives *btypes.Whatever.
	Cursor
	*T
}](ctx context.Context, s *Store, q *sqlf.Query, opts CursorOpts, rs recordScanner[T]) ([]PT, int64, error) {
	var records []PT
	if opts.Limit != 0 {
		records = make([]PT, 0, opts.Limit)
	} else {
		records = []PT{}
	}

	err := s.query(ctx, q, func(sc dbutil.Scanner) error {
		var record T
		if err := rs(&record, sc); err != nil {
			return err
		}
		records = append(records, &record)
		return nil
	})

	return CursorResultset(opts, records, err)
}