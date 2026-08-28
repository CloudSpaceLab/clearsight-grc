package evidence

func validAccessAssurance(value AccessAssurance) bool {
	return value == AssuranceLinkPossession || value == AssuranceEmailVerified
}
