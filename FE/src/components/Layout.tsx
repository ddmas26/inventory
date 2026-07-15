import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Typography } from 'antd';
import {
  DashboardOutlined,
  ShoppingOutlined,
  HomeOutlined,
  StockOutlined,
} from '@ant-design/icons';

const { Header, Sider, Content } = Layout;

const menuItems = [
  { key: '/', icon: <DashboardOutlined />, label: 'Dashboard' },
  { key: '/products', icon: <ShoppingOutlined />, label: 'Products' },
  { key: '/inventories', icon: <HomeOutlined />, label: 'Inventories' },
  { key: '/stock', icon: <StockOutlined />, label: 'Stock' },
];

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed}>
        <div style={{ height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Typography.Title level={4} style={{ color: '#fff', margin: 0 }}>
            {collapsed ? 'Inv' : 'Inventory'}
          </Typography.Title>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ background: '#fff', padding: '0 24px', fontSize: 18, fontWeight: 600 }}>
          {menuItems.find((m) => m.key === location.pathname)?.label || 'Inventory Management'}
        </Header>
        <Content style={{ margin: 24 }}>
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
