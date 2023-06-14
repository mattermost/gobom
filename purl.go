package gobom

import (
	"fmt"
	"strings"
)

// PURL package type
const (
	GenericPackage   = "generic"
	GolangPackage    = "golang"
	NpmPackage       = "npm"
	CocoapodsPackage = "cocoapods"
	GradlePackage    = "maven" // OSS Index doesn't support gradle PURLs so fall back to maven
	RpmPackage       = "rpm"
)

// PURL returns a package URL for the specified package type, name, and version
func PURL(packageType string, name string, version string) string {
	return fmt.Sprintf("pkg:%s/%s@%s", packageType, strings.ReplaceAll(name, "@", "%40"), version)
}
