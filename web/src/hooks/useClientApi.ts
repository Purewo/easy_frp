import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { clientApi } from '../api/client';

export type PortProtocol = 'tcp' | 'udp' | 'http';

export type LocalNode = {
  id: string;
  name: string;
  serverAddr: string;
  frpsPort: number;
  authMethod: 'token';
  authTokenSet: boolean;
  webBaseDomain?: string;
  webScheme?: string;
  vhostHTTPPort?: number;
  allowPorts?: Array<{ from: number; to: number }>;
  createdAt?: string;
  updatedAt?: string;
};

export type CreateNodeRequest = {
  id?: string;
  name: string;
  serverAddr: string;
  frpsPort: number;
  authMethod?: 'token';
  authToken?: string;
  webBaseDomain?: string;
  webScheme?: 'http' | 'https';
  vhostHTTPPort?: number;
  allowPorts?: Array<{ from: number; to: number }>;
};

export type UpdateNodeRequest = {
  name: string;
  serverAddr: string;
  frpsPort: number;
  authMethod?: 'token';
  authToken?: string;
  clearAuthToken?: boolean;
  webBaseDomain?: string;
  webScheme?: 'http' | 'https';
  vhostHTTPPort?: number;
  allowPorts?: Array<{ from: number; to: number }>;
};

export type NodeDoctorCheck = {
  id: string;
  name: string;
  status: 'pass' | 'warn' | 'fail' | 'skipped';
  message: string;
  durationMs?: number;
  details?: Record<string, string>;
};

export type NodeDoctorResult = {
  node: LocalNode;
  overall: 'pass' | 'warn' | 'fail';
  testedDomain?: string;
  checks: NodeDoctorCheck[];
};

export type PortRule = {
  id: string;
  nodeId: string;
  name: string;
  protocol: PortProtocol;
  localIP: string;
  localPort: number;
  remotePort?: number;
  subdomain?: string;
  domain?: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreatePortRuleRequest = {
  nodeId?: string;
  name?: string;
  protocol: PortProtocol;
  localIP?: string;
  localPort: number;
  remotePort?: number;
  subdomain?: string;
  domain?: string;
  enabled?: boolean;
};

export type UpdatePortRuleRequest = {
  nodeId?: string;
  name?: string;
  protocol: PortProtocol;
  localIP?: string;
  localPort: number;
  remotePort?: number;
  subdomain?: string;
  domain?: string;
  enabled: boolean;
};

export type FrpcStatus = {
  running: boolean;
  pid?: number;
  configPath: string;
  lastError?: string;
  nodes?: Array<{
    nodeId: string;
    running: boolean;
    pid?: number;
    configPath: string;
    logPath?: string;
    lastError?: string;
  }>;
};

export function getApiErrorMessage(error: unknown, fallback: string) {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { error?: string; message?: string } } }).response;
    return response?.data?.error || response?.data?.message || fallback;
  }

  return fallback;
}

export function useLocalNodes() {
  return useQuery<LocalNode[]>({
    queryKey: ['local-nodes'],
    queryFn: async () => {
      const res = await clientApi.get('/v1/nodes');
      return res.data;
    },
  });
}

export function useCreateNode() {
  const qc = useQueryClient();
  return useMutation<LocalNode, Error, CreateNodeRequest>({
    mutationFn: async (body) => {
      const res = await clientApi.post('/v1/nodes', body);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['local-nodes'] });
    },
  });
}

export function useUpdateNode() {
  const qc = useQueryClient();
  return useMutation<LocalNode, Error, { nodeId: string; body: UpdateNodeRequest }>({
    mutationFn: async ({ nodeId, body }) => {
      const res = await clientApi.put(`/v1/nodes/${nodeId}`, body);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['local-nodes'] });
      qc.invalidateQueries({ queryKey: ['port-rules'] });
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
    },
  });
}

export function useDeleteNode() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: async (nodeId) => {
      await clientApi.delete(`/v1/nodes/${nodeId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['local-nodes'] });
    },
  });
}

export function useDoctorNode() {
  return useMutation<NodeDoctorResult, Error, string>({
    mutationFn: async (nodeId) => {
      const res = await clientApi.post(`/v1/nodes/${nodeId}/doctor`);
      return res.data;
    },
  });
}

export function usePortRules() {
  return useQuery<PortRule[]>({
    queryKey: ['port-rules'],
    queryFn: async () => {
      const res = await clientApi.get('/v1/ports');
      return res.data;
    },
  });
}

export function useCreatePortRule() {
  const qc = useQueryClient();
  return useMutation<PortRule, Error, CreatePortRuleRequest>({
    mutationFn: async (body) => {
      const res = await clientApi.post('/v1/ports', body);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['port-rules'] });
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
    },
  });
}

export function useUpdatePortRule() {
  const qc = useQueryClient();
  return useMutation<PortRule, Error, { portId: string; body: UpdatePortRuleRequest }>({
    mutationFn: async ({ portId, body }) => {
      const res = await clientApi.put(`/v1/ports/${portId}`, body);
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['port-rules'] });
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
    },
  });
}

export function usePatchPortRule() {
  const qc = useQueryClient();
  return useMutation<PortRule, Error, { portId: string; enabled: boolean }>({
    mutationFn: async ({ portId, enabled }) => {
      const res = await clientApi.patch(`/v1/ports/${portId}`, { enabled });
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['port-rules'] });
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
    },
  });
}

export function useDeletePortRule() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: async (portId) => {
      await clientApi.delete(`/v1/ports/${portId}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['port-rules'] });
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
    },
  });
}

export function useClientFrpcStatus() {
  return useQuery<FrpcStatus>({
    queryKey: ['frpc-status'],
    queryFn: async () => {
      const res = await clientApi.get('/v1/frpc/status');
      return res.data;
    },
    refetchInterval: 3000,
  });
}

export function useClientFrpcReload() {
  const qc = useQueryClient();
  return useMutation<FrpcStatus, Error, void>({
    mutationFn: async () => {
      const res = await clientApi.post('/v1/frpc/reload');
      return res.data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['frpc-status'] });
      qc.invalidateQueries({ queryKey: ['port-rules'] });
    },
  });
}

export function useClientLogs() {
  return useQuery<string>({
    queryKey: ['frpc-logs'],
    queryFn: async () => {
      const res = await clientApi.get('/v1/logs');
      return res.data;
    },
    refetchInterval: 5000,
  });
}
