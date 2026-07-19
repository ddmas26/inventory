import { useQuery } from '@tanstack/react-query';
import { Row, Col, Card, Statistic, Table, Tag, Spin, Empty } from 'antd';
import {
  DollarOutlined,
  ShoppingOutlined,
  HomeOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { dashboardApi } from '../api/dashboard';
import type { LowStockItem, DashboardLocation } from '../types';

// Fix default marker icon issue with bundlers
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';

delete (L.Icon.Default.prototype as any)._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

const lowStockColumns = [
  { title: 'Product', dataIndex: 'product_name', key: 'product' },
  { title: 'Inventory', dataIndex: 'inventory_name', key: 'inventory' },
  { title: 'Quantity', dataIndex: 'quantity', key: 'quantity' },
  {
    title: 'Status',
    dataIndex: 'stats',
    key: 'status',
    render: (stats: number) =>
      stats === 0 ? (
        <Tag color="error">Low Stock</Tag>
      ) : stats === 1 ? (
        <Tag color="warning">Normal</Tag>
      ) : (
        <Tag color="success">OK</Tag>
      ),
  },
];

// Component that auto-fits the map bounds to all markers on mount
function FitBoundsOnMount({ locations }: { locations: DashboardLocation[] }) {
  const map = useMap();
  const bounds = L.latLngBounds(
    locations
      .filter((loc) => loc.lat && loc.long)
      .map((loc) => [parseFloat(loc.lat), parseFloat(loc.long)] as [number, number]),
  );
  if (bounds.isValid()) {
    map.fitBounds(bounds, { padding: [40, 40] });
  }
  return null;
}

export default function Dashboard() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => dashboardApi.get(),
  });

  if (isLoading) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  }

  const counts = data?.counts;

  return (
    <div>
      {/* Stats Cards */}
      <Row gutter={[24, 24]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Total Inventory Value"
              value={counts?.total_value ?? 0}
              precision={2}
              prefix={<DollarOutlined />}
              suffix="IDR"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Active Products"
              value={counts?.active_products ?? 0}
              prefix={<ShoppingOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Locations"
              value={counts?.locations_count ?? 0}
              prefix={<HomeOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Low Stock Items"
              value={counts?.low_stock ?? 0}
              prefix={<WarningOutlined />}
              valueStyle={counts?.low_stock && counts.low_stock > 0 ? { color: '#cf1322' } : undefined}
            />
          </Card>
        </Col>
      </Row>

      {/* Inventory Locations Map */}
      <Card title="Inventory Locations" style={{ marginTop: 24 }}>
        {data?.inventories && data.inventories.length > 0 ? (
          <div style={{ width: '100%', height: 400 }}>
            <MapContainer
              style={{ width: '100%', height: '100%' }}
              scrollWheelZoom={false}
            >
              <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              />
              <FitBoundsOnMount locations={data.inventories} />
              {data.inventories
                .filter((loc: DashboardLocation) => loc.lat && loc.long)
                .map((loc: DashboardLocation) => (
                  <Marker key={loc.id} position={[parseFloat(loc.lat), parseFloat(loc.long)]}>
                    <Popup>{loc.name}</Popup>
                  </Marker>
                ))}
            </MapContainer>
          </div>
        ) : (
          <Empty description="No inventory locations" />
        )}
      </Card>

      {/* Low Stock Items */}
      <Card title="Low Stock Items" style={{ marginTop: 24 }}>
        {data?.stocks && data.stocks.length > 0 ? (
          <Table<LowStockItem>
            dataSource={data.stocks}
            columns={lowStockColumns}
            rowKey="id"
            pagination={false}
            locale={{ emptyText: 'No low stock items' }}
          />
        ) : (
          <Empty description="All stock levels are healthy" />
        )}
      </Card>
    </div>
  );
}
