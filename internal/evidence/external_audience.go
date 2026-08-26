package evidence

// ExternalAudienceMatches reports whether audience matches the protected
// recipient hash already stored for an external Evidence Request.
func ExternalAudienceMatches(request Request, audience string) bool {
	return externalAudienceMatches(request, audience)
}
