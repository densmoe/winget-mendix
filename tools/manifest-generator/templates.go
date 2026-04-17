package main

import "text/template"

var versionManifestTemplate = template.Must(template.New("version").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
`))

var installerManifestTemplate = template.Must(template.New("installer").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
InstallModes:
  - interactive
  - silent
Installers:{{range .Installers}}
  - Architecture: {{.Arch}}
    InstallerType: exe
    Scope: {{.Scope}}
    InstallerUrl: {{.URL}}
    InstallerSha256: {{.SHA256}}{{if .ElevationRequirement}}
    ElevationRequirement: {{.ElevationRequirement}}{{end}}
    ProductCode: "{{.GUID}}"{{end}}
InstallationNotes: "Multiple versions can be installed side-by-side."
ManifestType: installer
ManifestVersion: 1.6.0
`))

var localeManifestTemplate = template.Must(template.New("locale").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
PackageLocale: en-US
Publisher: Mendix
PublisherUrl: https://www.mendix.com/
PublisherSupportUrl: https://www.mendix.com/support/
PackageUrl: https://www.mendix.com/studio-pro/
PackageName: Mendix Studio Pro
License: Proprietary
ShortDescription: Low-code application development platform
ManifestType: defaultLocale
ManifestVersion: 1.6.0
`))

type ManifestData struct {
	Version    string
	Installers []InstallerData
}

type InstallerData struct {
	Arch                 string
	Scope                string
	URL                  string
	SHA256               string
	GUID                 string
	ElevationRequirement string
}
