import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Table, Button, Modal, Form, Input, InputNumber, Space, Popconfirm, message, Card,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { productsApi } from '../api/products';
import type { Product } from '../types';

export default function Products() {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['products'],
    queryFn: () => productsApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: productsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      message.success('Product created');
      setModalOpen(false);
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: string; name: string; description?: string; price?: number }) =>
      productsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      message.success('Product updated');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      message.success('Product deleted');
    },
    onError: (err: Error) => message.error(err.message),
  });

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (product: Product) => {
    setEditing(product);
    form.setFieldsValue(product);
    setModalOpen(true);
  };

  const handleOk = () => {
    form.validateFields().then((values) => {
      if (editing) {
        updateMutation.mutate({ id: editing.id, ...values });
      } else {
        createMutation.mutate(values);
      }
    });
  };

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: 'Price', dataIndex: 'price', key: 'price', render: (v: number) => `$${v.toFixed(2)}` },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, record: Product) => (
        <Space>
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this product?" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Card
        title="Products"
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Product</Button>}
      >
        <Table
          dataSource={data?.data ?? []}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={{ pageSize: 20, total: data?.total ?? 0 }}
        />
      </Card>

      <Modal
        title={editing ? 'Edit Product' : 'Create Product'}
        open={modalOpen}
        onOk={handleOk}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="price" label="Price" rules={[{ type: 'number', min: 0 }]}>
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
