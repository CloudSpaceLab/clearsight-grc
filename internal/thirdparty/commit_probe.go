package thirdparty

import "errors"

func exactCommitResult(commitErr error, confirmed bool, probeErr error) error {
	if confirmed {
		return nil
	}
	if probeErr != nil {
		return errors.Join(commitErr, probeErr)
	}
	return commitErr
}
