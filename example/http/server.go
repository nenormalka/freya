package http

import (
	"github.com/gorilla/mux"
	httpdefault "net/http"

	"github.com/nenormalka/freya/example/service"
	"github.com/nenormalka/freya/http"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
	"go.uber.org/zap"
)

var Module = types.Module{
	{CreateFunc: newHTTP},
	{CreateFunc: adapter},
}

type (
	AdapterOut struct {
		dig.Out

		Server http.CustomServer `group:"custom_http_servers"`
	}
)

// adapter ...
func adapter(serv *Server) AdapterOut {
	return AdapterOut{
		Server: serv,
	}
}

type Server struct {
	logger  *zap.Logger
	service *service.Service
}

func newHTTP(
	logger *zap.Logger,
	service *service.Service,
) *Server {
	return &Server{
		logger:  logger,
		service: service,
	}
}

func (s *Server) GetServerName() string {
	return "example-http-server"
}

func (s *Server) StartServer(r *mux.Router) error {
	r.HandleFunc("/example/test", s.GetTest)
	return nil
}

func (s *Server) GetTest(resp httpdefault.ResponseWriter, req *httpdefault.Request) {
	now, err := s.service.Now(req.Context())
	if err != nil {
		resp.WriteHeader(httpdefault.StatusInternalServerError)
		resp.Write([]byte("500 - Something bad happened!"))

		return
	}

	s.logger.Info("request time", zap.String("time", now))

	resp.WriteHeader(httpdefault.StatusOK)
	resp.Write([]byte(now))
}
