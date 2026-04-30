package control

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"frp-ui-backend/internal/httpx"
)

func NewHandler(svc *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/groups", func(w http.ResponseWriter, r *http.Request) {
		var req CreateGroupRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateGroup(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /v1/groups/{groupId}/join", func(w http.ResponseWriter, r *http.Request) {
		var req JoinGroupRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.JoinGroup(r.PathValue("groupId"), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

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

	mux.HandleFunc("DELETE /v1/nodes/{nodeId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteNode(r.PathValue("nodeId")); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("GET /v1/exposures", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListExposures(r.URL.Query().Get("groupId"))
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/exposures/private", func(w http.ResponseWriter, r *http.Request) {
		var req CreatePrivateExposureRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreatePrivateExposure(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /v1/exposures/public", func(w http.ResponseWriter, r *http.Request) {
		var req CreatePublicExposureRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreatePublicExposure(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("PATCH /v1/exposures/{exposureId}", func(w http.ResponseWriter, r *http.Request) {
		var req UpdateExposureRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.UpdateExposure(r.PathValue("exposureId"), authFromHeaders(r), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("DELETE /v1/exposures/{exposureId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteExposure(r.PathValue("exposureId"), authFromHeaders(r)); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("POST /v1/access-routes", func(w http.ResponseWriter, r *http.Request) {
		var req CreateAccessRouteRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateAccessRoute(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("GET /v1/rooms", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.ListRooms()
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("POST /v1/rooms", func(w http.ResponseWriter, r *http.Request) {
		var req CreateRoomRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.CreateRoom(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /v1/rooms/join", func(w http.ResponseWriter, r *http.Request) {
		var req JoinRoomRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.JoinRoom(req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("GET /v1/rooms/{roomId}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.GetRoom(r.PathValue("roomId"))
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("PATCH /v1/rooms/{roomId}", func(w http.ResponseWriter, r *http.Request) {
		var req UpdateRoomRequest
		if err := httpx.Decode(r, &req); err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		resp, err := svc.UpdateRoom(r.PathValue("roomId"), roomAuthFromHeaders(r), req)
		if err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("DELETE /v1/rooms/{roomId}", func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteRoom(r.PathValue("roomId"), roomAuthFromHeaders(r)); err != nil {
			httpx.HandleError(w, err)
			return
		}
		httpx.Empty(w, http.StatusNoContent)
	})

	mux.HandleFunc("POST /internal/frps/plugin", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.HandleError(w, httpx.BadRequest(err.Error()))
			return
		}
		if !json.Valid(raw) {
			httpx.HandleError(w, httpx.BadRequest("invalid json"))
			return
		}
		httpx.JSON(w, http.StatusOK, svc.PluginDecision(raw, r.URL.Query().Get("op")))
	})

	return httpx.CORS(mux)
}

func authFromHeaders(r *http.Request) DeviceAuth {
	token := r.Header.Get("X-Device-Token")
	if token == "" {
		authz := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			token = strings.TrimSpace(authz[len("bearer "):])
		}
	}
	return DeviceAuth{
		GroupID:     r.Header.Get("X-Group-ID"),
		DeviceID:    r.Header.Get("X-Device-ID"),
		DeviceToken: token,
	}
}

func roomAuthFromHeaders(r *http.Request) RoomDeviceAuth {
	token := r.Header.Get("X-Room-Device-Token")
	if token == "" {
		authz := r.Header.Get("Authorization")
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			token = strings.TrimSpace(authz[len("bearer "):])
		}
	}
	return RoomDeviceAuth{
		RoomID:      r.Header.Get("X-Room-ID"),
		DeviceID:    r.Header.Get("X-Room-Device-ID"),
		DeviceToken: token,
	}
}
