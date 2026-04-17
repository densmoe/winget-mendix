package main

import "text/template"

var packageManifestTemplate = template.Must(template.New("package").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
PackageName: Mendix Studio Pro
Publisher: Mendix
PublisherUrl: https://www.mendix.com/
PublisherSupportUrl: https://www.mendix.com/support/
PackageUrl: https://www.mendix.com/studio-pro/
License: Proprietary
ShortDescription: Low-code application development platform
ManifestVersion: 1.4.0
`))

var installerManifestTemplate = template.Must(template.New("installer").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
InstallModes:
  - interactive
  - silent
Installers:{{range .Installers}}
  - Architecture: {{.Arch}}
    InstallerType: exe
    InstallerUrl: {{.URL}}
    InstallerSha256: {{.SHA256}}
    ProductCode: "{{.GUID}}"{{end}}
InstallationNotes: "Multiple versions can be installed side-by-side."
ManifestVersion: 1.4.0
`))

var localeManifestTemplate = template.Must(template.New("locale").Parse(`PackageIdentifier: Mendix.MendixStudioPro
PackageVersion: {{.Version}}
PackageLocale: en-US
Publisher: Mendix
PublisherUrl: https://www.mendix.com/
PublisherSupportUrl: https://www.mendix.com/support/
PackageUrl: https://www.mendix.com/studio-pro/
License: Proprietary
ShortDescription: Low-code application development platform
ManifestVersion: 1.4.0
`))

type ManifestData struct {
	Version    string
	Installers []InstallerData
}

type InstallerData struct {
	Arch   string
	URL    string
	SHA256 string
	GUID   string
}
