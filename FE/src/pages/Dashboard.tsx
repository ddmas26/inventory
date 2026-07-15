import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, Spin } from 'antd';
import { ShoppingOutlined, HomeOutlined, StockOutlined } from '@ant-design/icons';
import { productsApi } from '../api/products';
import { inventoriesApi } from '../api/inventories';

export default function Dashboard() {
  const { data: productsRes, isLoading: loadingProducts } = useQuery({
    queryKey: ['products', 'count'],
    queryFn: () => productsApi.list(1, 1),
  });

  const { data: inventoriesRes, isLoading: loadingInventories } = useQuery({
    queryKey: ['inventories', 'count'],
    queryFn: () => inventoriesApi.list(1, 1),
  });

  if (loadingProducts || loadingInventories) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;

  return (
    <Row gutter={[24, 24]}>
      <Col xs={24} sm={8}>
        <Card>
          <Statistic
            title="Total Products"
            value={productsRes?.total ?? 0}
            prefix={<ShoppingOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={8}>
        <Card>
          <Statistic
            title="Total Inventories"
            value={inventoriesRes?.total ?? 0}
            prefix={<HomeOutlined />}
          />
        </Card>
      </Col>
      <Col xs={24} sm={8}>
        <Card>
          <Statistic
            title="Stock Entries"
            value="—"
            prefix={<StockOutlined />}
          />
        </Card>
      </Col>
    </Row>
  );
}
