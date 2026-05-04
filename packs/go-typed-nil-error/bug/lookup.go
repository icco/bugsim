package db

import "fmt"

// DBError is the package's error type.
type DBError struct {
	Code int
}

func (e *DBError) Error() string {
	return fmt.Sprintf("db error %d", e.Code)
}

// Lookup returns a *DBError on bad input, or "nothing" otherwise.
func Lookup(id int) error {
	var dberr *DBError
	if id < 0 {
		dberr = &DBError{Code: 400}
	}
	return dberr
}
