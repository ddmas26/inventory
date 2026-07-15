import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card, Table, Select, Button, Modal, Form, InputNumber, Space, message, Tabs, Row, Col, Statistic, Input,
} from 'antd';
import { ReloadOutlined, SearchOutlined, ArrowUpOutlined, ArrowDownOutlined } from '@ant-design/icons';
import { productsApi } from '../api/products';
import { inventoriesApi } from '../api/inventories';
import { stockApi } from '../api/stock';
import type { Stock } from '../types';

export default function StockPage() {
  const queryClient = useQueryClient();
  const [selectedInv, setSelectedInv] = useState<string | undefined>();
  const [selectedProd, setSelectedProd] = useState<string | undefined>();
  const [search, setSearch] = useState('');
  const [orderBy, setOrderBy] = useState('stock.created_at');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [modalOpen, setModalOpen] = useState(false);
  const [modalAction, setModalAction] = useState<'add' | 'deduct' | 'set'>('add');
  const [form] = Form.useForm();

  const { data: products } = useQuery({ queryKey: ['products', 'all'], queryFn: () => productsApi.list(1, 1000) });
  const { data: inventories } = useQuery({ queryKey: ['inventories', 'all'], queryFn: () => inventoriesApi.list(1, 1000) });

  const { data: stockRes, isLoading, refetch } = useQuery({
    queryKey: ['stock', selectedInv, selectedProd, search, orderBy, sortDir],
    queryFn: () => stockApi.list({
      inventory_id: selectedInv,
      product_id: selectedProd,
      search: search || undefined,
      order_by: orderBy,
      sort: sortDir,
    }),
  });
  const stockList = stockRes?.data ?? [];

  const productOptions = (products?.data ?? []).map((p) => ({ label: p.name, value: p.id }));
  const inventoryOptions = (inventories?.data ?? []).map((i) => ({ label: i.name, value: i.id }));

  const sortOptions = [
    { label: 'Product', value: 'products.name' },
    { label: 'Inventory', value: 'inventories.name' },
    { label: 'Quantity', value: 'stock.quantity' },
    { label: 'Created', value: 'stock.created_at' },
    { label: 'Updated', value: 'stock.updated_at' },
  ];

  const stockMutation = useMutation({
    mutationFn: ({ action, inventoryId, productId, quantity }: { action: string; inventoryId: string; productId: string; quantity: number }) => {
      switch (action) {
        case 'add': return stockApi.add(inventoryId, productId, quantity);
        case 'deduct': return stockApi.deduct(inventoryId, productId, quantity);
        case 'set': return stockApi.set(inventoryId, productId, quantity);
        default: throw new Error('Unknown action');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stock'] });
      message.success(`Stock ${modalAction}ed successfully`);
      setModalOpen(false);
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  const openStockAction = (action: 'add' | 'deduct' | 'set') => {
    setModalAction(action);
    form.resetFields();
    if (selectedInv) {
      form.setFieldsValue({ inventory_id: selectedInv });
    }
    setModalOpen(true);
  };

  const handleStockOk = () => {
    form.validateFields().then((values) => {
      stockMutation.mutate({
        action: modalAction,
        inventoryId: values.inventory_id,
        productId: values.product_id,
        quantity: values.quantity,
      });
    });
  };

  const columns = [
    { title: 'Product', dataIndex: 'product_name', key: 'product' },
    { title: 'Inventory', dataIndex: 'inventory_name', key: 'inventory' },
    { title: 'Quantity', dataIndex: 'quantity', key: 'quantity',
      render: (v: number) => (
        <Statistic value={v} suffix="units" valueStyle={{ fontSize: 16 }} />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, record: Stock) => (
        <Space>
          <Button size="small" onClick={() => {
            setModalAction('add');
            form.setFieldsValue({ inventory_id: record.inventory_id, product_id: record.product_id, quantity: 1 });
            setModalOpen(true);
          }}>
            Add
          </Button>
          <Button size="small" onClick={() => {
            setModalAction('deduct');
            form.setFieldsValue({ inventory_id: record.inventory_id, product_id: record.product_id, quantity: 1 });
            setModalOpen(true);
          }}>
            Deduct
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="large">
      {/* Filters & Actions */}
      <Card>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} md={6}>
            <Input
              placeholder="Search by product name..."
              prefix={<SearchOutlined />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={12} md={4}>
            <Select
              placeholder="Filter by Inventory"
              options={inventoryOptions}
              value={selectedInv}
              onChange={setSelectedInv}
              style={{ width: '100%' }}
              allowClear
            />
          </Col>
          <Col xs={12} md={4}>
            <Select
              placeholder="Filter by Product"
              options={productOptions}
              value={selectedProd}
              onChange={setSelectedProd}
              style={{ width: '100%' }}
              allowClear
              showSearch
            />
          </Col>
          <Col xs={12} md={3}>
            <Select
              options={sortOptions}
              value={orderBy}
              onChange={setOrderBy}
              style={{ width: '100%' }}
            />
          </Col>
          <Col xs={6} md={1}>
            <Button
              icon={sortDir === 'desc' ? <ArrowDownOutlined /> : <ArrowUpOutlined />}
              onClick={() => setSortDir(sortDir === 'desc' ? 'asc' : 'desc')}
            />
          </Col>
          <Col xs={6} md={2}>
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>Refresh</Button>
          </Col>
          <Col xs={24} md={4}>
            <Space>
              <Button type="primary" onClick={() => openStockAction('add')}>
                Add Stock
              </Button>
              <Button onClick={() => openStockAction('deduct')}>
                Deduct Stock
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      <Tabs items={[
        {
          key: 'inventory',
          label: 'Stock Entries',
          children: (
            <Card>
              <Table
                dataSource={stockList}
                columns={columns}
                rowKey={(r) => `${r.inventory_id}-${r.product_id}`}
                loading={isLoading}
                locale={{ emptyText: 'No stock entries found' }}
                pagination={{
                  total: stockRes?.total ?? 0,
                  pageSize: stockRes?.page_size ?? 20,
                  current: stockRes?.page_index ?? 1,
                  showSizeChanger: false,
                }}
              />
            </Card>
          ),
        },
        {
          key: 'transfer',
          label: 'Transfer Stock',
          children: <TransferStock inventoryOptions={inventoryOptions} productOptions={productOptions} />,
        },
      ]} />

      <Modal
        title={`${modalAction.charAt(0).toUpperCase() + modalAction.slice(1)} Stock`}
        open={modalOpen}
        onOk={handleStockOk}
        onCancel={() => { setModalOpen(false); form.resetFields(); }}
        confirmLoading={stockMutation.isPending}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="inventory_id" label="Inventory" rules={[{ required: true, message: 'Select an inventory' }]}>
            <Select options={inventoryOptions} showSearch placeholder="Select an inventory" />
          </Form.Item>
          <Form.Item name="product_id" label="Product" rules={[{ required: true, message: 'Select a product' }]}>
            <Select options={productOptions} showSearch />
          </Form.Item>
          <Form.Item name="quantity" label="Quantity" rules={[{ required: true, type: 'number', min: modalAction === 'set' ? 0 : 1 }]}>
            <InputNumber min={modalAction === 'set' ? 0 : 1} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

function TransferStock({ inventoryOptions, productOptions }: { inventoryOptions: { label: string; value: string }[]; productOptions: { label: string; value: string }[] }) {
  const queryClient = useQueryClient();
  const [form] = Form.useForm();

  const transferMutation = useMutation({
    mutationFn: (values: { from_inventory_id: string; to_inventory_id: string; product_id: string; quantity: number }) =>
      stockApi.transfer(values.from_inventory_id, values.to_inventory_id, values.product_id, values.quantity),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stock'] });
      message.success('Stock transferred successfully');
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  return (
    <Card>
      <Form form={form} layout="vertical" style={{ maxWidth: 500 }} onFinish={(values) => transferMutation.mutate(values)}>
        <Form.Item name="from_inventory_id" label="From Inventory" rules={[{ required: true }]}>
          <Select options={inventoryOptions} showSearch />
        </Form.Item>
        <Form.Item name="to_inventory_id" label="To Inventory" rules={[{ required: true }]}>
          <Select options={inventoryOptions} showSearch />
        </Form.Item>
        <Form.Item name="product_id" label="Product" rules={[{ required: true }]}>
          <Select options={productOptions} showSearch />
        </Form.Item>
        <Form.Item name="quantity" label="Quantity" rules={[{ required: true, type: 'number', min: 1 }]}>
          <InputNumber min={1} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={transferMutation.isPending}>
            Transfer
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
}
