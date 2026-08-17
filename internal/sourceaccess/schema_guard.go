package sourceaccess

func validateExpectedSchema(view View, observed string) error {
	if view.SchemaFingerprint != "" && observed != "" && view.SchemaFingerprint != observed {
		return ErrSchemaDrift
	}
	return nil
}
