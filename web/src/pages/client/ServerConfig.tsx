import { Alert, Button, Card, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

const { Title, Paragraph } = Typography;

export default function ServerConfigPage() {
  const navigate = useNavigate();

  return (
    <Card style={{ maxWidth: 720, borderRadius: 20 }}>
      <Title level={3}>服务器配置已迁移</Title>
      <Paragraph>
        当前版本由本地后端启动参数配置 frps 地址、端口和 token，前端不再提供单独的服务器配置表单。
      </Paragraph>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        message="请使用 README 中的 client 启动命令配置本地后端。"
      />
      <Button type="primary" onClick={() => navigate('/ports')}>返回端口暴露</Button>
    </Card>
  );
}
