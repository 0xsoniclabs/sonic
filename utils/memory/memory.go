// Package memory provides methods to obtain system memory information.
package memory

// #include <unistd.h>
import "C"

// TotalMemory returns the total system memory reported by the OS in bytes.
func TotalMemory() uint64 {
	return uint64(C.sysconf(C._SC_PHYS_PAGES)) * uint64(C.sysconf(C._SC_PAGE_SIZE))
}
