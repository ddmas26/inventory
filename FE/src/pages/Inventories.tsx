import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Table, Button, Modal, Form, Input, Space, Popconfirm, message, Card,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { inventoriesApi } from '../api/inventories';
import type { Inventory } from '../types';

export default function Inventories() {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Inventory | null>(null);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['inventories'],
    queryFn: () => inventoriesApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: inventoriesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['inventories'] });
      message.success('Inventory created');
      setModalOpen(false);
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: string; name: string; address?: string; latitude?: string; longitude?: string }) =>
      inventoriesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['inventories'] });
      message.success('Inventory updated');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
    },
    onError: (err: Error) => message.error(err.message),
  });

  const deleteMutation = useMutation({
    mutationFn: inventoriesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['inventories'] });
      message.success('Inventory deleted');
    },
    onError: (err: Error) => message.error(err.message),
  });

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (inv: Inventory) => {
    setEditing(inv);
    form.setFieldsValue(inv);
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
    { title: 'Address', dataIndex: 'address', key: 'address', ellipsis: true },
    { title: 'Latitude', dataIndex: 'latitude', key: 'latitude' },
    { title: 'Longitude', dataIndex: 'longitude', key: 'longitude' },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, record: Inventory) => (
        <Space>
          <Button icon={<EditOutlined />} size="small" onClick={() => openEdit(record)} />
          <Popconfirm title="Delete this inventory?" onConfirm={() => deleteMutation.mutate(record.id)}>
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Card
        title="Inventories"
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add Inventory</Button>}
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
        title={editing ? 'Edit Inventory' : 'Create Inventory'}
        open={modalOpen}
        onOk={handleOk}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="address" label="Address">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="latitude" label="Latitude">
            <Input />
          </Form.Item>
          <Form.Item name="longitude" label="Longitude">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
