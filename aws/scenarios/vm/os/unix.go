package os

type unix struct{}

func (unix) Visit(v Visitor)                     { v.VisitUnix() }
func (unix) GetAMIArch(arch Architecture) string { return string(arch) }
func (unix) GetDefaultInstanceType(arch Architecture) string {
	return getDefaultInstanceType(arch)
}
func (unix) GetTenancy() string    { return "default" }
func (unix) GetConfigPath() string { return "/etc/datadog-agent/datadog.yaml" }

func getDefaultInstanceType(arch Architecture) string {
	switch arch {
	case AMD64Arch:
		return "t3.large"
	case ARM64Arch:
		return "m6g.medium"
	default:
		panic("Architecture not supportede")
	}
}
