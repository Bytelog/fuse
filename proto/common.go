package proto

const (
	// space reserved in buffer for header
	BUFFER_HEADER_SIZE = 4096

	// maximum number of pages received in InitOut
	MAX_MAX_PAGES = 256

	// coarsest allowed time granularity
	MAX_TIME_GRAN = 1e9
)
