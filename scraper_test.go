package iconscraper

import (
	"testing"
)

func TestSomeSites(t *testing.T) {
	config := Config{
		SquareOnly:            true,
		TargetHeight:          128,
		MaxConcurrentRequests: 32,
		AllowSvg:              false,
	}
	domains := []string{"google.com", "jotpot.uk", "example.com", "gov.uk", "mevitae.com", "microsoft.com", "apple.com", "golang.org", "rust-lang.org", "pkg.go.dev"}
	icons := GetIcons(config, domains)
	if icon, ok := icons["example.com"]; ok {
		t.Error("found icon for example.com", ok, icon)
	}
	if icon, ok := icons["jotpot.uk"]; !ok || icon.ImageConfig.Height != 144 {
		t.Error("didn't find icon for jotpot.uk", ok, icon)
	}
	if icon, ok := icons["google.com"]; !ok || icon.ImageConfig.Height != 128 {
		t.Error("didn't find icon for google.com", ok, icon)
	}
	if icon, ok := icons["gov.uk"]; !ok || icon.ImageConfig.Height != 152 {
		t.Error("didn't find icon for gov.uk", ok, icon)
	}
	if icon, ok := icons["mevitae.com"]; !ok || icon.ImageConfig.Height != 300 {
		t.Error("didn't find icon for mevitae.com", ok, icon)
	}
	if icon, ok := icons["microsoft.com"]; !ok || icon.ImageConfig.Height != 128 {
		t.Error("didn't find icon for microsoft.com", ok, icon)
	}
	if icon, ok := icons["apple.com"]; !ok || icon.ImageConfig.Height != 64 {
		t.Error("didn't find icon for apple.com", ok, icon)
	}
	if icon, ok := icons["golang.org"]; !ok || icon.ImageConfig.Height != 288 {
		t.Error("didn't find icon for golang.org", ok, icon)
	}
	if icon, ok := icons["rust-lang.org"]; !ok || icon.ImageConfig.Height != 180 {
		t.Error("didn't find icon for rust-land.org", ok, icon)
	}
	if icon, ok := icons["pkg.go.dev"]; !ok || icon.ImageConfig.Height != 32 {
		t.Error("didn't find icon for pkg.go.dev", ok, icon)
	}
}

func TestSomeSitesWithSVG(t *testing.T) {
	config := Config{
		SquareOnly:            true,
		TargetHeight:          32,
		MaxConcurrentRequests: 32,
		AllowSvg:              true,
	}
	domains := []string{"google.com", "jotpot.uk", "example.com", "gov.uk", "mevitae.com", "microsoft.com", "apple.com", "golang.org", "rust-lang.org", "pkg.go.dev"}
	icons := GetIcons(config, domains)
	if icon, ok := icons["example.com"]; ok {
		t.Error("found icon for example.com", ok, icon)
	}
	if icon, ok := icons["jotpot.uk"]; !ok || icon.ImageConfig.Height != 32 {
		t.Error("didn't find icon for jotpot.uk", ok, icon)
	}
	if icon, ok := icons["google.com"]; !ok || icon.ImageConfig.Height != 32 {
		t.Error("didn't find icon for google.com", ok, icon)
	}
	if icon, ok := icons["gov.uk"]; !ok || icon.ImageConfig.Height != 48 {
		t.Error("didn't find icon for gov.uk", ok, icon)
	}
	if icon, ok := icons["mevitae.com"]; !ok || icon.ImageConfig.Height != 100 {
		t.Error("didn't find icon for mevitae.com", ok, icon)
	}
	if icon, ok := icons["microsoft.com"]; !ok || icon.ImageConfig.Height != 128 {
		t.Error("didn't find icon for microsoft.com", ok, icon)
	}
	if icon, ok := icons["apple.com"]; !ok || icon.ImageConfig.Height != 64 {
		t.Error("didn't find icon for apple.com", ok, icon)
	}
	if icon, ok := icons["golang.org"]; !ok || icon.Type != svgMimeType {
		t.Error("didn't find icon for golang.org", ok, icon)
	}
	if icon, ok := icons["rust-lang.org"]; !ok || icon.Type != svgMimeType {
		t.Error("didn't find icon for rust-lang.org", ok, icon)
	}
	if icon, ok := icons["pkg.go.dev"]; !ok || icon.ImageConfig.Height != 32 {
		t.Error("didn't find icon for pkg.go.dev", ok, icon)
	}
}
