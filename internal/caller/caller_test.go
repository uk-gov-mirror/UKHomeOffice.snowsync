package caller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/UKHomeOffice/snowsync/internal/client"
	"github.com/tidwall/gjson"
)

//  getMsg gets  test input
func getMsg(p int) string {

	body, err := ioutil.ReadFile("../../test_payloads.json")
	if err != nil {
		return ""
	}

	path := fmt.Sprintf("cases.%v", p)
	res := gjson.GetManyBytes(body, path)

	return res[0].Raw
}

func TestCaller(t *testing.T) {

	tt := []struct {
		name    string
		extID   string
		user    string
		pass    string
		payload string
		err     string
	}{
		{name: "happy", user: "foo", pass: "bar", extID: "inc-123", payload: getMsg(0)},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			os.Setenv("SNOW_USER", tc.user)
			os.Setenv("SNOW_PASS", tc.pass)

			e := client.Envelope{
				MsgID:   "CUSTOM MESSAGE",
				ExtID:   tc.extID,
				Payload: tc.payload,
			}

			testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				ct := r.Header.Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("wrong content type: %v", ct)
				}

				sa := r.Header.Get("Authorization")
				if sa != "Basic Zm9vOmJhcg==" {
					t.Errorf("wrong auth header: %v", sa)
				}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("could not read request body: %v", sa)
				}

				env, err := json.Marshal(e)
				if err != nil {
					t.Fatalf("could not marshal comparator")
				}

				if string(body) != string(env) {
					t.Errorf("expected %v, got %v", string(env), string(body))
				}

				// respond with an identifier
				type result struct {
					InternalIdentifier string `json:"internal_identifier,omitempty"`
				}
				type resp struct {
					result `json:"result,omitempty"`
				}

				response := resp{
					result{
						InternalIdentifier: "inc-123",
					},
				}

				w.Header().Set("Content-Type", "application/json")
				bytes, _ := json.Marshal(response)
				w.Write(bytes)

			}))

			os.Setenv("SNOW_URL", testSrv.URL)
			_, err := Call(e)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

		})
	}
}
