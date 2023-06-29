package common

type (
	Config struct {
		Addresses           []string
		SkipUnmarshalErrors map[Topic]struct{}
	}
)
