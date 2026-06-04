//go:build !production

package log

func init() {
	terminalOutput = true
}
