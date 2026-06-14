package constant

type MultiKeyMode string

const (
	MultiKeyModeRandom    MultiKeyMode = "random"
	MultiKeyModePolling   MultiKeyMode = "polling"
	MultiKeyModeFillFirst MultiKeyMode = "fill_first"
)
