package aigateway

// GatewaySecurityFacts returns the compact, non-content detector facts derived
// from a request. The returned facts contain only bounded classifications and
// booleans; raw request text is never included.
func GatewaySecurityFacts(request Request) []Fact {
	facts := gatewaySecurityFacts(request)
	return append([]Fact(nil), facts...)
}
