package client

import (
	"net/http"

	"frp-ui-backend/internal/httpx"
)

func NewHandler(svc *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListNodes()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		var req CreateNodeRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateNode(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("PUT /v1/nodes/{nodeId}", func(w http.ResponseWriter, r *http.Request) {
		var req UpdateNodeRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.UpdateNode(r.Context(), r.PathValue("nodeId"), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/nodes/{nodeId}/doctor", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.DoctorNode(r.Context(), r.PathValue("nodeId"))
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("DELETE /v1/nodes/{nodeId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteNode(r.PathValue("nodeId")); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("GET /v1/ports", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListPortRules()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/ports", func(w http.ResponseWriter, r *http.Request) {
		var req CreatePortRuleRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreatePortRule(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("PUT /v1/ports/{portId}", func(w http.ResponseWriter, r *http.Request) {
		var req UpdatePortRuleRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.UpdatePortRule(r.Context(), r.PathValue("portId"), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("PATCH /v1/ports/{portId}", func(w http.ResponseWriter, r *http.Request) {
		var req PatchPortRuleRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.PatchPortRule(r.Context(), r.PathValue("portId"), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("DELETE /v1/ports/{portId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeletePortRule(r.Context(), r.PathValue("portId")); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("GET /v1/client/rooms", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListRoomRules()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("GET /v1/client/rooms/status", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListRoomRuleStatuses()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/client/rooms/host", func(w http.ResponseWriter, r *http.Request) {
		var req CreateRoomHostRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateRoomHost(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /v1/client/rooms/join", func(w http.ResponseWriter, r *http.Request) {
		var req JoinRoomRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.JoinRoom(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("PATCH /v1/client/rooms/{roomRuleId}", func(w http.ResponseWriter, r *http.Request) {
		var req PatchRoomRuleRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.PatchRoomRule(r.Context(), r.PathValue("roomRuleId"), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("DELETE /v1/client/rooms/{roomRuleId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteRoomRule(r.Context(), r.PathValue("roomRuleId")); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("POST /v1/client/rooms/{roomRuleId}/doctor", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.DoctorRoomRule(r.Context(), r.PathValue("roomRuleId"))
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("GET /v1/client/network/interfaces", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListNetworkInterfaces()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/client/xtcp/nathole/discover", func(w http.ResponseWriter, r *http.Request) {
		var req NatHoleDiscoverRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.DiscoverNatHole(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/client/server", func(w http.ResponseWriter, r *http.Request) {
		var req ConfigureServerRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.ConfigureServer(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/client/groups/join", func(w http.ResponseWriter, r *http.Request) {
		var req ClientJoinGroupRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.JoinGroup(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/client/exposures", func(w http.ResponseWriter, r *http.Request) {
		var req ClientCreateExposureRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateExposure(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("DELETE /v1/client/exposures/{exposureId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteExposure(r.Context(), r.PathValue("exposureId")); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("POST /v1/client/access-routes", func(w http.ResponseWriter, r *http.Request) {
		var req ClientCreateAccessRouteRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateAccessRoute(r.Context(), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("GET /v1/client/frpc/status", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, svc.Status())
	})

	mux.HandleFunc("GET /v1/frpc/status", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, svc.Status())
	})

	mux.HandleFunc("POST /v1/client/frpc/reload", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.Reload(r.Context())
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/frpc/reload", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.Reload(r.Context())
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("GET /v1/client/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(svc.Logs()))
	})

	mux.HandleFunc("GET /v1/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(svc.Logs()))
	})

	return httpx.CORS(mux)
}
