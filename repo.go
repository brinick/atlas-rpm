package rpm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rpm "github.com/cavaliercoder/go-rpm"
)

// Repos is a collection of RPM repo instances
type Repos []Repo

// Repo represents an RPM repository
type Repo struct {
	Name    string
	Label   string
	URL     string
	Prefix  string
	Enabled bool
}

// Filename returns the file name into which this repo will write its description
func (r Repo) Filename() string {
	return fmt.Sprintf("%s.repo", r.Label)
}
func (r Repo) String() string {
	var tokens []string
	tokens = append(tokens, fmt.Sprintf("[%s]", r.Label))
	tokens = append(tokens, fmt.Sprintf("name=%s", r.Name))
	tokens = append(tokens, fmt.Sprintf("baseurl=%s", r.URL))
	tokens = append(tokens, fmt.Sprintf("enabled=%t", r.Enabled))
	if len(r.Prefix) > 0 {
		tokens = append(tokens, fmt.Sprintf("prefix=%s", r.Prefix))
	}
	return strings.Join(tokens, "\n") + "\n"
}

// ---------------------------------------------------------------------

// NewFinder creates a new RPM Finder object
func NewFinder(path string) *Finder {
	return &Finder{
		basedir: path,
	}
}

// Finder is the object that locates RPMs below a given base directory
type Finder struct {
	basedir string
}

// SrcDir returns the path to the root directory below which RPMs are found
func (f *Finder) SrcDir() string {
	return f.basedir
}

type pathGlob func(string) ([]string, error)

// findTopRPM finds the top RPM which we need to install (with its dependencies)
func (f *Finder) findTopRPM(glob pathGlob, project, platform string) (string, error) {
	fname := fmt.Sprintf("%s_*_%s.rpm", project, platform)
	fpath := filepath.Join(f.basedir, fname)
	matches, err := glob(fpath)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no top RPM found to install (%s)", fpath)
	}

	return matches[0], nil
}

// Find is the method that finds RPMs
func (f *Finder) Find(project, platform string) (*RPMs, error) {
	path, err := f.findTopRPM(filepath.Glob, project, platform)
	if err != nil {
		return nil, err
	}
	topRPM, err := New(path)
	if topRPM.Size == 0 {
		return nil, fmt.Errorf("%s: RPM has zero size", path)
	}

	deps, err := topRPM.LocalDependencies()
	if err != nil {
		return nil, err
	}

	// Ensure that no dependencies have zero size, else fail
	emptyDeps := deps.ZeroSize()
	if len(emptyDeps) > 0 {
		err = fmt.Errorf(
			"%d rpm dependencies in %s have zero size:\n%s",
			len(emptyDeps),
			path,
			strings.Join(emptyDeps, "\n"),
		)
		return nil, err
	}

	// Prepend the topRPM
	allRPMs := RPMs(append([]*RPM{topRPM}, *deps...))
	return &allRPMs, nil
}

// ---------------------------------------------------------------------

// New creates an RPM instance for the RPM at the given path
func New(path string) (*RPM, error) {
	size, err := fileSize(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get rpm file size (%w)", err)
	}

	return &RPM{Path: path, Size: size}, nil
}

// ---------------------------------------------------------------------

// RPMs is a collection of RPM instances
type RPMs []*RPM

// ZeroSize returns the list of those RPMs that are empty length
func (r *RPMs) ZeroSize() []string {
	var zero []string
	for _, rr := range *r {
		if rr.Size == 0 {
			zero = append(zero, filepath.Base(rr.Path))
		}
	}

	return zero
}

// Paths returns the paths to each of the RPM instances
func (r *RPMs) Paths() []string {
	var paths []string
	for _, rpm := range *r {
		paths = append(paths, rpm.Path)
	}

	return paths
}

// Names returns the names of each of the RPM instances
func (r *RPMs) Names() []string {
	var names []string
	for _, rpm := range *r {
		names = append(names, rpm.Name())
	}

	return names
}

// ---------------------------------------------------------------------

// RPM is the basic wrapper around the given RPM path
type RPM struct {
	Path string
	Size int64
}

// Name returns the name of the RPM
func (r *RPM) Name() string {
	return filepath.Base(r.Path)
}

// NameStartsWith indicates if the RPM name has the given prefix
func (r *RPM) NameStartsWith(prefix string) bool {
	return strings.HasPrefix(r.Name(), prefix)
}

// LocalDependencies finds only those dependencies
// that are in the same directory as the RPM
func (r *RPM) LocalDependencies() (*RPMs, error) {
	deps, err := listDeps(r.Path)
	if err != nil {
		return nil, err
	}

	deps, err = listDir(filepath.Dir(r.Path), deps)
	if err != nil {
		return nil, err
	}

	var localdeps []*RPM
	for _, dep := range deps {
		depPath := filepath.Join(filepath.Dir(r.Path), dep)
		fi, err := os.Stat(depPath)
		if err != nil {
			return nil, fmt.Errorf("cannot get file size for dependency %s (%w)", depPath, err)
		}
		depSize := fi.Size()
		localdeps = append(localdeps, &RPM{depPath, depSize})
	}

	rpmsList := RPMs(localdeps)
	return &rpmsList, nil
}

// --------------------------------------------------------------------

// listDeps is a helper function to get the names of
// dependencies of a given starting root RPM
func listDeps(path string) ([]string, error) {
	p, err := rpm.OpenPackageFile(path)
	if err != nil {
		return nil, err
	}

	deps := p.Requires()
	names := make([]string, len(deps))
	for _, dep := range deps {
		names = append(names, dep.Name())
	}

	return names, nil
}

func listDir(dir string, filenames []string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	lut := toLUT(filenames)

	var found []string
	for _, entry := range entries {
		name := entry.Name()
		if _, keyExists := lut[name]; keyExists && !entry.IsDir() {
			found = append(found, name)
		}
	}

	return found, nil
}

func toLUT(items []string) map[string]struct{} {
	var m = map[string]struct{}{}
	for _, item := range items {
		m[item] = struct{}{}
	}

	return m
}

func fileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	return fi.Size(), nil
}
