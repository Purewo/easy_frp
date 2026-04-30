import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { controlApi } from '../api/client';
import type {
  components,
} from '../api/types';

type Group = components['schemas']['Group'];
type Node = components['schemas']['Node'];
type Exposure = components['schemas']['Exposure'];
type AccessRoute = components['schemas']['AccessRoute'];
type CreateGroupRequest = components['schemas']['CreateGroupRequest'];
type JoinGroupRequest = components['schemas']['JoinGroupRequest'];
type JoinGroupResponse = components['schemas']['JoinGroupResponse'];
export type RoomView = components['schemas']['RoomView'];
export type RoomDeviceView = components['schemas']['RoomDeviceView'];
export type CreateRoomRequest = components['schemas']['CreateRoomRequest'];
export type CreateRoomResponse = components['schemas']['CreateRoomResponse'];
export type JoinRoomRequest = components['schemas']['JoinRoomRequest'];
export type JoinRoomResponse = components['schemas']['JoinRoomResponse'];
export type UpdateRoomRequest = components['schemas']['UpdateRoomRequest'];
type CreateNodeRequest = components['schemas']['CreateNodeRequest'];
type CreatePrivateExposureRequest = components['schemas']['CreatePrivateExposureRequest'];
type CreatePublicExposureRequest = components['schemas']['CreatePublicExposureRequest'];
type UpdateExposureRequest = components['schemas']['UpdateExposureRequest'];
type CreateAccessRouteRequest = components['schemas']['CreateAccessRouteRequest'];

export type RoomAuthHeaders = {
  roomId: string;
  deviceId: string;
  deviceToken: string;
};

function roomAuthHeaders(auth: RoomAuthHeaders) {
  return {
    'X-Room-ID': auth.roomId,
    'X-Room-Device-ID': auth.deviceId,
    'X-Room-Device-Token': auth.deviceToken,
  };
}

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation<Group, Error, CreateGroupRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/groups', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['groups'] }),
  });
}

export function useJoinGroup() {
  return useMutation<JoinGroupResponse, Error, { groupId: string; body: JoinGroupRequest }>({
    mutationFn: async ({ groupId, body }) => {
      const res = await controlApi.post(`/v1/groups/${groupId}/join`, body);
      return res.data;
    },
  });
}

export function useListRooms() {
  return useQuery<RoomView[]>({
    queryKey: ['rooms'],
    queryFn: async () => {
      const res = await controlApi.get('/v1/rooms');
      return res.data;
    },
  });
}

export function useCreateRoom() {
  const qc = useQueryClient();
  return useMutation<CreateRoomResponse, Error, CreateRoomRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/rooms', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['rooms'] }),
  });
}

export function useJoinRoom() {
  const qc = useQueryClient();
  return useMutation<JoinRoomResponse, Error, JoinRoomRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/rooms/join', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['rooms'] }),
  });
}

export function useUpdateRoom() {
  const qc = useQueryClient();
  return useMutation<RoomView, Error, { roomId: string; auth: RoomAuthHeaders; body: UpdateRoomRequest }>({
    mutationFn: async ({ roomId, auth, body }) => {
      const res = await controlApi.patch(`/v1/rooms/${roomId}`, body, { headers: roomAuthHeaders(auth) });
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['rooms'] }),
  });
}

export function useDeleteRoom() {
  const qc = useQueryClient();
  return useMutation<void, Error, { roomId: string; auth: RoomAuthHeaders }>({
    mutationFn: async ({ roomId, auth }) => {
      await controlApi.delete(`/v1/rooms/${roomId}`, { headers: roomAuthHeaders(auth) });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['rooms'] }),
  });
}

export function useListNodes() {
  return useQuery<Node[]>({
    queryKey: ['nodes'],
    queryFn: async () => {
      const res = await controlApi.get('/v1/nodes');
      return res.data;
    },
  });
}

export function useCreateNode() {
  const qc = useQueryClient();
  return useMutation<Node, Error, CreateNodeRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/nodes', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['nodes'] }),
  });
}

export function useDeleteNode() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: async (nodeId) => {
      await controlApi.delete(`/v1/nodes/${nodeId}`);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['nodes'] }),
  });
}

export function useListExposures(groupId?: string) {
  return useQuery<Exposure[]>({
    queryKey: ['exposures', groupId],
    queryFn: async () => {
      const res = await controlApi.get('/v1/exposures', {
        params: groupId ? { groupId } : undefined,
      });
      return res.data;
    },
  });
}

export function useCreatePrivateExposure() {
  const qc = useQueryClient();
  return useMutation<Exposure, Error, CreatePrivateExposureRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/exposures/private', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exposures'] }),
  });
}

export function useCreatePublicExposure() {
  const qc = useQueryClient();
  return useMutation<Exposure, Error, CreatePublicExposureRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/exposures/public', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exposures'] }),
  });
}

export function useUpdateExposure() {
  const qc = useQueryClient();
  return useMutation<Exposure, Error, { exposureId: string; headers: Record<string, string>; body: UpdateExposureRequest }>({
    mutationFn: async ({ exposureId, headers, body }) => {
      const res = await controlApi.patch(`/v1/exposures/${exposureId}`, body, { headers });
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exposures'] }),
  });
}

export function useDeleteExposure() {
  const qc = useQueryClient();
  return useMutation<void, Error, { exposureId: string; headers: Record<string, string> }>({
    mutationFn: async ({ exposureId, headers }) => {
      await controlApi.delete(`/v1/exposures/${exposureId}`, { headers });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['exposures'] }),
  });
}

export function useCreateAccessRoute() {
  const qc = useQueryClient();
  return useMutation<AccessRoute, Error, CreateAccessRouteRequest>({
    mutationFn: async (body) => {
      const res = await controlApi.post('/v1/access-routes', body);
      return res.data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['access-routes'] }),
  });
}
