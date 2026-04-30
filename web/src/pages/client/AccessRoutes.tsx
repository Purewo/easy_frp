import { Alert, Button, Card, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

const { Title, Paragraph } = Typography;

export default function ClientAccessRoutesPage() {
  const navigate = useNavigate();

  return (
    <Card style={{ maxWidth: 720, borderRadius: 20 }}>
      <Title level={3}>访问路由已迁移</Title>
      <Paragraph>
        当前产品方向是本机 TCP/UDP 端口直接暴露，访问路由不再作为主流程展示。
      </Paragraph>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        message="请使用端口暴露页面管理本地映射规则。"
      />
      <Button type="primary" onClick={() => navigate('/ports')}>返回端口暴露</Button>
    </Card>
  );
}
