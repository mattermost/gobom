package upload

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattermost/gobom/dt"
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
	client, err := dt.NewClient(api, key)
	if err != nil {
		log.Error("%v", err)
		return
	}

	// the version check is unauthenticated; if it fails, the user probably
	// typoed the URL, but because we fail there, the API token doesn't get
	// sent to the wrong server
	version, err := client.Version()
	if err != nil {
		log.Error("failed to check server version: %s", err)
		return
	}
	log.Debug("server version is %s", version)

	p1, err := getProject(client, project)
	if err != nil {
		log.Error("project lookup failed: %v", err)
		return
	}
	log.Debug("last BOM import: %s", time.Unix(p1.LastBomImport/1000, 0).Format(time.Stamp))
	token, err := uploadBOM(client, file, project)
	if err != nil {
		log.Error("%v", err)
		return
	}
	log.Debug("received upload token: '%s'", token)

	// did the project already get updated?
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		log.Debug("polling project status")
		p2, err := getProject(client, project)
		if err != nil {
			log.Error("project lookup failed: %v", err)
			return
		}
		if p2 != nil && (p1 == nil || p1.LastBomImport != p2.LastBomImport) {
			log.Info("BOM successfully uploaded on %s", time.Unix(p2.LastBomImport/1000, 0).Format(time.Stamp))
			return
		}
	}

	log.Error("upload was successful but project was not updated; check server logs for more info")
}

func uploadBOM(client *dt.Client, file io.Reader, project string) (string, error) {
	if strings.Contains(project, "@") {
		project := strings.SplitN(project, "@", 2)
		return client.Upload(file, project[0], project[1], "")
	}
	return client.Upload(file, "", "", project)
}

func getProject(client *dt.Client, project string) (*dt.Project, error) {
	if strings.Contains(project, "@") {
		project := strings.SplitN(project, "@", 2)
		return client.Lookup(project[0], project[1])
	}
	return client.GetProject(project)
}

func init() {
	Command.Flags().StringVarP(&api, "url", "u", "", "Dependency-Track API server URL")
	Command.Flags().StringVarP(&key, "key", "k", "", "Dependency-Track API key")
	Command.Flags().StringVarP(&project, "project", "P", "", "Dependency-Track project in the form of an UUID or 'name@version'")
}
