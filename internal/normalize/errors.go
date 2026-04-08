package normalize

import "fmt"

type emptyValueError struct {
	field string
}

func (e emptyValueError) Error() string {
	return fmt.Sprintf("%s cannot be empty", e.field)
}

func ErrEmptyValue(field string) error {
	return emptyValueError{field: field}
}
