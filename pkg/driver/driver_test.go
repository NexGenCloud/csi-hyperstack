package driver

import (
	"fmt"
	"strings"
	"testing"
)

// The API key must never reach a log sink through any formatting verb.
// See the CreateVolume credential-disclosure report (CWE-532).
func TestDriverOptsNeverFormatsApiKey(t *testing.T) {
	const apiKey = "hs-DUMMY-KEY-0000-POC"

	opts := DriverOpts{
		Endpoint:             "unix:///tmp/csi.sock",
		HyperstackApiKey:     apiKey,
		HyperstackApiAddress: "https://infrahub-api.nexgencloud.com/v1",
	}

	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		for name, arg := range map[string]any{"value": opts, "pointer": &opts} {
			out := fmt.Sprintf(verb, arg)
			if strings.Contains(out, apiKey) {
				t.Errorf("%s of %s leaked the API key: %s", verb, name, out)
			}
			if !strings.Contains(out, redactedApiKey) {
				t.Errorf("%s of %s is missing the redaction marker: %s", verb, name, out)
			}
			if !strings.Contains(out, opts.HyperstackApiAddress) {
				t.Errorf("%s of %s dropped HyperstackApiAddress: %s", verb, name, out)
			}
		}
	}
}
