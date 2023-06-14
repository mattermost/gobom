package dt

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	const v = "1.2.3"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/version" {
			t.Errorf("received a request to an unexpected path: '%s'", req.URL.Path)
		}
		if req.Method != http.MethodGet {
			t.Errorf("unexpected request method '%s'", req.Method)
		}
		if req.Header.Get("X-Api-Key") != "" {
			t.Errorf("API key not expected in an unauthenticated API")
		}

		fmt.Fprintf(w, `{"version":"%s"}`, v)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	version, err := client.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != v {
		t.Errorf("unexpected version '%s'", version)
	}
}

func TestUpload(t *testing.T) {
	const (
		secret  = "secret"
		bom     = "bom contents"
		project = "someproject"
		version = "1.2.3"
		token   = "token"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/bom" {
			t.Errorf("received a request to an unexpected path: '%s'", req.URL.Path)
		}
		if req.Method != http.MethodPost {
			t.Errorf("unexpected request method '%s'", req.Method)
		}
		if req.Header.Get("X-Api-Key") != secret {
			t.Errorf("no valid API key in request")
		}

		err := req.ParseMultipartForm(int64(len(bom)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		file, _, _ := req.FormFile("bom")
		uploaded, _ := io.ReadAll(file)

		if string(uploaded) != bom {
			t.Errorf("unexpected bom contents: '%s'", string(uploaded))
		}

		fmt.Fprintf(w, `{"token":"%s"}`, token)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := client.Upload(strings.NewReader(bom), project, version, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != token {
		t.Errorf("unexpected token value '%s'", result)
	}
}

func TestLookup(t *testing.T) {
	const (
		secret  = "secret"
		project = "someproject"
		version = "1.2.3"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/project/lookup" {
			t.Errorf("received a request to an unexpected path: '%s'", req.URL.Path)
		}
		if req.Method != http.MethodGet {
			t.Errorf("unexpected request method '%s'", req.Method)
		}
		if req.Header.Get("X-Api-Key") != secret {
			t.Errorf("no valid API key in request")
		}

		query := req.URL.Query()
		if query.Get("name") == project && query.Get("version") == version {
			fmt.Fprintf(w, `{"name":"%s","version":"%s"}`, project, version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, err := client.Lookup(project, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != project {
		t.Errorf("unexpected project name '%s'", p.Name)
	}
	if p.Version != version {
		t.Errorf("unexpected project version '%s'", p.Version)
	}

	p, err = client.Lookup("otherproject", version)
	if err == nil {
		t.Error("expected an error, saw nil")
	}
	if p != nil {
		t.Errorf("expected nil project, saw %v", p)
	}
}

func TestGetProject(t *testing.T) {
	const (
		secret  = "secret"
		project = "someproject"
		version = "1.2.3"
		uuid    = "uuid"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != fmt.Sprintf("/api/v1/project/%s", uuid) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if req.Method != http.MethodGet {
			t.Errorf("unexpected request method '%s'", req.Method)
		}
		if req.Header.Get("X-Api-Key") != secret {
			t.Errorf("no valid API key in request")
		}

		fmt.Fprintf(w, `{"name":"%s","version":"%s"}`, project, version)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p, err := client.GetProject(uuid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != project {
		t.Errorf("unexpected project name '%s'", p.Name)
	}
	if p.Version != version {
		t.Errorf("unexpected project version '%s'", p.Version)
	}

	p, err = client.GetProject("other-uuid")
	if err == nil {
		t.Error("expected an error, saw nil")
	}
	if p != nil {
		t.Errorf("expected nil project, saw %v", p)
	}
}
