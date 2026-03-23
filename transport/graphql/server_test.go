package graphql

import (
	"net/http"
	"testing"
	"time"
)

func TestNewServer_appliesTimeoutAndHeaderLimits(t *testing.T) {
	s := NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil)

	if s.srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout: got %v, want %v", s.srv.ReadHeaderTimeout, 5*time.Second)
	}
	if s.srv.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout: got %v, want %v", s.srv.ReadTimeout, 10*time.Second)
	}
	if s.srv.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout: got %v, want %v", s.srv.WriteTimeout, 15*time.Second)
	}
	if s.srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout: got %v, want %v", s.srv.IdleTimeout, 60*time.Second)
	}
	if s.srv.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes: got %d, want %d", s.srv.MaxHeaderBytes, 1<<20)
	}
}
