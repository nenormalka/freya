package common

type (
	Config struct {
		Addresses   []string
		SkipErrors  map[Topic]struct{}
		EnableDebug bool
		AppName     string
	}
)
