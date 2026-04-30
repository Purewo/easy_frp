import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  GlobalOutlined,
  KeyOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  DatabaseOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import {
  getApiErrorMessage,
  type CreateNodeRequest,
  type LocalNode,
  type NodeDoctorResult,
  type UpdateNodeRequest,
  useCreateNode,
  useDeleteNode,
  useDoctorNode,
  useLocalNodes,
  usePortRules,
  useUpdateNode,
} from '../../hooks/useClientApi';

const { Title, Text } = Typography;

type NodeFormValues = {
  id?: string;
  name: string;
  serverAddr: string;
  frpsPort: number;
  authToken?: string;
  clearAuthToken?: boolean;
  webBaseDomain?: string;
  webScheme?: 'http' | 'https';
  vhostHTTPPort?: number;
  allowPortsText?: string;
};

function trimOptional(value?: string) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function formatPortRanges(ranges?: Array<{ from: number; to: number }>) {
  return ranges?.map((range) => `${range.from}-${range.to}`).join('\n') || '';
}

function parsePortRanges(value?: string) {
  const raw = value?.trim();
  if (!raw) return undefined;

  return raw.split(/[\n,]+/).map((part) => {
    const trimmed = part.trim();
    const match = trimmed.match(/^(\d+)(?:\s*-\s*(\d+))?$/);
    if (!match) {
      throw new Error('端口范围格式应为 18000-18999，每行或逗号分隔');
    }
    const from = Number(match[1]);
    const to = Number(match[2] || match[1]);
    if (from < 1024 || from > 65535 || to < 1024 || to > 65535 || from > to) {
      throw new Error('端口范围必须在 1024-65535 内，且起始端口不能大于结束端口');
    }
    return { from, to };
  });
}

function doctorStatusColor(status: string) {
  if (status === 'pass') return 'green';
  if (status === 'warn') return 'gold';
  if (status === 'fail') return 'red';
  return 'default';
}

function doctorStatusIcon(status: string) {
  if (status === 'pass') return <CheckCircleOutlined />;
  if (status === 'warn') return <ExclamationCircleOutlined />;
  if (status === 'fail') return <CloseCircleOutlined />;
  return <MinusCircleOutlined />;
}

export default function ClientNodesPage() {
  const [open, setOpen] = useState(false);
  const [editingNode, setEditingNode] = useState<LocalNode | null>(null);
  const [doctorOpen, setDoctorOpen] = useState(false);
  const [doctorResult, setDoctorResult] = useState<NodeDoctorResult | null>(null);
  const [form] = Form.useForm<NodeFormValues>();
  const { data: nodes = [], isLoading: nodesLoading } = useLocalNodes();
  const { data: rules = [] } = usePortRules();
  const createNode = useCreateNode();
  const updateNode = useUpdateNode();
  const deleteNode = useDeleteNode();
  const doctorNode = useDoctorNode();

  const ruleStats = useMemo(() => {
    const stats = new Map<string, { total: number; enabled: number }>();
    for (const rule of rules) {
      const current = stats.get(rule.nodeId) || { total: 0, enabled: 0 };
      current.total += 1;
      if (rule.enabled) current.enabled += 1;
      stats.set(rule.nodeId, current);
    }
    return stats;
  }, [rules]);

  const openCreateModal = () => {
    setEditingNode(null);
    form.setFieldsValue({
      frpsPort: 7000,
      webScheme: 'https',
      vhostHTTPPort: 8080,
    });
    setOpen(true);
  };

  const openEditModal = (node: LocalNode) => {
    setEditingNode(node);
    form.setFieldsValue({
      id: node.id,
      name: node.name,
      serverAddr: node.serverAddr,
      frpsPort: node.frpsPort,
      authToken: '',
      clearAuthToken: false,
      webBaseDomain: node.webBaseDomain,
      webScheme: (node.webScheme as 'http' | 'https') || 'https',
      vhostHTTPPort: node.vhostHTTPPort || 8080,
      allowPortsText: formatPortRanges(node.allowPorts),
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingNode(null);
    form.resetFields();
  };

  const handleSubmit = async (values: NodeFormValues) => {
    try {
      const allowPorts = parsePortRanges(values.allowPortsText);
      if (editingNode) {
        const body: UpdateNodeRequest = {
          name: values.name,
          serverAddr: values.serverAddr,
          frpsPort: values.frpsPort,
          authMethod: 'token',
          webBaseDomain: trimOptional(values.webBaseDomain),
          webScheme: values.webScheme || 'https',
          vhostHTTPPort: values.vhostHTTPPort || 8080,
          allowPorts,
          clearAuthToken: values.clearAuthToken,
        };
        const authToken = trimOptional(values.authToken);
        if (authToken) body.authToken = authToken;
        await updateNode.mutateAsync({ nodeId: editingNode.id, body });
        message.success('节点已更新');
      } else {
        const body: CreateNodeRequest = {
          id: trimOptional(values.id),
          name: values.name,
          serverAddr: values.serverAddr,
          frpsPort: values.frpsPort,
          authMethod: 'token',
          authToken: trimOptional(values.authToken),
          webBaseDomain: trimOptional(values.webBaseDomain),
          webScheme: values.webScheme || 'https',
          vhostHTTPPort: values.vhostHTTPPort || 8080,
          allowPorts,
        };
        await createNode.mutateAsync(body);
        message.success('节点已创建');
      }
      closeModal();
    } catch (error) {
      message.error(error instanceof Error ? error.message : getApiErrorMessage(error, editingNode ? '更新节点失败' : '创建节点失败'));
    }
  };

  const handleDelete = async (node: LocalNode) => {
    try {
      await deleteNode.mutateAsync(node.id);
      message.success('节点已删除');
    } catch (error) {
      message.error(getApiErrorMessage(error, '删除节点失败'));
    }
  };

  const handleDoctor = async (node: LocalNode) => {
    setDoctorOpen(true);
    setDoctorResult(null);
    try {
      const result = await doctorNode.mutateAsync(node.id);
      setDoctorResult(result);
    } catch (error) {
      message.error(getApiErrorMessage(error, '节点诊断失败'));
      setDoctorOpen(false);
    }
  };

  const columns: ColumnsType<LocalNode> = [
    {
      title: '节点',
      key: 'node',
      render: (_, node) => (
        <Space>
          <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 18, borderRadius: 12 }}>
            <DatabaseOutlined />
          </div>
          <div>
            <div style={{ fontWeight: 600 }}>{node.name}</div>
            <Text type="secondary" style={{ fontSize: 12 }}>{node.id}</Text>
          </div>
        </Space>
      ),
    },
    {
      title: 'frps 地址',
      key: 'server',
      render: (_, node) => <Text code>{node.serverAddr}:{node.frpsPort}</Text>,
    },
    {
      title: 'Web 域名',
      key: 'web',
      render: (_, node) => node.webBaseDomain ? (
        <Text code>{node.webScheme || 'https'}://*.{node.webBaseDomain}</Text>
      ) : (
        <Text type="secondary">未配置</Text>
      ),
    },
    {
      title: 'Token',
      key: 'token',
      render: (_, node) => (
        <Tag color={node.authTokenSet ? 'green' : 'red'} icon={<KeyOutlined />}>
          {node.authTokenSet ? '已配置' : '未配置'}
        </Tag>
      ),
    },
    {
      title: '允许端口',
      key: 'allowPorts',
      render: (_, node) => node.allowPorts?.length ? (
        <Space size={4} wrap>
          {node.allowPorts.map((range) => (
            <Tag key={`${range.from}-${range.to}`} color="blue">{range.from}-{range.to}</Tag>
          ))}
        </Space>
      ) : (
        <Text type="secondary">不限</Text>
      ),
    },
    {
      title: '规则',
      key: 'rules',
      render: (_, node) => {
        const stats = ruleStats.get(node.id) || { total: 0, enabled: 0 };
        return <Text>{stats.enabled} 启用 / {stats.total} 总数</Text>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 260,
      render: (_, node) => {
        const stats = ruleStats.get(node.id) || { total: 0, enabled: 0 };
        const deleteDisabled = node.id === 'default' || stats.total > 0;
        return (
          <Space>
            <Button type="text" icon={<SearchOutlined />} onClick={() => handleDoctor(node)} loading={doctorNode.isPending}>
              诊断
            </Button>
            <Button type="text" icon={<EditOutlined />} onClick={() => openEditModal(node)}>
              编辑
            </Button>
            <Popconfirm
              title="确认删除节点？"
              description={deleteDisabled ? '默认节点或仍被规则使用的节点不能删除。' : '删除后无法恢复。'}
              onConfirm={() => handleDelete(node)}
              disabled={deleteDisabled}
            >
              <Button type="text" danger icon={<DeleteOutlined />} disabled={deleteDisabled} loading={deleteNode.isPending}>
                删除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <Title level={2} style={{ marginBottom: 24 }}>
        <span className="gradient-text">frps 节点</span>
      </Title>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        message="当前版本支持多个 frps 节点同时在线"
        description="后端会按节点生成独立 frpc 配置和进程，每个节点有自己的运行状态、admin 端口和日志文件。"
      />

      <Card
        style={{
          marginBottom: 24,
          borderRadius: 24,
          background: '#0f172a',
          border: 'none',
        }}
        bodyStyle={{ padding: 24 }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 24 }}>
          <Space size={36} wrap>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>节点数量</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#fff', marginTop: 4 }}>{nodes.length}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>Token 完整</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#34d399', marginTop: 4 }}>
                {nodes.filter((node) => node.authTokenSet).length}
              </div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>Web 节点</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#22d3ee', marginTop: 4 }}>
                {nodes.filter((node) => node.webBaseDomain).length}
              </div>
            </div>
          </Space>
          <Button
            type="primary"
            size="large"
            icon={<PlusOutlined />}
            onClick={openCreateModal}
            style={{
              height: 48,
              padding: '0 28px',
              borderRadius: 12,
              fontSize: 15,
              fontWeight: 600,
              background: '#fff',
              color: '#0f172a',
              border: 'none',
              boxShadow: 'none',
              flexShrink: 0,
            }}
          >
            添加节点
          </Button>
        </div>
      </Card>

      <Card style={{ borderRadius: 24 }}>
        <Table rowKey="id" columns={columns} dataSource={nodes} loading={nodesLoading} pagination={false} />
      </Card>

      <Modal
        title={
          <Space size={12}>
            <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 20 }}>
              <GlobalOutlined />
            </div>
            <span style={{ fontSize: 20, fontWeight: 700 }}>{editingNode ? '编辑 frps 节点' : '添加 frps 节点'}</span>
          </Space>
        }
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={createNode.isPending || updateNode.isPending}
        width={640}
        okText={editingNode ? '保存节点' : '创建节点'}
        styles={{ body: { padding: '24px 0' } }}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} size="large">
          <Form.Item
            name="id"
            label="节点 ID"
            rules={[
              { pattern: /^[A-Za-z0-9][A-Za-z0-9_-]{1,63}$/, message: '2-64 位，只能包含字母、数字、_ 或 -' },
            ]}
            extra={editingNode ? '节点 ID 创建后不可修改。' : '可选。留空时后端会根据名称自动生成。'}
          >
            <Input disabled={Boolean(editingNode)} placeholder="例如 default、ma1、backup-cn" />
          </Form.Item>

          <Form.Item name="name" label="节点名称" rules={[{ required: true, message: '请输入节点名称' }]}>
            <Input placeholder="例如 美国节点 1" />
          </Form.Item>

          <Form.Item name="serverAddr" label="frps 地址" rules={[{ required: true, message: '请输入 frps 地址' }]}>
            <Input placeholder="149.118.158.112 或 frps.example.com" />
          </Form.Item>

          <Form.Item name="frpsPort" label="frps 端口" rules={[{ required: true, message: '请输入 frps 端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="7000" />
          </Form.Item>

          <Form.Item
            name="authToken"
            label="Auth Token"
            rules={editingNode ? [] : [{ required: true, message: '请输入 frps token' }]}
            extra={editingNode ? '留空表示保留当前 token。' : undefined}
          >
            <Input.Password placeholder={editingNode ? '留空保留当前 token' : 'frps auth.token'} />
          </Form.Item>

          {editingNode && (
            <Form.Item name="clearAuthToken" label="清空 Token" valuePropName="checked">
              <Switch />
            </Form.Item>
          )}

          <Form.Item name="webBaseDomain" label="Web 基础域名" extra="用于 HTTP/HTTPS 网站暴露，例如 ma1.gameuniverse.top。">
            <Input placeholder="ma1.gameuniverse.top" />
          </Form.Item>

          <Form.Item name="webScheme" label="公网 Web 协议" initialValue="https">
            <Select
              options={[
                { label: 'HTTPS', value: 'https' },
                { label: 'HTTP', value: 'http' },
              ]}
            />
          </Form.Item>

          <Form.Item name="vhostHTTPPort" label="frps vhost HTTP 端口" initialValue={8080}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="8080" />
          </Form.Item>

          <Form.Item
            name="allowPortsText"
            label="TCP/UDP 远程端口允许范围"
            extra="可选。每行一个范围，例如 18000-18999；留空表示不限制。"
          >
            <Input.TextArea rows={3} placeholder={'18000-18999\n20000-20010'} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          <Space size={12}>
            <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 20 }}>
              <SearchOutlined />
            </div>
            <span style={{ fontSize: 20, fontWeight: 700 }}>节点诊断</span>
          </Space>
        }
        open={doctorOpen}
        onCancel={() => setDoctorOpen(false)}
        footer={null}
        width={760}
      >
        {doctorResult ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Alert
              type={doctorResult.overall === 'pass' ? 'success' : doctorResult.overall === 'warn' ? 'warning' : 'error'}
              showIcon
              message={`${doctorResult.node.name}：${doctorResult.overall}`}
              description={doctorResult.testedDomain ? `测试域名：${doctorResult.testedDomain}` : undefined}
            />
            <Table
              size="small"
              rowKey="id"
              pagination={false}
              dataSource={doctorResult.checks}
              columns={[
                {
                  title: '检查项',
                  dataIndex: 'name',
                  key: 'name',
                  width: 150,
                },
                {
                  title: '结果',
                  dataIndex: 'status',
                  key: 'status',
                  width: 110,
                  render: (status: string) => (
                    <Tag color={doctorStatusColor(status)} icon={doctorStatusIcon(status)}>
                      {status}
                    </Tag>
                  ),
                },
                {
                  title: '耗时',
                  dataIndex: 'durationMs',
                  key: 'durationMs',
                  width: 90,
                  render: (durationMs?: number) => durationMs ? `${durationMs} ms` : '-',
                },
                {
                  title: '说明',
                  dataIndex: 'message',
                  key: 'message',
                  render: (messageText: string) => <Text>{messageText}</Text>,
                },
              ]}
            />
          </Space>
        ) : (
          <Alert type="info" showIcon message="正在诊断节点" description="会检查 frps TCP 连通性、临时 frpc 登录、通配 DNS 和 Web 入口。" />
        )}
      </Modal>
    </div>
  );
}
