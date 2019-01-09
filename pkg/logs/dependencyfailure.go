package logs

const (
	// DependencyStorage identifies a storage failure
	DependencyStorage = "storage"

	//DependencyQueuer identifies a queuer failure
	DependencyQueuer = "queuer"

	// DependencyMarker identifies a marker failure
	DependencyMarker = "marker"
)

// DependencyFailure is logged when a downstream dependency fails
type DependencyFailure struct {
	Dependency string `logevent:"dependency"`
	Reason     string `logevent:"reason"`
	Message    string `logevent:"message,default=dependency-failure"`
}
