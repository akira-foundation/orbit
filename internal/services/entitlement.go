package services

type Entitlements interface {
	Has(feature string) bool
}

type allowAll struct{}

func (allowAll) Has(string) bool { return true }

func AllowAll() Entitlements { return allowAll{} }

const FeatureServices = "services"
