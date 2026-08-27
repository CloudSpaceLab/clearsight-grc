package evidence

import "bytes"

// otpFailedAttemptMutation identifies the only challenge update that may safely
// commute with stale readers: recording one failed verification attempt. The
// store owns the increment so concurrent guesses cannot evade the attempt cap.
func otpFailedAttemptMutation(challenge OTPChallenge, expectedAttempts, expectedResends int, expectedDigest []byte) bool {
	return challenge.Attempts == expectedAttempts+1 &&
		challenge.Resends == expectedResends &&
		bytes.Equal(challenge.Digest, expectedDigest) &&
		challenge.ConsumedAt == nil
}
