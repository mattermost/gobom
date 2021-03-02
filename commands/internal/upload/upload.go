package upload

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattermost/gobom/log"
)

var (
	api     string
	key     string
	project string
)

// Command .
var Command = &cobra.Command{
	Use:   "upload [flags] [filename]",
	Short: "upload a BOM file to Dependency-Track",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Debug("uploading from stdin")
			Upload(os.Stdin)
		} else {
			log.Debug("uploading from '%s'", args[0])
			file, err := os.Open(args[0])
			if err != nil {
				log.Error("error opening BOM file: %v", err)
				return
			}
			Upload(file)
		}
	},
}

// Upload .
func Upload(file io.Reader) {
	bomURL, err := url.Parse(api)
	if err != nil {
		log.Error("bad url: %v", err)
		return
	} else if bomURL.Scheme == "" {
		log.Error("bad url: no scheme specified")
		return
	}
	bomURL.Path = path.Join(bomURL.Path, "api/v1/bom")
	log.Debug("using BOM upload URL '%s'", bomURL.String())

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if strings.Contains(project, "@") {
		project := strings.SplitN(project, "@", 2)
		writer.WriteField("projectName", project[0])
		writer.WriteField("projectVersion", project[1])
	} else {
		writer.WriteField("project", project)
	}
	writer.WriteField("autoCreate", "true")

	bom, _ := writer.CreateFormFile("bom", "bom.xml")
	io.Copy(bom, file)

	writer.Close()

	request, err := http.NewRequest(http.MethodPost, bomURL.String(), body)
	if err != nil {
		log.Error("error uploading BOM: %v", err)
		return
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Api-Key", key)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Error("error uploading BOM: %v", err)
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("error reading response from server")
		return
	}
	if response.StatusCode > 299 {
		log.Error("error response from server: %s\n%s", response.Status, string(result))
		return
	}
	log.Info("BOM successfully uploaded")
}

func init() {
	Command.Flags().StringVarP(&api, "url", "u", "", "Dependency-Track API server URL")
	Command.Flags().StringVarP(&key, "key", "k", "", "Dependency-Track API key")
	Command.Flags().StringVarP(&project, "project", "P", "", "Dependency-Track project in the form of an UUID or 'name@version'")
}
