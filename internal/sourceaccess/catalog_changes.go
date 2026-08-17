package sourceaccess

import "context"

func (s *CatalogService) CaptureBindingChange(ctx context.Context, tenantID, bindingID string, version int64, event ChangeEvent) (ChangeCaptureResult, error) {
	if err := event.ValidateInput(); err != nil {
		return ChangeCaptureResult{}, err
	}
	bindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	if !statefulBindingRevision(bindingRevision) || !bindingRevisionAllows(bindingRevision, OperationChanges) {
		return ChangeCaptureResult{}, ErrCatalogInvalid
	}
	viewRevision, err := s.repoOrError().ViewRevision(ctx, tenantID, bindingRevision.ViewID, bindingRevision.ViewVersion)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	connectionRevision, err := s.repoOrError().ConnectionRevision(ctx, tenantID, viewRevision.ConnectionID, viewRevision.ConnectionVersion)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	connection, view, adapter, err := s.executionContracts(connectionRevision, viewRevision)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	binding, err := bindingRevision.Contract(viewRevision)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	session, err := adapter.Open(operationCtx, connection, s.secrets)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	defer session.Close()
	if !session.Capabilities().Has(CapabilityChanges) {
		return ChangeCaptureResult{}, ErrCapabilityUnavailable
	}
	receiver, ok := session.(ChangeReceiver)
	if !ok {
		return ChangeCaptureResult{}, ErrCapabilityUnavailable
	}
	result, err := receiver.CaptureChange(operationCtx, view, binding, event)
	if err != nil {
		return ChangeCaptureResult{}, err
	}
	if err := validateCatalogReceipt(result.Receipt, connection, view, binding, OperationChanges, 1); err != nil {
		return ChangeCaptureResult{}, err
	}
	if err := validateExpectedSchema(view, result.Receipt.SchemaFingerprint); err != nil {
		return ChangeCaptureResult{}, err
	}
	if result.Receipt.Position == nil || result.Receipt.Bytes > limits.ResponseBytes {
		return ChangeCaptureResult{}, ErrLimitExceeded
	}
	if !result.Accepted {
		return ChangeCaptureResult{}, ErrExecution
	}
	return result, nil
}

func bindingRevisionAllows(binding BindingRevision, operation Operation) bool {
	for _, candidate := range binding.Operations {
		if candidate == operation {
			return true
		}
	}
	return false
}
