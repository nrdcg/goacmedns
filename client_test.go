package goacmedns

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const updateValue = "idkmybffjill"

var (
	errBody  = []byte(`{"error":"this is a test"}`)
	testAcct = Account{
		FullDomain: "lettuceencrypt.org",
		SubDomain:  "tossed.lettuceencrypt.org",
		Username:   "cpu",
		Password:   "hunter2",
	}
)

func TestClient_RegisterAccount(t *testing.T) {
	testAllowFrom := []string{"space", "earth"}

	testCases := []struct {
		Name            string
		RegisterHandler func(http.ResponseWriter, *http.Request)
		AllowFrom       []string
		ExpectedErr     *ClientError
		ExpectedAccount *Account
	}{
		{
			Name:            "registration failure",
			RegisterHandler: errHandler,
			ExpectedErr: &ClientError{
				HTTPStatus: http.StatusBadRequest,
				Body:       errBody,
				Message:    "response error",
			},
		},
		{
			Name:            "registration success",
			RegisterHandler: newRegHandler(t, nil),
			ExpectedAccount: &testAcct,
		},
		{
			Name:            "registration success, allow from",
			AllowFrom:       testAllowFrom,
			RegisterHandler: newRegHandler(t, testAllowFrom),
			ExpectedAccount: &testAcct,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			client, mux := setupTest(t)
			mux.HandleFunc("/register", tc.RegisterHandler)

			acct, err := client.RegisterAccount(context.Background(), tc.AllowFrom)

			if tc.ExpectedErr == nil && err != nil {
				t.Errorf("expected no error, got %v", err)

				return
			}

			if tc.ExpectedErr != nil && err == nil {
				t.Errorf("expected error %v, got nil", tc.ExpectedErr)

				return
			}

			if tc.ExpectedErr != nil && err != nil {
				var cErr *ClientError
				if ok := errors.As(errors.Unwrap(err), &cErr); !ok {
					t.Fatalf("expected ClientError from RegisterAccount. Got %T", errors.Unwrap(err))
				} else if !reflect.DeepEqual(cErr, tc.ExpectedErr) {
					t.Errorf("got %#v,\n expected err %#v", errors.Unwrap(err), tc.ExpectedErr)
				}

				return
			}

			if tc.ExpectedErr == nil && err == nil {
				// Needed to be able to assert equivalence, as the server addr is dynamic
				tc.ExpectedAccount.ServerURL = acct.ServerURL

				if !reflect.DeepEqual(acct, *tc.ExpectedAccount) {
					t.Errorf("expected account %v, got %v\n", tc.ExpectedAccount, acct)
				}
			}
		})
	}
}

func TestClient_UpdateTXTRecord(t *testing.T) {
	testCases := []struct {
		Name          string
		UpdateHandler func(http.ResponseWriter, *http.Request)
		Value         string
		ExpectedErr   *ClientError
	}{
		{
			Name:          "update failure",
			UpdateHandler: errHandler,
			ExpectedErr: &ClientError{
				HTTPStatus: http.StatusBadRequest,
				Body:       errBody,
				Message:    "response error",
			},
		},
		{
			Name:          "update success",
			UpdateHandler: updateTXTHandler(t),
			ExpectedErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			client, mux := setupTest(t)
			mux.HandleFunc("/update", tc.UpdateHandler)

			err := client.UpdateTXTRecord(context.Background(), testAcct, updateValue)

			switch {
			case tc.ExpectedErr == nil && err != nil:
				t.Errorf("expected no error, got %v", err)

			case tc.ExpectedErr != nil && err == nil:
				t.Errorf("expected error %v, got nil", tc.ExpectedErr)

			case tc.ExpectedErr != nil && err != nil:
				var cErr *ClientError
				if ok := errors.As(errors.Unwrap(err), &cErr); !ok {
					t.Fatalf("expected ClientError from UpdateTXTRecord. Got %v", errors.Unwrap(err))
				} else if !reflect.DeepEqual(cErr, tc.ExpectedErr) {
					t.Errorf("expected err %#v, got %#v\n", tc.ExpectedErr, cErr)
				}
			}
		})
	}
}

func errHandler(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusBadRequest)
	_, _ = resp.Write(errBody)
}

func newRegHandler(t *testing.T, expectedAllowFrom []string) http.HandlerFunc {
	t.Helper()

	return func(resp http.ResponseWriter, req *http.Request) {
		expectedCT := "application/json"
		if ct := req.Header.Get("Content-Type"); ct != expectedCT {
			t.Errorf("expected Content-Type %q got %q", expectedCT, ct)
		}

		if ua := req.Header.Get("User-Agent"); ua != userAgent() {
			t.Errorf("expected User-Agent %q got %q", userAgent(), ua)
		}

		if len(expectedAllowFrom) > 0 {
			decoder := json.NewDecoder(req.Body)

			var regReq Register

			err := decoder.Decode(&regReq)
			if err != nil {
				t.Fatalf("error decoding request body JSON: %v", err)
			}

			if !reflect.DeepEqual(regReq.AllowFrom, expectedAllowFrom) {
				t.Errorf("expected AllowFrom %#v, got %#v", expectedAllowFrom, regReq.AllowFrom)
			}
		}

		resp.WriteHeader(http.StatusCreated)

		newRegBody, _ := json.Marshal(testAcct)
		_, _ = resp.Write(newRegBody)
	}
}

func updateTXTHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(resp http.ResponseWriter, req *http.Request) {
		expectedCT := "application/json"
		if ct := req.Header.Get("Content-Type"); ct != expectedCT {
			t.Errorf("expected Content-Type %q got %q", expectedCT, ct)
		}

		if ua := req.Header.Get("User-Agent"); ua != userAgent() {
			t.Errorf("expected User-Agent %q got %q", userAgent(), ua)
		}

		if key := req.Header.Get("X-Api-Key"); key != testAcct.Password {
			t.Errorf("expected X-Api-Key %q got %q", testAcct.Password, key)
		}

		if user := req.Header.Get("X-Api-User"); user != testAcct.Username {
			t.Errorf("expected X-Api-User %q got %q", testAcct.Username, user)
		}

		decoder := json.NewDecoder(req.Body)

		var updateReq Update

		err := decoder.Decode(&updateReq)
		if err != nil {
			t.Fatalf("error decoding request body JSON: %v", err)
		}

		if updateReq.SubDomain != testAcct.SubDomain {
			t.Errorf("expected update req to have SubDomain %q, had %q",
				testAcct.SubDomain, updateReq.SubDomain)
		}

		if updateReq.Txt != updateValue {
			t.Errorf("expected update req to have Txt %q, had %q",
				updateValue, updateReq.Txt)
		}

		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte(`{}`))
	}
}

func setupTest(t *testing.T) (*Client, *http.ServeMux) {
	t.Helper()

	mux := http.NewServeMux()

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	client, _ := NewClient(ts.URL)

	return client, mux
}
