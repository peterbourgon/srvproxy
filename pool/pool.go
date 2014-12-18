package pool

// Pool describes anything which can yield hosts.
type Pool interface {
	Get() (string, error)
	Put(string, bool)
}
