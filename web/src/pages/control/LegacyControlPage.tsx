import { Alert, Button, Card, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';

const { Title, Paragraph } = Typography;

type LegacyControlPageProps = {
  title: string;
};

export default function LegacyControlPage({ title }: LegacyControlPageProps) {
  const navigate = useNavigate();

  return (
    <Card style={{ maxWidth: 720, borderRadius: 20 }}>
      <Title level={3}>{title} 已迁移</Title>
      <Paragraph>
        后端已重塑为本地直连端口暴露流程，旧控制端页面暂不作为当前前端主流程使用。
      </Paragraph>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 24, borderRadius: 12 }}
        message="请使用端口暴露页面对接新的本地客户端 API。"
      />
      <Button type="primary" onClick={() => navigate('/ports')}>返回端口暴露</Button>
    </Card>
  );
}
