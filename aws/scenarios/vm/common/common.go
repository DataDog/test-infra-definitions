package common

func ApplyOption[T any](instance *T, option func(*T) error, options []func(*T) error) (*T, error) {
	options = append([]func(*T) error{option}, options...)
	for _, o := range options {
		if err := o(instance); err != nil {
			return nil, err
		}
	}
	return instance, nil
}
