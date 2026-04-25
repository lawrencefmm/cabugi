package auth

import (
	"os"
	"strings"
)

const localTestAuthIssuer = "https://cabugi.local.test"

const localTestAuthAudience = "cabugi-local-test"

const localTestAuthPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx6dXtLfQmM2j0iJbvfpQ
WZWVCXVcGSESpbP1+vFP9Z5hg+fEkwXRFmTQzg2bSfylONEyFWutPmhq3TCoCqVC
vVQBkF2fUL7ro7ChvfALguas7dSp43tFDwQISKbL5OP8UrAlD4I07UV29oieiwRb
3qyLRK8YJsJTzMsODrNWGN5RRo/tsPtxe9vza4rrSfJ7NB0NPUUWbk0yCLFXdtBx
/up86pYWD1BtEUOCru5ySfzV8nYL4PHcoF7RYZ4Xv0CnxuekGEqpC9ooqHHZRc86
M1KCccrEySxcgTnWXBrA3eleJf3WsWR9ZKvr12waiinY/wT5qYMcIbLMsLAb920J
vQIDAQAB
-----END PUBLIC KEY-----`

func localTestAuthEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("LOCAL_TEST_AUTH_ENABLED")), "true")
}
