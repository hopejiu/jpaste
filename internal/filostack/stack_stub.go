//go:build !windows

package filostack

func init() {
	platformStartHook = startStubHook
}

func startStubHook(onVKeyDown func()) func() {
	return nil
}
