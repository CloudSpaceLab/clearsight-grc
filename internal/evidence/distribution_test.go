package evidence

import (
	"reflect"
	"testing"
)

func TestDistributionRecipientProjectionDoesNotExposeProtectedAddressMaterial(t *testing.T) {
	typeOf := reflect.TypeOf(DistributionRecipient{})
	for _, forbidden := range []string{"Address", "AddressHash", "AddressCiphertext", "AddressKeyID"} {
		if _, ok := typeOf.FieldByName(forbidden); ok {
			t.Fatalf("safe recipient projection must not expose %s", forbidden)
		}
	}
}

func TestDistributionContractUsesExplicitRolesPoliciesAndAssurance(t *testing.T) {
	if RecipientTo != "TO" || RecipientCC != "CC" {
		t.Fatalf("unexpected recipient roles: %q %q", RecipientTo, RecipientCC)
	}
	if AccessDirectMagicLink != "DIRECT_MAGIC_LINK" || AccessSharedEmailOTP != "SHARED_LINK_EMAIL_OTP" || AccessDirectEmailOTP != "DIRECT_LINK_EMAIL_OTP" {
		t.Fatal("distribution access policies changed")
	}
	if AssuranceLinkPossession != "LINK_POSSESSION" || AssuranceEmailVerified != "EMAIL_VERIFIED" {
		t.Fatal("distribution assurance levels changed")
	}
}
