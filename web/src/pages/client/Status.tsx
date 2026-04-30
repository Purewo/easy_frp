import { Card, Button, message, Statistic, Row, Col, Space, Progress, Typography, Table, Tag } from 'antd';
import { ReloadOutlined, ThunderboltOutlined, SettingOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { getApiErrorMessage, useClientFrpcReload, useClientFrpcStatus } from '../../hooks/useClientApi';

const { Title, Text } = Typography;

export default function StatusPage() {
  const { data: status } = useClientFrpcStatus();
  const reload = useClientFrpcReload();

  const handleReload = async () => {
    try {
      await reload.mutateAsync();
      message.success({
        content: 'frpc 配置已重载',
        style: { borderRadius: 16 },
      });
    } catch (error) {
      message.error(getApiErrorMessage(error, '重载失败'));
    }
  };

  const isRunning = status?.running;
  const nodeStatuses = status?.nodes || [];
  const runningCount = nodeStatuses.filter((node) => node.running).length;
  const processCount = nodeStatuses.length || (status ? 1 : 0);

  return (
    <div>
      <Title level={2} style={{ marginBottom: 24 }}>
        <span className="gradient-text">frpc 状态</span>
      </Title>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card
            style={{
              borderRadius: 20,
              background: '#0f172a',
              border: 'none',
            }}
          >
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <Space>
                <div
                  className="icon-bg"
                  style={{
                    width: 48,
                    height: 48,
                    fontSize: 22,
                    borderRadius: 12,
                    background: isRunning ? 'rgba(52, 211, 153, 0.12)' : 'rgba(248, 113, 113, 0.12)',
                    color: isRunning ? '#34d399' : '#f87171',
                  }}
                >
                  {isRunning ? <ThunderboltOutlined /> : <ExclamationCircleOutlined />}
                </div>
                <div>
                  <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: 1 }}>
                    运行状态
                  </Text>
                  <Title level={4} style={{ margin: 0, color: isRunning ? '#34d399' : '#f87171', fontSize: 22 }}>
                    {isRunning ? '运行中' : '已停止'}
                  </Title>
                </div>
              </Space>
              {isRunning && (
                <Progress
                  percent={100}
                  strokeColor="#34d399"
                  trailColor="rgba(52, 211, 153, 0.15)"
                  showInfo={false}
                  size="small"
                />
              )}
            </Space>
          </Card>
        </Col>

        <Col span={8}>
          <Card
            style={{
              borderRadius: 20,
              background: '#0f172a',
              border: 'none',
            }}
          >
            <Space>
              <div
                className="icon-bg"
                style={{
                  width: 48,
                  height: 48,
                  fontSize: 22,
                  borderRadius: 12,
                  background: 'rgba(129, 140, 248, 0.12)',
                  color: '#818cf8',
                }}
              >
                <SettingOutlined />
              </div>
              <div>
                <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: 1 }}>
                  frpc 进程
                </Text>
                <Title level={4} style={{ margin: 0, color: '#fff', fontSize: 22 }}>
                  {processCount > 1 ? `${runningCount}/${processCount}` : status?.pid ?? '-'}
                </Title>
              </div>
            </Space>
          </Card>
        </Col>

        <Col span={8}>
          <Card
            style={{
              borderRadius: 20,
              background: '#0f172a',
              border: 'none',
            }}
          >
            <Space>
              <div
                className="icon-bg"
                style={{
                  width: 48,
                  height: 48,
                  fontSize: 22,
                  borderRadius: 12,
                  background: 'rgba(251, 191, 36, 0.12)',
                  color: '#fbbf24',
                }}
              >
                <CheckCircleOutlined />
              </div>
              <div>
                <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: 1 }}>
                  配置路径
                </Text>
                <Text style={{ fontSize: 13, color: 'rgba(255,255,255,0.7)', fontFamily: 'monospace' }}>
                  {status?.configPath?.split(/[\\/]/).pop() || '-'}
                </Text>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>

      <Card
        style={{ borderRadius: 20 }}
        title={
          <Space>
            <div className="icon-bg icon-bg-primary" style={{ width: 36, height: 36, fontSize: 18 }}>
              <SettingOutlined />
            </div>
            <span style={{ fontSize: 16, fontWeight: 600 }}>控制面板</span>
          </Space>
        }
        extra={
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            loading={reload.isPending}
            onClick={handleReload}
            style={{
              borderRadius: 10,
              background: '#0f172a',
              border: 'none',
              boxShadow: 'none',
            }}
          >
            重载配置
          </Button>
        }
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {status?.lastError ? (
            <div
              style={{
                background: 'rgba(239, 68, 68, 0.06)',
                border: '1px solid rgba(239, 68, 68, 0.15)',
                borderRadius: 12,
                padding: '12px 16px',
                display: 'flex',
                alignItems: 'center',
                gap: 10,
              }}
            >
              <ExclamationCircleOutlined style={{ color: '#ef4444', fontSize: 18 }} />
              <Text type="danger" strong>错误：{status.lastError}</Text>
            </div>
          ) : (
            <div
              style={{
                background: 'rgba(52, 211, 153, 0.06)',
                border: '1px solid rgba(52, 211, 153, 0.15)',
                borderRadius: 12,
                padding: '12px 16px',
                display: 'flex',
                alignItems: 'center',
                gap: 10,
              }}
            >
              <CheckCircleOutlined style={{ color: '#34d399', fontSize: 18 }} />
              <Text type="success" strong>frpc 运行正常，无错误</Text>
            </div>
          )}

          {nodeStatuses.length > 0 && (
            <Table
              size="small"
              rowKey="nodeId"
              pagination={false}
              dataSource={nodeStatuses}
              columns={[
                {
                  title: '节点',
                  dataIndex: 'nodeId',
                  key: 'nodeId',
                  render: (nodeId: string) => <Text code>{nodeId}</Text>,
                },
                {
                  title: '状态',
                  dataIndex: 'running',
                  key: 'running',
                  render: (running: boolean) => (
                    <Tag color={running ? 'green' : 'red'}>{running ? '运行中' : '已停止'}</Tag>
                  ),
                },
                {
                  title: 'PID',
                  dataIndex: 'pid',
                  key: 'pid',
                  render: (pid?: number) => pid || '-',
                },
                {
                  title: '配置',
                  dataIndex: 'configPath',
                  key: 'configPath',
                  render: (configPath: string) => <Text code>{configPath?.split(/[\\/]/).pop() || '-'}</Text>,
                },
                {
                  title: '错误',
                  dataIndex: 'lastError',
                  key: 'lastError',
                  render: (lastError?: string) => lastError ? <Text type="danger">{lastError}</Text> : <Text type="secondary">-</Text>,
                },
              ]}
            />
          )}

          <Row gutter={[16, 16]}>
            <Col span={12}>
              <Card
                size="small"
                style={{
                  borderRadius: 12,
                  background: '#f8fafc',
                  border: '1px solid #e2e8f0',
                }}
              >
                <Statistic
                  title="自动刷新"
                  value="3 秒"
                  valueStyle={{ fontSize: 22, fontWeight: 700, color: '#0f172a' }}
                />
              </Card>
            </Col>
            <Col span={12}>
              <Card
                size="small"
                style={{
                  borderRadius: 12,
                  background: '#f8fafc',
                  border: '1px solid #e2e8f0',
                }}
              >
                <Statistic
                  title="连接状态"
                  value={isRunning ? '已连接' : '未连接'}
                  valueStyle={{
                    fontSize: 22,
                    fontWeight: 700,
                    color: isRunning ? '#34d399' : '#ef4444',
                  }}
                />
              </Card>
            </Col>
          </Row>
        </Space>
      </Card>
    </div>
  );
}
