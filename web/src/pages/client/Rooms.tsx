import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Descriptions,
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
  KeyOutlined,
  PlusOutlined,
  RadarChartOutlined,
  SearchOutlined,
  ShareAltOutlined,
} from '@ant-design/icons';
import {
  getApiErrorMessage,
  type ClientCreateRoomHostRequest,
  type ClientJoinRoomRequest,
  type ClientRoomDoctorResult,
  type ClientRoomRuleStatus,
  type ClientRoomRuleView,
  type NatHoleDiscoverResult,
  type TunnelProtocol,
  useClientRoomStatuses,
  useCreateClientRoomHost,
  useDeleteClientRoomRule,
  useDiscoverNatHole,
  useDoctorClientRoomRule,
  useJoinClientRoom,
  useNetworkInterfaces,
  usePatchClientRoomRule,
} from '../../hooks/useClientApi';
import { type RoomView, useListRooms } from '../../hooks/useControlApi';

const { Title, Text, Paragraph } = Typography;

type HostFormValues = {
  name: string;
  deviceName?: string;
  serverBaseURL?: string;
  tunnelProtocol: TunnelProtocol;
  natHoleStunServer?: string;
  localIP?: string;
  localPort: number;
  enabled?: boolean;
};

type JoinFormValues = {
  roomCode: string;
  name?: string;
  deviceName?: string;
  serverBaseURL?: string;
  tunnelProtocol: TunnelProtocol;
  natHoleStunServer?: string;
  bindAddr?: string;
  bindPort: number;
  enabled?: boolean;
};

type NatFormValues = {
  stunServer?: string;
  localAddr?: string;
};

function optionalText(value?: string) {
  const trimmed = value?.trim();
  return trimmed || undefined;
}

function localEndpoint(rule: ClientRoomRuleView) {
  if (rule.role === 'host') return `${rule.localIP || '127.0.0.1'}:${rule.localPort || '-'}`;
  return `${rule.bindAddr || '127.0.0.1'}:${rule.bindPort || '-'}`;
}

function protocolColor(protocol: TunnelProtocol) {
  return protocol === 'xtcp' ? 'purple' : 'blue';
}

function remoteEndpoint(room: RoomView) {
  return `${room.frpsAddr}:${room.frpsPort}`;
}

export default function RoomsPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [joinOpen, setJoinOpen] = useState(false);
  const [natOpen, setNatOpen] = useState(false);
  const [createdRule, setCreatedRule] = useState<ClientRoomRuleView | null>(null);
  const [doctorResult, setDoctorResult] = useState<ClientRoomDoctorResult | null>(null);
  const [natResult, setNatResult] = useState<NatHoleDiscoverResult | null>(null);
  const [hostForm] = Form.useForm<HostFormValues>();
  const [joinForm] = Form.useForm<JoinFormValues>();
  const [natForm] = Form.useForm<NatFormValues>();

  const { data: remoteRooms = [], isLoading: remoteRoomsLoading } = useListRooms();
  const { data: statuses = [], isLoading } = useClientRoomStatuses();
  const { data: networkInterfaces = [] } = useNetworkInterfaces();
  const createHost = useCreateClientRoomHost();
  const joinRoom = useJoinClientRoom();
  const patchRule = usePatchClientRoomRule();
  const deleteRule = useDeleteClientRoomRule();
  const doctorRule = useDoctorClientRoomRule();
  const discoverNat = useDiscoverNatHole();

  const rules = statuses.map((item) => item.rule);
  const runningCount = statuses.filter((item) => item.process.running).length;
  const xtcpCount = rules.filter((rule) => rule.tunnelProtocol === 'xtcp').length;
  const stcpCount = rules.filter((rule) => rule.tunnelProtocol === 'stcp').length;
  const remoteEnabledCount = remoteRooms.filter((room) => room.enabled).length;

  const closeCreate = () => {
    setCreateOpen(false);
    hostForm.resetFields();
  };

  const closeJoin = () => {
    setJoinOpen(false);
    joinForm.resetFields();
  };

  const buildHostBody = (values: HostFormValues): ClientCreateRoomHostRequest => ({
    name: values.name.trim(),
    deviceName: optionalText(values.deviceName),
    serverBaseURL: optionalText(values.serverBaseURL),
    tunnelProtocol: values.tunnelProtocol || 'xtcp',
    natHoleStunServer: optionalText(values.natHoleStunServer),
    localIP: optionalText(values.localIP) || '127.0.0.1',
    localPort: values.localPort,
    enabled: values.enabled ?? true,
  });

  const buildJoinBody = (values: JoinFormValues): ClientJoinRoomRequest => ({
    roomCode: values.roomCode.trim(),
    name: optionalText(values.name),
    deviceName: optionalText(values.deviceName),
    serverBaseURL: optionalText(values.serverBaseURL),
    tunnelProtocol: values.tunnelProtocol || 'xtcp',
    natHoleStunServer: optionalText(values.natHoleStunServer),
    bindAddr: optionalText(values.bindAddr) || '127.0.0.1',
    bindPort: values.bindPort,
    enabled: values.enabled ?? true,
  });

  const handleCreate = async (values: HostFormValues) => {
    try {
      const rule = await createHost.mutateAsync(buildHostBody(values));
      setCreatedRule(rule);
      closeCreate();
      message.success('本地主机房间已创建并应用');
    } catch (error) {
      message.error(getApiErrorMessage(error, '创建主机房间失败'));
    }
  };

  const handleJoin = async (values: JoinFormValues) => {
    try {
      const rule = await joinRoom.mutateAsync(buildJoinBody(values));
      setCreatedRule(rule);
      closeJoin();
      message.success('已加入房间并应用本地 visitor 规则');
    } catch (error) {
      message.error(getApiErrorMessage(error, '加入房间失败'));
    }
  };

  const handleToggle = async (rule: ClientRoomRuleView, enabled: boolean) => {
    try {
      await patchRule.mutateAsync({ roomRuleId: rule.id, body: { enabled } });
      message.success(enabled ? '房间规则已启用' : '房间规则已停用');
    } catch (error) {
      message.error(getApiErrorMessage(error, '房间规则状态更新失败'));
    }
  };

  const handleDelete = async (rule: ClientRoomRuleView) => {
    try {
      await deleteRule.mutateAsync(rule.id);
      message.success('本地房间规则已删除并重新应用 frpc');
    } catch (error) {
      message.error(getApiErrorMessage(error, '删除房间规则失败'));
    }
  };

  const handleDoctor = async (rule: ClientRoomRuleView) => {
    try {
      setDoctorResult(await doctorRule.mutateAsync(rule.id));
    } catch (error) {
      message.error(getApiErrorMessage(error, '房间诊断失败'));
    }
  };

  const handleNatDiscover = async (values: NatFormValues) => {
    try {
      setNatResult(await discoverNat.mutateAsync({
        stunServer: optionalText(values.stunServer),
        localAddr: optionalText(values.localAddr),
      }));
    } catch (error) {
      message.error(getApiErrorMessage(error, 'NAT 探测失败'));
    }
  };

  const columns: ColumnsType<ClientRoomRuleStatus> = [
    {
      title: '房间规则',
      key: 'name',
      render: (_, item) => (
        <Space>
          <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 18 }}>
            <ShareAltOutlined />
          </div>
          <div>
            <div style={{ fontWeight: 700 }}>{item.rule.name}</div>
            <Text type="secondary" style={{ fontSize: 12 }}>{item.rule.id}</Text>
          </div>
        </Space>
      ),
    },
    {
      title: '角色/协议',
      key: 'role',
      render: (_, item) => (
        <Space>
          <Tag color={item.rule.role === 'host' ? 'green' : 'cyan'}>{item.rule.role}</Tag>
          <Tag color={protocolColor(item.rule.tunnelProtocol)}>{item.rule.tunnelProtocol.toUpperCase()}</Tag>
        </Space>
      ),
    },
    {
      title: '本地端点',
      key: 'localEndpoint',
      render: (_, item) => <Text code>{item.process.localEndpoint || localEndpoint(item.rule)}</Text>,
    },
    {
      title: '房间 frps',
      key: 'server',
      render: (_, item) => <Text code>{item.rule.serverAddr}:{item.rule.serverPort}</Text>,
    },
    {
      title: '进程',
      key: 'process',
      render: (_, item) => item.process.running ? <Tag color="success">运行中 {item.process.pid}</Tag> : <Tag color="default">未运行</Tag>,
    },
    {
      title: '启用',
      key: 'enabled',
      render: (_, item) => (
        <Switch checked={item.rule.enabled} loading={patchRule.isPending} onChange={(checked) => handleToggle(item.rule, checked)} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 210,
      render: (_, item) => (
        <Space>
          <Button type="text" icon={<SearchOutlined />} loading={doctorRule.isPending} onClick={() => handleDoctor(item.rule)}>
            诊断
          </Button>
          <Popconfirm title="确认删除本地房间规则？" description="删除后会立即重新应用 frpc 配置" onConfirm={() => handleDelete(item.rule)}>
            <Button type="text" danger icon={<DeleteOutlined />} loading={deleteRule.isPending}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const remoteColumns: ColumnsType<RoomView> = [
    {
      title: '远端房间',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, room) => (
        <Space>
          <div className="icon-bg icon-bg-success" style={{ width: 40, height: 40, fontSize: 18 }}>
            <ShareAltOutlined />
          </div>
          <div>
            <div style={{ fontWeight: 700 }}>{name}</div>
            <Text type="secondary" style={{ fontSize: 12 }}>{room.id}</Text>
          </div>
        </Space>
      ),
    },
    {
      title: '服务名',
      dataIndex: 'serverName',
      key: 'serverName',
      render: (value: string) => <Text code>{value}</Text>,
    },
    {
      title: '房间 frps',
      key: 'endpoint',
      render: (_, room) => <Text code>{remoteEndpoint(room)}</Text>,
    },
    {
      title: '成员',
      dataIndex: 'memberCount',
      key: 'memberCount',
      render: (count: number) => <Tag color="cyan">{count}</Tag>,
    },
    {
      title: '服务端状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) => <Tag color={enabled ? 'success' : 'default'}>{enabled ? 'enabled' : 'disabled'}</Tag>,
    },
  ];

  return (
    <div>
      <Title level={2} style={{ marginBottom: 24 }}>
        <span className="gradient-text">XTCP / STCP 房间</span>
      </Title>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        title="Room mode 使用双数据面"
        description="远端房间记录来自 http://149.118.158.112:18080/v1/rooms，本机运行规则来自 127.0.0.1:7410/v1/client/rooms/status。远端有房间但本地为空是正常状态，表示当前机器还没有 host/join 规则。"
      />

      <Card style={{ marginBottom: 24, borderRadius: 24, background: '#07111f', border: 'none' }} styles={{ body: { padding: 24 } }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 24 }}>
          <Space size={36} wrap>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>远端房间</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#fff', marginTop: 4 }}>{remoteRooms.length}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>服务端启用</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#fbbf24', marginTop: 4 }}>{remoteEnabledCount}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>运行中</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#34d399', marginTop: 4 }}>{runningCount}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>XTCP</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#c084fc', marginTop: 4 }}>{xtcpCount}</div>
            </div>
            <div>
              <Text style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14 }}>STCP</Text>
              <div style={{ fontSize: 36, fontWeight: 700, color: '#22d3ee', marginTop: 4 }}>{stcpCount}</div>
            </div>
          </Space>
          <Space wrap>
            <Button size="large" icon={<RadarChartOutlined />} onClick={() => setNatOpen(true)}>
              NAT 探测
            </Button>
            <Button size="large" icon={<KeyOutlined />} onClick={() => setJoinOpen(true)}>
              加入房间
            </Button>
            <Button type="primary" size="large" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
              创建主机房间
            </Button>
          </Space>
        </div>
      </Card>

      <Card title="远端房间记录" style={{ borderRadius: 24, marginBottom: 24 }}>
        <Paragraph type="secondary" style={{ marginTop: -8 }}>
          这是控制服务器上的全局房间元数据，不代表当前机器已经启动了本地 frpc。需要房间码才能在本机加入 visitor 规则。
        </Paragraph>
        <Table rowKey="id" columns={remoteColumns} dataSource={remoteRooms} loading={remoteRoomsLoading} pagination={false} />
      </Card>

      <Card style={{ borderRadius: 24 }}>
        <Title level={4} style={{ marginTop: 0, marginBottom: 16 }}>本机房间规则与进程</Title>
        <Table rowKey={(item) => item.rule.id} columns={columns} dataSource={statuses} loading={isLoading} pagination={false} />
      </Card>

      <Modal title="创建主机房间" open={createOpen} onCancel={closeCreate} onOk={() => hostForm.submit()} confirmLoading={createHost.isPending} okText="创建并启动" width={680}>
        <Form form={hostForm} layout="vertical" size="large" onFinish={handleCreate} initialValues={{ localIP: '127.0.0.1', tunnelProtocol: 'xtcp', enabled: true }}>
          <Form.Item name="name" label="房间名称" rules={[{ required: true, message: '请输入房间名称' }, { max: 80 }]}>
            <Input placeholder="例如 private-api" />
          </Form.Item>
          <Form.Item name="deviceName" label="本机设备名">
            <Input placeholder="可选，例如 alice-laptop" />
          </Form.Item>
          <Form.Item name="serverBaseURL" label="控制服务器地址">
            <Input placeholder="可选；留空使用内置远端 http://149.118.158.112:18080" />
          </Form.Item>
          <Form.Item name="tunnelProtocol" label="隧道协议" rules={[{ required: true }]}>
            <Select options={[{ label: 'XTCP NAT 穿透', value: 'xtcp' }, { label: 'STCP 可靠回退', value: 'stcp' }]} />
          </Form.Item>
          <Form.Item name="natHoleStunServer" label="STUN 服务器">
            <Input placeholder="可选，例如 stun.easyvoip.com:3478" />
          </Form.Item>
          <Form.Item name="localIP" label="本地服务 IP">
            <Input placeholder="127.0.0.1 或 localhost" />
          </Form.Item>
          <Form.Item name="localPort" label="本地服务端口" rules={[{ required: true, message: '请输入本地端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="例如 8080" />
          </Form.Item>
          <Form.Item name="enabled" label="立即启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="加入房间" open={joinOpen} onCancel={closeJoin} onOk={() => joinForm.submit()} confirmLoading={joinRoom.isPending} okText="加入并启动" width={680}>
        <Form form={joinForm} layout="vertical" size="large" onFinish={handleJoin} initialValues={{ bindAddr: '127.0.0.1', tunnelProtocol: 'xtcp', enabled: true }}>
          <Form.Item name="roomCode" label="房间码" rules={[{ required: true, message: '请输入房间码' }]}>
            <Input.Password placeholder="room_abc.secret" />
          </Form.Item>
          <Form.Item name="name" label="本地规则名称">
            <Input placeholder="可选，留空使用房间名称" />
          </Form.Item>
          <Form.Item name="deviceName" label="本机设备名">
            <Input placeholder="可选，例如 bob-desktop" />
          </Form.Item>
          <Form.Item name="serverBaseURL" label="控制服务器地址">
            <Input placeholder="可选；留空使用内置远端 http://149.118.158.112:18080" />
          </Form.Item>
          <Form.Item name="tunnelProtocol" label="隧道协议" rules={[{ required: true }]}>
            <Select options={[{ label: 'XTCP NAT 穿透', value: 'xtcp' }, { label: 'STCP 可靠回退', value: 'stcp' }]} />
          </Form.Item>
          <Form.Item name="natHoleStunServer" label="STUN 服务器">
            <Input placeholder="可选，例如 stun.easyvoip.com:3478" />
          </Form.Item>
          <Form.Item name="bindAddr" label="本地访问地址">
            <Input placeholder="127.0.0.1" />
          </Form.Item>
          <Form.Item name="bindPort" label="本地访问端口" rules={[{ required: true, message: '请输入绑定端口' }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="例如 18080" />
          </Form.Item>
          <Form.Item name="enabled" label="立即启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="房间交接信息" open={!!createdRule} onCancel={() => setCreatedRule(null)} footer={null} width={720}>
        {createdRule && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {createdRule.roomCode && (
              <Alert
                type="warning"
                showIcon
                title="房间码只显示一次"
                description={
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Text code copyable={{ text: createdRule.roomCode }}>{createdRule.roomCode}</Text>
                    <Text type="secondary">请立即交给访问端，不要写入日志、URL 或埋点。</Text>
                  </Space>
                }
              />
            )}
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="角色"><Tag color={createdRule.role === 'host' ? 'green' : 'cyan'}>{createdRule.role}</Tag></Descriptions.Item>
              <Descriptions.Item label="协议"><Tag color={protocolColor(createdRule.tunnelProtocol)}>{createdRule.tunnelProtocol.toUpperCase()}</Tag></Descriptions.Item>
              <Descriptions.Item label="房间 ID"><Text code copyable>{createdRule.roomId}</Text></Descriptions.Item>
              <Descriptions.Item label="服务名"><Text code copyable>{createdRule.serverName}</Text></Descriptions.Item>
              <Descriptions.Item label="本地端点"><Text code>{localEndpoint(createdRule)}</Text></Descriptions.Item>
            </Descriptions>
          </Space>
        )}
      </Modal>

      <Modal title="房间诊断" open={!!doctorResult} onCancel={() => setDoctorResult(null)} footer={null} width={760}>
        {doctorResult && (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Alert type={doctorResult.overall === 'pass' ? 'success' : doctorResult.overall === 'warn' ? 'warning' : 'error'} showIcon title={`诊断结果：${doctorResult.overall}`} />
            <Table
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={doctorResult.checks}
              columns={[
                { title: '检查项', dataIndex: 'name' },
                { title: '状态', dataIndex: 'status', render: (status: string) => <Tag color={status === 'pass' ? 'success' : status === 'warn' ? 'warning' : status === 'skipped' ? 'default' : 'error'}>{status}</Tag> },
                { title: '消息', dataIndex: 'message' },
                { title: '耗时', dataIndex: 'durationMs', render: (value?: number) => value ? `${value}ms` : '-' },
              ]}
            />
          </Space>
        )}
      </Modal>

      <Modal title="NAT Hole 探测" open={natOpen} onCancel={() => setNatOpen(false)} onOk={() => natForm.submit()} confirmLoading={discoverNat.isPending} okText="开始探测" width={720}>
        <Form form={natForm} layout="vertical" size="large" onFinish={handleNatDiscover}>
          <Form.Item name="stunServer" label="STUN 服务器">
            <Input placeholder="可选，例如 stun.easyvoip.com:3478" />
          </Form.Item>
          <Form.Item name="localAddr" label="诊断 UDP 绑定地址">
            <Select
              allowClear
              showSearch
              placeholder="可选，例如 10.7.24.208:0"
              options={networkInterfaces.map((item) => ({ label: `${item.name} - ${item.address}:0${item.loopback ? ' (loopback)' : ''}`, value: `${item.address}:0` }))}
            />
          </Form.Item>
        </Form>
        {natResult && (
          <Card size="small" style={{ marginTop: 16 }}>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="结果"><Tag color={natResult.success ? 'success' : 'error'}>{natResult.success ? 'success' : 'failed'}</Tag></Descriptions.Item>
              <Descriptions.Item label="NAT 类型">{natResult.natType || '-'}</Descriptions.Item>
              <Descriptions.Item label="行为">{natResult.behavior || '-'}</Descriptions.Item>
              <Descriptions.Item label="外部地址">{natResult.externalAddresses?.join(', ') || '-'}</Descriptions.Item>
              <Descriptions.Item label="本地地址">{natResult.localAddress || '-'}</Descriptions.Item>
              <Descriptions.Item label="公网网络">{natResult.publicNetwork === undefined ? '-' : natResult.publicNetwork ? '是' : '否'}</Descriptions.Item>
              <Descriptions.Item label="耗时">{natResult.durationMs ? `${natResult.durationMs}ms` : '-'}</Descriptions.Item>
            </Descriptions>
            <Paragraph style={{ marginTop: 12 }} copyable={{ text: natResult.rawOutput }}>
              <Text code>{natResult.rawOutput || natResult.error || '-'}</Text>
            </Paragraph>
          </Card>
        )}
      </Modal>
    </div>
  );
}
