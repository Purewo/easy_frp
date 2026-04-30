import { Card, Spin, Typography, Space, Tag } from 'antd';
import { FileTextOutlined, SyncOutlined } from '@ant-design/icons';
import { useClientLogs } from '../../hooks/useClientApi';

const { Title } = Typography;

export default function LogsPage() {
  const { data: logs, isLoading } = useClientLogs();

  const getLogLevel = (line: string) => {
    if (line.includes('[E]') || line.includes('error') || line.includes('ERROR')) return 'error';
    if (line.includes('[W]') || line.includes('warn') || line.includes('WARN')) return 'warning';
    if (line.includes('[I]') || line.includes('info') || line.includes('INFO')) return 'info';
    if (line.includes('[D]') || line.includes('debug') || line.includes('DEBUG')) return 'processing';
    return 'default';
  };

  const coloredLogs = logs?.split('\n').map((line, index) => {
    const color = getLogLevel(line);
    const colorMap: Record<string, string> = {
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#22d3ee',
      processing: '#a855f7',
      default: '#6b7280',
    };
    return { line, color: colorMap[color], index };
  });

  return (
    <div>
      <Title level={2} style={{ marginBottom: 24 }}>
        <span className="gradient-text">运行日志</span>
      </Title>

      <Card
        style={{ borderRadius: 24 }}
        title={
          <Space size={12}>
            <div
              className="icon-bg"
              style={{
                width: 40,
                height: 40,
                fontSize: 20,
                borderRadius: 12,
                background: 'linear-gradient(135deg, rgba(34, 211, 238, 0.2) 0%, rgba(6, 182, 212, 0.2) 100%)',
                color: '#0891b2',
              }}
            >
              <FileTextOutlined />
            </div>
            <div>
              <span style={{ fontSize: 18, fontWeight: 600 }}>frpc 日志</span>
              <div style={{ fontSize: 12, color: '#6b7280' }}>
                <Tag icon={<SyncOutlined spin />} color="blue" style={{ borderRadius: 8 }}>
                  5 秒刷新
                </Tag>
              </div>
            </div>
          </Space>
        }
      >
        <Spin spinning={isLoading}>
          <div
            style={{
              background: '#1f2937',
              borderRadius: 16,
              padding: 16,
              maxHeight: 'calc(100vh - 300px)',
              overflow: 'auto',
            }}
          >
            {coloredLogs?.map(({ line, color, index }) => (
              <div
                key={index}
                style={{
                  fontFamily: 'monospace',
                  fontSize: 13,
                  color,
                  lineHeight: 1.6,
                  padding: '2px 0',
                  borderBottom: index < coloredLogs.length - 1 ? '1px solid #374151' : 'none',
                }}
              >
                {line || ' '}
              </div>
            ))}
            {!logs && (
              <div style={{ color: '#6b7280', textAlign: 'center', padding: 40 }}>
                <FileTextOutlined style={{ fontSize: 48, marginBottom: 16, opacity: 0.5 }} />
                <div>暂无日志数据</div>
              </div>
            )}
          </div>
        </Spin>
      </Card>
    </div>
  );
}
