package updater

// Updater is the interface that all dependency updaters must implement.
type Updater interface {
	// Name returns the name of the updater (e.g., "docker", "gomod").
	Name() string
	// Scan checks a given file or directory for potential updates.
	Scan(path string) ([]Dependency, error)
	// ApplyUpdate applies a specific update.
	ApplyUpdate(dep Dependency) error
}

// Dependency represents a single dependency that can be updated.
type Dependency struct {
	// The name of the dependency (e.g., "docker.io/library/ubuntu").
	Name string
	// The current version found.
	CurrentVersion string
	// The latest available version.
	LatestVersion string
	// The file where the dependency is defined.
	File string
	// The updater that found this dependency.
	UpdaterName string
}