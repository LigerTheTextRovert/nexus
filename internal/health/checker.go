package health

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LigerTheTextRovert/nexus/internal/config"
)

type HealthChecker struct {
	Config   config.HealthCheck
	Backends []config.Backend
	client   http.Client
}
