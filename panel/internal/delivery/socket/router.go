package socket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/config"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
	"github.com/FareinheitsTemp/opd_panel/panel/internal/usecase"
)

type Router struct {
	cfg        *config.Config
	serverUC   *usecase.ServerUseCase
	scheduleUC *usecase.ScheduleUseCase
	subuserUC  *usecase.SubuserUseCase
	networkUC  *usecase.NetworkUseCase
	databaseUC *usecase.DatabaseUseCase
}

func NewRouter(
	cfg *config.Config,
	serverUC *usecase.ServerUseCase,
	scheduleUC *usecase.ScheduleUseCase,
	subuserUC *usecase.SubuserUseCase,
	networkUC *usecase.NetworkUseCase,
	databaseUC *usecase.DatabaseUseCase,
) *Router {
	return &Router{
		cfg:        cfg,
		serverUC:   serverUC,
		scheduleUC: scheduleUC,
		subuserUC:  subuserUC,
		networkUC:  networkUC,
		databaseUC: databaseUC,
	}
}

func (r *Router) Handle(ctx context.Context, req *Request, enc *json.Encoder) {
	var err error
	var data map[string]any

	switch req.Action {
	case "ping":
		data = map[string]any{"pong": true}

	// ── Servers ──────────────────────────────────────────────────
	case "server.list":
		servers, e := r.serverUC.List(ctx)
		if e != nil { err = e; break }
		data = map[string]any{"servers": servers}

	case "server.create":
		in := usecase.CreateServerInput{
			Name:    str(req.Payload, "name"),
			Type:    domain_type(req.Payload, "type"),
			Version: str(req.Payload, "version"),
			Port:    intVal(req.Payload, "port"),
			RAMMin:  intVal(req.Payload, "ram_min"),
			RAMMax:  intVal(req.Payload, "ram_max"),
		}
		s, e := r.serverUC.Create(ctx, in)
		if e != nil { err = e; break }
		data = map[string]any{"server": s}

	case "server.start":
		err = r.serverUC.Start(ctx, str(req.Payload, "server_id"))
		if err == nil { data = map[string]any{"status": "starting"} }

	case "server.stop":
		err = r.serverUC.Stop(ctx, str(req.Payload, "server_id"))
		if err == nil { data = map[string]any{"status": "stopping"} }

	case "server.restart":
		err = r.serverUC.Restart(ctx, str(req.Payload, "server_id"))
		if err == nil { data = map[string]any{"status": "restarting"} }

	case "server.info":
		s, metrics, e := r.serverUC.Info(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"server": s, "metrics": metrics}

	case "server.delete":
		err = r.serverUC.Delete(ctx, str(req.Payload, "server_id"))
		if err == nil { data = map[string]any{"ok": true} }

	// ── Schedules ─────────────────────────────────────────────────
	case "schedule.list":
		list, e := r.scheduleUC.List(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"schedules": list}

	case "schedule.create":
		tasks := parseTasks(req.Payload)
		in := usecase.CreateScheduleInput{
			ServerID: str(req.Payload, "server_id"),
			Name:     str(req.Payload, "name"),
			CronExpr: str(req.Payload, "cron_expr"),
			Enabled:  boolVal(req.Payload, "enabled"),
			Tasks:    tasks,
		}
		s, e := r.scheduleUC.Create(ctx, in)
		if e != nil { err = e; break }
		data = map[string]any{"schedule": s}

	case "schedule.update":
		tasks := parseTasks(req.Payload)
		in := usecase.CreateScheduleInput{
			ServerID: str(req.Payload, "server_id"),
			Name:     str(req.Payload, "name"),
			CronExpr: str(req.Payload, "cron_expr"),
			Enabled:  boolVal(req.Payload, "enabled"),
			Tasks:    tasks,
		}
		s, e := r.scheduleUC.Update(ctx, str(req.Payload, "schedule_id"), in)
		if e != nil { err = e; break }
		data = map[string]any{"schedule": s}

	case "schedule.delete":
		err = r.scheduleUC.Delete(ctx, str(req.Payload, "schedule_id"))
		if err == nil { data = map[string]any{"ok": true} }

	case "schedule.run":
		err = r.scheduleUC.RunNow(ctx, str(req.Payload, "schedule_id"))
		if err == nil { data = map[string]any{"ok": true} }

	// ── Subusers ──────────────────────────────────────────────────
	case "subuser.list":
		list, e := r.subuserUC.List(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"subusers": list}

	case "subuser.add":
		perms := strSlice(req.Payload, "permissions")
		s, e := r.subuserUC.Add(ctx, str(req.Payload, "server_id"), str(req.Payload, "email"), perms)
		if e != nil { err = e; break }
		data = map[string]any{"subuser": s}

	case "subuser.update":
		perms := strSlice(req.Payload, "permissions")
		s, e := r.subuserUC.UpdatePermissions(ctx, str(req.Payload, "subuser_id"), perms)
		if e != nil { err = e; break }
		data = map[string]any{"subuser": s}

	case "subuser.remove":
		err = r.subuserUC.Remove(ctx, str(req.Payload, "subuser_id"))
		if err == nil { data = map[string]any{"ok": true} }

	// ── Network / Ports ───────────────────────────────────────────
	case "network.list":
		list, e := r.networkUC.List(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"allocations": list}

	case "network.assign":
		a, e := r.networkUC.Assign(ctx,
			str(req.Payload, "server_id"),
			str(req.Payload, "ip"),
			str(req.Payload, "alias"),
		)
		if e != nil { err = e; break }
		data = map[string]any{"allocation": a}

	case "network.free":
		err = r.networkUC.Free(ctx, str(req.Payload, "allocation_id"))
		if err == nil { data = map[string]any{"ok": true} }

	// ── Databases ─────────────────────────────────────────────────
	case "database.list":
		list, e := r.databaseUC.List(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"databases": list}

	case "database.create":
		d, password, e := r.databaseUC.Create(ctx, str(req.Payload, "server_id"))
		if e != nil { err = e; break }
		data = map[string]any{"database": d, "password": password}

	case "database.delete":
		err = r.databaseUC.Delete(ctx, str(req.Payload, "database_id"))
		if err == nil { data = map[string]any{"ok": true} }

	default:
		err = fmt.Errorf("unknown action: %s", req.Action)
	}

	if err != nil {
		_ = enc.Encode(Response{ID: req.ID, OK: false, Error: err.Error()})
		return
	}
	_ = enc.Encode(Response{ID: req.ID, OK: true, Data: data})
}

// ── helpers ───────────────────────────────────────────────────────────────

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func boolVal(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func strSlice(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok { return nil }
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok { out = append(out, s) }
		}
		return out
	}
	return nil
}

func domain_type(m map[string]any, key string) domain.ServerType {
	return domain.ServerType(str(m, key))
}

func parseTasks(m map[string]any) []domain.ScheduleTask {
	raw, ok := m["tasks"]
	if !ok { return nil }
	bytes, err := json.Marshal(raw)
	if err != nil { return nil }
	var tasks []domain.ScheduleTask
	_ = json.Unmarshal(bytes, &tasks)
	return tasks
}
