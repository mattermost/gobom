package cyclonedx

import "encoding/xml"

// BOM represents the bom element in the CycloneDX spec
type BOM struct {
	XMLName    xml.Name     `xml:"http://cyclonedx.org/schema/bom/1.2 bom"`
	Metadata   Metadata     `xml:"metadata"`
	Components []*Component `xml:"components>component"`
}

// Metadata represents bom metadata in the CycloneDX spec
type Metadata struct {
	Component *Component `xml:"component,omitempty"`
}

// Component represents a component in the CycloneDX spec
type Component struct {
	XMLName     xml.Name       `xml:"component"`
	Type        Classification `xml:"type,attr"`
	Group       string         `xml:"group,omitempty"`
	Name        string         `xml:"name"`
	Version     string         `xml:"version"`
	Description string         `xml:"description"`
	PURL        string         `xml:"purl"`
	Components  []*Component   `xml:"components>component"`
}

// Classification represents a classification in the CycloneDX spec
type Classification string

// Classification values
const (
	Application     Classification = "application"
	Framework       Classification = "framework"
	Library         Classification = "library"
	Container       Classification = "container"
	OperatingSystem Classification = "operating-system"
	Device          Classification = "device"
	Firmware        Classification = "firmware"
	File            Classification = "file"
)
