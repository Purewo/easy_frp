import { Alert, Button, Card, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

const { Title, Paragraph } = Typography;

export default function GroupJoinPage() {
  const navigate = useNavigate();

  return (
    <Card style={{ maxWidth: 720, borderRadius: 20 }}>
      <Title level={3}>分组流程已停用</Title>
      <Paragraph>
        当前前端改为本地直连端口暴露，不再使用 group/device token 加入分组流程。
      </Paragraph>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        message="请在端口暴露页面创建 TCP/UDP 规则。"
      />
      <Button type="primary" onClick={() => navigate('/ports')}>返回端口暴露</Button>
    </Card>
  );
}
