package evidence

func securityKeyConfigured(key [32]byte) bool {
	var aggregate byte
	for _, value := range key {
		aggregate |= value
	}
	return aggregate != 0
}
