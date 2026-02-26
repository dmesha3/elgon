package elgon

// Validator validates a value and returns an error if invalid.
type Validator interface {
	Validate(any) error
}
