package parser

import "errors"

// ErrFormatNotRecognized is returned when no parser accepts a file.
var ErrFormatNotRecognized = errors.New("no parser recognizes this file format")
