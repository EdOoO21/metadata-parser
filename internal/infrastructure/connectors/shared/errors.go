package shared

import "errors"

var ErrNoMatchingFiles = errors.New("files source does not contain matching files for scanner")
