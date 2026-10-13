package session

import (
	"testing"
	"time"
)

// Fixtures were encoded before moving lifecycle models out of this codec.
// All identities, credentials and key material are synthetic test data.
func TestPreRefactorCookieCompatibility(t *testing.T) {
	const secret = "compatibility-test-only"
	const sessionValue = "f3Ho1z2mknTo4fsLKCTlLlmLxP7YAhnfVPl8-4IoiupuJ9sgFQFcxGDqQOCduBesCmC1bNkSu5XEbR9BLDXNuj0OPpmjj4_5ACSCcj6miruJWze6KX26RMfEzV3bdCMIVreUpTkD6u8dpFmuJ_kCchilNrDKi4ohq7o5PXnxeBgmRnJuurn83v9zdooNXe3PZty3yyVJeRKCkEi_G9UwXXtKkJsDd-gr4h7Fzwef5fUhZQ"
	const flowValue = "OGPw_tvE_Uf9Xhy9b5c10I08QadGqI_mTfK5TN4uz2Ke0iTEzued3kiT66d-ixhh1mXnX1qsHpGQRcObDb9Qv-4iz9dMw6atqKLxbtV8qjY81__eacXP68-_s8RY4bxKVaoQ4m32oC65g-Rr9_XpLgcQ0lo_yC75u2hZt_GwlTra9REoXXqZe3JaS96yQ3DB2GPuG7ZT3s7m8gZzHtpEdceoFmNLL-4RmwS3JvW2yMWFFQ7t6sLxPzz52sXmssjCAo5idWKF0Tby0omt2RfWdEpxj57ZL6zPEhM6mHwPf-231F8ebufYcoNm0GJf1yIU71HOs_HOZZ_o4g"
	issued := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	codec := NewCodec(secret)
	current, err := codec.Decode(sessionValue)
	if err != nil {
		t.Fatal(err)
	}
	if current.ProviderID != "google" || current.AccountID != "synthetic" || current.AccountEmail != "synthetic@example.invalid" || current.Credentials != "synthetic" || !current.IssuedAt.Equal(issued) {
		t.Fatal("session fields changed")
	}
	flow, err := codec.DecodeLoginFlow(flowValue)
	if err != nil {
		t.Fatal(err)
	}
	if flow.ProviderID != "google" || !flow.IssuedAt.Equal(issued) || flow.State != "Kf6pCHjPxKvk7lRkK422ecMe1otJyQYoa-QoHv3HoII" || flow.Verifier != "UfEURiuaIpqFiv1VF6W_CVMY7zzYUXH1OTKZpI0eMlU" || flow.Nonce != "cbY2j57_7dDZAc4xzjVSqdn_fBthEDGrH8ftllLRrBA" {
		t.Fatal("flow fields changed")
	}
}
