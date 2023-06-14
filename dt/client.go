package dt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
)

// Client provides methods for communicating with a Dependency-Track API server
type Client struct {
	*http.Client

	baseURL *url.URL
	secret  string
}

// NewClient initializes a new Dependency-Track API client
func NewClient(api, secret string) (*Client, error) {
	client := &Client{
		Client: http.DefaultClient,
		secret: secret,
	}

	parsedURL, err := url.Parse(api)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" {
		return nil, errors.New("cannot use relative URL as the Dependency-Track API location")
	} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URI scheme '%s'", parsedURL.Scheme)
	}
	parsedURL.Fragment = ""
	parsedURL.RawQuery = ""

	client.baseURL = parsedURL

	return client, nil
}

// Upload posts a BOM file to a project
func (c *Client) Upload(file io.Reader, project, version, uuid string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if len(project) > 0 {
		err := writer.WriteField("projectName", project)
		if err != nil {
			return "", err
		}
	}

	if len(version) > 0 {
		err := writer.WriteField("projectVersion", version)
		if err != nil {
			return "", err
		}
	}

	if len(uuid) > 0 {
		err := writer.WriteField("project", uuid)
		if err != nil {
			return "", err
		}
	}

	err := writer.WriteField("autoCreate", "true")
	if err != nil {
		return "", err
	}

	bom, _ := writer.CreateFormFile("bom", "bom.xml")
	_, err = io.Copy(bom, file)
	if err != nil {
		return "", err
	}

	writer.Close()

	request, err := http.NewRequest(http.MethodPost, c.url("api/v1/bom"), body)
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Api-Key", c.secret)
	response, err := c.Do(request)
	if err != nil {
		return "", err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode > 299 {
		return "", fmt.Errorf("error response from server: %s -- %s", response.Status, string(result))
	}

	token := &struct {
		Token string
	}{}
	err = json.Unmarshal(result, token)
	if err != nil {
		return "", err
	}

	return token.Token, nil
}

// Version fetches version information from the server
func (c *Client) Version() (string, error) {
	response, err := c.Get(c.url("api/version"))
	if err != nil {
		return "", err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	version := &struct {
		Version string
	}{}
	err = json.Unmarshal(result, version)
	if err != nil {
		return "", err
	}

	return version.Version, nil
}

// Project represents a project on the Dependency-Track server
type Project struct {
	Name          string
	Version       string
	LastBomImport int64
}

// Lookup returns information about a named project
func (c *Client) Lookup(project, version string) (*Project, error) {
	values := url.Values{
		"name":    []string{project},
		"version": []string{version},
	}

	request, err := http.NewRequest(http.MethodGet,
		c.url("api/v1/project/lookup")+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("X-Api-Key", c.secret)
	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode > 299 {
		return nil, fmt.Errorf("error response from server: %s -- %s", response.Status, string(result))
	}

	p := &Project{}
	err = json.Unmarshal(result, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// GetProject returns information about a project by UUID
func (c *Client) GetProject(uuid string) (*Project, error) {
	request, err := http.NewRequest(http.MethodGet,
		c.url(fmt.Sprintf("api/v1/project/%s", uuid)), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("X-Api-Key", c.secret)
	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	p := &Project{}
	err = json.Unmarshal(result, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *Client) url(target string) string {
	result := &url.URL{}
	*result = *c.baseURL
	result.Path = path.Join(result.Path, target)

	return result.String()
}
