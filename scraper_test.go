package scraper

import (
	"testing"
)

func Test1(t *testing.T) {
	print("-----")
	domains := []string{"hs2.org.uk", "google.com", "britishland.com", "welocalize.com"}
	GetIcons(domains, true, 120, 20)
	println("GetIcons --- ")
}

func Test2(t *testing.T) {
	//	println("-------")
	//	GetIcon("hs2.org.uk", true, 120, 5)
	//
	// /	println("Get Icon")
}
