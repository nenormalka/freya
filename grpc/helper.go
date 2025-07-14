package grpc

type (
	ServersHelper struct {
		GRPCDefinitions []Definition
	}
)

func newServersHelper() *ServersHelper {
	return &ServersHelper{
		GRPCDefinitions: make([]Definition, 0),
	}
}

func (s *ServersHelper) AddDefinition(def Definition) {
	s.GRPCDefinitions = append(s.GRPCDefinitions, def)
}
