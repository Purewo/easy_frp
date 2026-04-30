import { Layout, Menu, Typography, Avatar, Badge, Space, Divider } from 'antd';
import {
  DatabaseOutlined,
  DesktopOutlined,
  FileTextOutlined,
  FireOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import type { ReactNode } from 'react';

const { Sider, Content, Header } = Layout;
const { Title, Text } = Typography;

const menuItems = [
  { key: '/ports', label: '端口暴露', icon: <ThunderboltOutlined /> },
  { key: '/nodes', label: 'frps 节点', icon: <DatabaseOutlined /> },
  { key: '/status', label: 'frpc 状态', icon: <PlayCircleOutlined /> },
  { key: '/logs', label: '运行日志', icon: <FileTextOutlined /> },
];

const pageMeta: Record<string, { title: string; subtitle: string; icon: ReactNode }> = {
  '/ports': { title: '端口暴露', subtitle: '管理本地 TCP/UDP 端口和 HTTPS Web 域名', icon: <ThunderboltOutlined /> },
  '/nodes': { title: 'frps 节点', subtitle: '管理可用的 frp 服务器和通配域名', icon: <DatabaseOutlined /> },
  '/status': { title: 'frpc 状态', subtitle: '查看进程状态并手动重载配置', icon: <PlayCircleOutlined /> },
  '/logs': { title: '运行日志', subtitle: '查看本地 frpc 最近输出', icon: <FileTextOutlined /> },
};

export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const meta = pageMeta[location.pathname] || pageMeta['/ports'];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="dark" collapsible width={240} style={{ padding: '24px 0 0' }}>
        <div style={{ padding: '0 20px 20px', display: 'flex', alignItems: 'center', gap: 12 }}>
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 10,
              background: '#6366f1',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 18,
              color: '#fff',
              boxShadow: '0 4px 12px rgba(99, 102, 241, 0.35)',
              flexShrink: 0,
            }}
          >
            <FireOutlined />
          </div>
          <div>
            <Title level={5} style={{ color: '#fff', margin: 0, fontWeight: 700, fontSize: 16, letterSpacing: 0.5 }}>
              FRP 管理器
            </Title>
            <Text style={{ color: 'rgba(255,255,255,0.35)', fontSize: 11 }}>Local Client</Text>
          </div>
        </div>

        <Divider style={{ margin: '0 16px 12px', borderColor: 'rgba(255,255,255,0.06)' }} />

        <div style={{ padding: '0 12px 16px' }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '10px 14px',
              borderRadius: 10,
              background: 'rgba(99, 102, 241, 0.1)',
              border: '1px solid rgba(99, 102, 241, 0.15)',
            }}
          >
            <div
              style={{
                width: 28,
                height: 28,
                borderRadius: 8,
                background: 'rgba(99, 102, 241, 0.18)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 14,
                color: '#818cf8',
              }}
            >
              <DesktopOutlined />
            </div>
            <div>
              <div style={{ color: 'rgba(255,255,255,0.9)', fontSize: 13, fontWeight: 600 }}>本地客户端</div>
              <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: 11 }}>直连 frpc 端口暴露</div>
            </div>
          </div>
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ background: 'transparent', border: 'none' }}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            background: '#fff',
            padding: '0 32px',
            height: 72,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Space size={16} align="center">
            <div className="icon-bg icon-bg-primary" style={{ width: 40, height: 40, fontSize: 20, flexShrink: 0 }}>
              {meta.icon}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
              <Title level={4} style={{ margin: 0, fontWeight: 700, lineHeight: '1.4' }}>
                {meta.title}
              </Title>
              <Text type="secondary" style={{ fontSize: 13, lineHeight: '1.4' }}>
                {meta.subtitle}
              </Text>
            </div>
          </Space>

          <Badge dot color="#22d3ee">
            <Avatar
              style={{
                background: 'linear-gradient(135deg, #22d3ee 0%, #06b6d4 100%)',
                cursor: 'pointer',
              }}
            >
              U
            </Avatar>
          </Badge>
        </Header>

        <Content style={{ margin: 24, padding: 32, background: '#fff', borderRadius: 24, boxShadow: '0 4px 20px rgba(0,0,0,0.05)' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
