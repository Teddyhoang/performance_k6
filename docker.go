package k6

import "go.k6.io/k6/js/modules"

// init is called by the Go runtime at application startup.
func init() {
    images := Images{}
	images.SetupClient()
	modules.Register("k6/x/compare", &images)
}