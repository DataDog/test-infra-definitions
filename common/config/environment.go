package config

type Environment interface {
	Region() string
	VPCID() string
}
