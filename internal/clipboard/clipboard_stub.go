//go:build !windows

package clipboard

// platformStart is nil on non-Windows — the Watcher becomes a no-op.

// Stub format IDs for ComputeTagMask on non-Windows (never actually used).
var cfHTML uint32 = 0xC000
var cfRTF uint32 = 0xC001
