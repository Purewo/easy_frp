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
  DeleteOutlined,
  EditOutlined,
  GlobalOutlined,
  LinkOutlined,
  PlusOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  getApiErrorMessage,
  type CreatePortRuleRequest,
  type LocalNode,
  type PortProtocol,
  type PortRule,
  useCreatePortRule,
  useDeletePortRule,
  useLocalNodes,
  usePatchPortRule,
  usePortRules,
  useUpdatePortRule,
} from '../../hooks/useClientApi';

const { Title, Text, Paragraph } = Typography;

type PortRuleFormValues = CreatePortRuleRequest;

function optionalText(value?: string) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function publicURL(rule: PortRule, node?: LocalNode) {
  if (rule.protocol === 'http') {
    if (!rule.domain) return '-';
    return `${node?.webScheme || 'https'}://${rule.domain}`;
  }

  if (!rule.remotePort) return '-';
  return `${node?.serverAddr || rule.nodeId}:${rule.remotePort}`;
}

function protocolColor(protocol: PortProtocol) {
  if (protocol === 'http') return 'green';
  if (protocol === 'udp') return 'geekblue';
  return 'blue';
}

export default function ClientExposuresPage() {
  const [open, setOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<PortRule | null>(null);
  const [form] = Form.useForm<PortRuleFormValues>();
  const { data: nodes = [], isLoading: nodesLoading } = useLocalNodes();
  const { data: rules = [], isLoading: rulesLoading } = usePortRules();
  const createRule = useCreatePortRule();
  const updateRule = useUpdatePortRule();
  const patchRule = usePatchPortRule();
  const deleteRule = useDeletePortRule();

  const defaultNode = nodes[0];
  const selectedNode = Form.useWatch('nodeId', form) as string | undefined;
  const selectedProtocol = (Form.useWatch('protocol', form) as PortProtocol | undefined) || 'http';
  const watchedSubdomain = Form.useWatch('subdomain', form) as string | undefined;
  const watchedDomain = Form.useWatch('domain', form) as string | undefined;
  const nodeById = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const selectedNodeInfo = nodeById.get(selectedNode || defaultNode?.id || '');
  const createBlocked = nodes.length === 0 || nodes.every((node) => !node.authTokenSet);
  const activeCount = rules.filter((rule) => rule.enabled).length;
  const webCount = rules.filter((rule) => rule.protocol === 'http').length;

  const previewDomain = (() => {
    const explicitDomain = optionalText(watchedDomain);
    if (explicitDomain) return explicitDomain;

    const subdomain = optionalText(watchedSubdomain);
    if (subdomain && selectedNodeInfo?.webBaseDomain) {
      return `${subdomain}.${selectedNodeInfo.webBaseDomain}`;
    }

    return undefined;
  })();

  const openCreateModal = () => {
    setEditingRule(null);
    form.setFieldsValue({
      nodeId: defaultNode?.id || 'default',
      protocol: 'http',
      localIP: '127.0.0.1',
      enabled: true,
    });
    setOpen(true);
  };

  const openEditModal = (rule: PortRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      nodeId: rule.nodeId,
      name: rule.name,
      protocol: rule.protocol,
      localIP: rule.localIP,
      localPort: rule.localPort,
      remotePort: rule.remotePort,
      subdomain: rule.subdomain,
      domain: rule.domain,
      enabled: rule.enabled,
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingRule(null);
    form.resetFields();
  };

  const buildRequestBody = (values: PortRuleFormValues): CreatePortRuleRequest => {
    const protocol = values.protocol || 'http';
    const base: CreatePortRuleRequest = {
      nodeId: values.nodeId || defaultNode?.id || 'default',
      name: optionalText(values.name),
      protocol,
      localIP: optionalText(values.localIP) || '127.0.0.1',
      localPort: values.localPort,
      enabled: values.enabled ?? true,
    };

    if (protocol === 'http') {
      return {
        ...base,
        subdomain: optionalText(values.subdomain),
        domain: optionalText(values.domain),
      };
    }

    return {
      ...base,
      remotePort: values.remotePort,
    };
  };

  const handleSubmit = async (values: PortRuleFormValues) => {
    const body = buildRequestBody(values);

    try {
      if (editingRule) {
        await updateRule.mutateAsync({
          portId: editingRule.id,
          body: {
            ...body,
            enabled: body.enabled ?? true,
          },
        });
        message.success('规则已更新并应用');
      } else {
        await createRule.mutateAsync(body);
        message.success('规则已创建并应用');
      }
      closeModal();
    } catch (error) {
      message.error(getApiErrorMessage(error, editingRule ? '更新失败' : '创建失败'));
    }
  };

  const handleToggle = async (rule: PortRule, enabled: boolean) => {
    const otherActiveNodeIds = new Set(rules.filter((item) => item.enabled && item.id !== rule.id).map((item) => item.nodeId));
    if (enabled && otherActiveNodeIds.size > 0 && !otherActiveNodeIds.has(rule.nodeId)) {
      message.warning('当前已有其他 frps 节点的启用规则，请先停用后再切换节点');
      return;
    }

    try {
      await patchRule.mutateAsync({ portId: rule.id, enabled });
      message.success(enabled ? '规则已启用' : '规则已停用');
    } catch (error) {
      message.error(getApiErrorMessage(error, '状态更新失败'));
    }
  };

  const handleDelete = async (rule: PortRule) => {
    try {
      await deleteRule.mutateAsync(rule.id);
      message.success('规则已删除并应用');
    } catch (error) {
      message.error(getApiErrorMessage(error, '删除失败'));
    }
  };

  const columns: ColumnsType<PortRule> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, rule) => (
        <Space>
          <div
            className="icon-bg"
            style={{
              width: 40,
              height: 40,
              fontSize: 18,
              borderRadius: 12,
              background: rule.enabled ? 'rgba(52, 211, 153, 0.1)' : 'rgba(148, 163, 184, 0.14)',
              color: rule.enabled ? '#10b981' : '#64748b',
            }}
          >
            <ThunderboltOutlined />
          </div>
          <div>
            <div style={{ fontWeight: 600 }}>{name || `${rule.protocol}-${rule.localPort}`}</div>
            <div style={{ color: '#94a3b8', fontSize: 12 }}>{rule.id}</div>
          </div>
        </Space>
      ),
    },
    {
      title: '节点',
      dataIndex: 'nodeId',
      key: 'nodeId',
      render: (nodeId: string) => {
        const node = nodeById.get(nodeId);
        return <Tag color="default">{node?.name || nodeId}</Tag>;
      },
    },
    {
      title: '协议',
      dataIndex: 'protocol',
      key: 'protocol',
      render: (protocol: PortProtocol) => (
        <Tag color={protocolColor(protocol)} style={{ textTransform: 'uppercase' }}>
          {protocol === 'http' ? 'HTTP/HTTPS' : protocol}
        </Tag>
      ),
    },
    {
      title: '本地地址',
      key: 'local',
      render: (_, rule) => <Text code>{rule.localIP}:{rule.localPort}</Text>,
    },
    {
      title: '公网访问',
      key: 'remote',
      render: (_, rule) => {
        const node = nodeById.get(rule.nodeId);
        const value = publicURL(rule, node);
        return value === '-' ? <Text type="secondary">-</Text> : <Text code>{value}</Text>;
      },
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, rule) => (
        <Switch checked={enabled} loading={patchRule.isPending} onChange={(checked) => handleToggle(rule, checked)} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, rule) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => openEditModal(rule)}>
            编辑
          </Button>
          <Popconfirm title="确认删除？" description="删除后会立即应用新的 frpc 配置" onConfirm={() => handleDelete(rule)}>
            <Button type="text" danger icon={<DeleteOutlined />} loading={deleteRule.isPending}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Title level={2} style={{ marginBottom: 24 }}>
        <span className="gradient-text">端口暴露</span>
      </Title>

      {createBlocked && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 24, borderRadius: 12 }}
          message={nodes.length === 0 ? '未发现可用 frps 节点' : 'frps token 未配置'}
          description={nodes.length === 0
            ? '请确认本地后端已经启动，并成功初始化默认节点。'
            : '请使用 FRP_PANEL_FRPS_TOKEN 或 --frps-token 启动本地后端，否则无法创建端口规则。'}
        />
      )}

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
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>端口规则</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#fff', marginTop: 4 }}>{rules.length}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>启用中</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#34d399', marginTop: 4 }}>{activeCount}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>Web 域名</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#22d3ee', marginTop: 4 }}>{webCount}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>默认节点</Text>
              <div style={{ color: '#fff', marginTop: 10 }}>
                {defaultNode?.serverAddr || '-'}
                {defaultNode?.webBaseDomain ? ` / *.${defaultNode.webBaseDomain}` : ''}
              </div>
            </div>
          </Space>
          <Button
            type="primary"
            size="large"
            icon={<PlusOutlined />}
            onClick={openCreateModal}
            disabled={createBlocked}
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
            创建规则
          </Button>
        </div>
      </Card>

      <Card style={{ borderRadius: 24 }}>
        <Table rowKey="id" columns={columns} dataSource={rules} loading={rulesLoading || nodesLoading} pagination={false} />
      </Card>

      <Modal
        title={
          <Space size={12}>
            <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 20 }}>
              <GlobalOutlined />
            </div>
            <span style={{ fontSize: 20, fontWeight: 700 }}>{editingRule ? '编辑端口规则' : '创建端口规则'}</span>
          </Space>
        }
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={createRule.isPending || updateRule.isPending}
        width={640}
        okText={editingRule ? '保存并应用' : '创建并应用'}
        styles={{ body: { padding: '24px 0' } }}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} size="large">
          <Form.Item name="nodeId" label="frps 节点" rules={[{ required: true }]}>
            <Select
              placeholder="选择 frps 节点"
              options={nodes.map((node) => ({
                label: `${node.name} (${node.serverAddr}:${node.frpsPort}${node.authTokenSet ? '' : '，未配置 token'})`,
                value: node.id,
                disabled: !node.authTokenSet,
              }))}
            />
          </Form.Item>

          {selectedNodeInfo && !selectedNodeInfo.authTokenSet && (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16, borderRadius: 12 }}
              message="当前节点未配置 auth token，无法应用规则"
            />
          )}

          <Form.Item name="name" label="规则名称">
            <Input placeholder="可选，例如 path2agi、blog、dev-api" />
          </Form.Item>

          <Form.Item name="protocol" label="协议" rules={[{ required: true }]} initialValue="http">
            <Select
              options={[
                { label: 'HTTP 网站（公网 HTTPS 域名）', value: 'http' },
                { label: 'TCP 端口', value: 'tcp' },
                { label: 'UDP 端口', value: 'udp' },
              ]}
            />
          </Form.Item>

          <Form.Item name="localIP" label="本地 IP" initialValue="127.0.0.1">
            <Input placeholder="127.0.0.1 或 localhost" />
          </Form.Item>

          <Form.Item name="localPort" label="本地端口" rules={[{ required: true, message: '请输入本地端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="例如 8080" />
          </Form.Item>

          {selectedProtocol === 'http' ? (
            <>
              <Alert
                type="info"
                showIcon
                icon={<LinkOutlined />}
                style={{ marginBottom: 16, borderRadius: 12 }}
                message="HTTP 规则会通过服务器上的通配 HTTPS 域名访问"
                description="后端会自动重写回源 Host，Vite 等本地开发服务器通常不需要再手动配置 allowedHosts。"
              />

              {selectedNodeInfo && !selectedNodeInfo.webBaseDomain && (
                <Alert
                  type="warning"
                  showIcon
                  style={{ marginBottom: 16, borderRadius: 12 }}
                  message="当前节点没有配置 webBaseDomain，无法生成 HTTPS 域名"
                />
              )}

              <Form.Item
                name="subdomain"
                label="子域名"
                rules={[
                  {
                    pattern: /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/,
                    message: '只能使用小写字母、数字和中划线，且不能以中划线开头或结尾',
                  },
                ]}
                extra={selectedNodeInfo?.webBaseDomain ? `留空时后端会根据规则名自动生成；最终域名位于 *.${selectedNodeInfo.webBaseDomain}` : undefined}
              >
                <Input placeholder="例如 path2agi、blog、dev" />
              </Form.Item>

              <Form.Item
                name="domain"
                label="完整域名"
                extra="可选。填写后会覆盖子域名，必须属于当前节点的 webBaseDomain。"
              >
                <Input placeholder={selectedNodeInfo?.webBaseDomain ? `例如 blog.${selectedNodeInfo.webBaseDomain}` : '例如 blog.example.com'} />
              </Form.Item>

              <Paragraph style={{ marginTop: -8, marginBottom: 16 }}>
                <Text type="secondary">预览地址：</Text>{' '}
                <Text code>
                  {previewDomain
                    ? `${selectedNodeInfo?.webScheme || 'https'}://${previewDomain}`
                    : selectedNodeInfo?.webBaseDomain
                      ? `${selectedNodeInfo.webScheme || 'https'}://<子域名>.${selectedNodeInfo.webBaseDomain}`
                      : '-'}
                </Text>
              </Paragraph>
            </>
          ) : (
            <Form.Item
              name="remotePort"
              label="远程端口"
              rules={[{ required: true, message: '请输入远程端口' }]}
            >
              <InputNumber min={1024} max={65535} style={{ width: '100%' }} placeholder="例如 18080" />
            </Form.Item>
          )}

          <Form.Item name="enabled" label="立即启用" initialValue={true} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
