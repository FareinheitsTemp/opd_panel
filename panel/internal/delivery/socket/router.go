package socket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/config"
)

type Router struct {
	cfg *config.Config
	// TODO: inject usecases here
}

func NewRouter(cfg *config.Config) *Router {
	return &Router{cfg: cfg}
}

func (r *Router) Handle(ctx context.Context, req *Request, enc *json.Encoder) {
	var err error
	var data map[string]any

	switch req.Action {
	case "ping":
		data = map[string]any{"pong": true}

	case "server.list":
		// TODO: r.serverUC.List(ctx)
		data = map[string]any{"servers": []any{}}

	case "server.start":
		id, _ := req.Payload["server_id"].(string)
		_ = id
		// TODO: r.serverUC.Start(ctx, id)
		data = map[string]any{"status": "starting"}

	case "server.stop":
		id, _ := req.Payload["server_id"].(string)
		_ = id
		// TODO: r.serverUC.Stop(ctx, id)
		data = map[string]any{"status": "stopping"}

	default:
		err = fmt.Errorf("unknown action: %s", req.Action)
	}

	if err != nil {
		_ = enc.Encode(Response{ID: req.ID, OK: false, Error: err.Error()})
		return
	}
	_ = enc.Encode(Response{ID: req.ID, OK: true, Data: data})
}
