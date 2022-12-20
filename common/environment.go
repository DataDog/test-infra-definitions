package common

type Environment interface {
	DefaultInstanceType() string
	DefaultARMInstanceType() string
}
