package state

type Server struct {
	metrics Metrics
}

func (s *Server) Metrics() *Metrics {
	return &s.metrics
}
