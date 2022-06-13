package rpm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func createRepo() *Repo {
	return &Repo{
		Name:    "repo",
		Label:   "label",
		URL:     "https://example.repo",
		Prefix:  "blah",
		Enabled: false,
	}

}

func createRPMs() *RPMs {
	return &RPMs{
		&RPM{
			Path: "/blip/blop",
			Size: 12345,
		},
		&RPM{
			Path: "/blip/blop2",
			Size: 0,
		},
	}
}

func TestRepoStringer(t *testing.T) {
	got := createRepo().String()
	expect := "[label]\nname=repo\nbaseurl=https://example.repo\nenabled=false\nprefix=blah\n"

	if got != expect {
		t.Errorf("Repo stringer method should return %s, got %s", expect, got)
	}
}

func TestRepoName(t *testing.T) {
	got := createRepo().Filename()
	if got != "label.repo" {
		t.Errorf("Repo filename should be label.repo, got %s", got)
	}
}

func TestRPMsNames(t *testing.T) {
	rpms := createRPMs()
	got := len(rpms.Names())
	if got != 2 {
		t.Errorf("RPMs Names length should be 2, got %d", got)
	}
}

func TestRPMsZeroLength(t *testing.T) {
	got := createRPMs().ZeroSize()
	if len(got) != 1 {
		t.Errorf("RPMs zero size should return 1, got %d", len(got))
	}

	if got[0] != "blop2" {
		t.Errorf("RPMs zero size path should be blop2, got %s", got[0])
	}
}

func TestRPMFinderInexistantPath(t *testing.T) {
	// Inexistant path, so expect an error
	f := Finder{"/blip/blop"}
	_, err := f.findTopRPM(filepath.Glob, "project", "platform")
	if err == nil {
		t.Errorf("RPM finder should have returned an error, got nil")
	}
}

func TestRPMFinderTopRPM(t *testing.T) {
	// Inexistant path, so expect an error
	f := Finder{"/blip/blop"}
	getMatches := func(string) ([]string, error) {
		return []string{"topRPM.rpm"}, nil
	}

	top, _ := f.findTopRPM(getMatches, "project", "platform")
	if top != "topRPM.rpm" {
		t.Errorf("Top RPM finder failed, got %s, expected topRPM.rpm", top)
	}
}

func TestNewRPM(t *testing.T) {
	dir, err := ioutil.TempDir("", "atlas-rpm-installer-test")
	if err != nil {
		t.Errorf("TestNewRPM failed to create temp dir %v", err)
	}

	defer os.RemoveAll(dir)

	file, err := ioutil.TempFile(dir, "")
	if err != nil {
		t.Errorf("TestNewRPM failed to create temp file %v", err)
	}

	fName := file.Name()

	defer os.Remove(fName)
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("failed to close temp file %s (%v)", fName, err)
		}
	}()

	r, err := New(fName)
	if err != nil {
		t.Errorf("New failed to create an RPM at %s (%v)", fName, err)
	}

	if r.Path != fName {
		t.Errorf("Newly created RPM has bad path %s, expected %s", r.Path, fName)
	}

	if r.Size != 0 {
		t.Errorf("Newly created RPM has wrong size %d, expected 0", r.Size)
	}

}
