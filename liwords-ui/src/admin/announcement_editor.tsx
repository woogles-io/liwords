import {
  DownOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  UpOutlined,
} from '@ant-design/icons';
import { Button, Divider, Form, Input, message } from 'antd';
import axios from 'axios';
import { Store } from 'rc-field-form/lib/interface';
import React, { useEffect } from 'react';
import { toAPIUrl } from '../api/api';
import { Announcements } from '../lobby/announcements';

const layout = {
  labelCol: {
    span: 2,
  },
  wrapperCol: {
    span: 16,
  },
};

export const AnnouncementEditor = () => {
  const [form] = Form.useForm();

  useEffect(() => {
    axios
      .post<Announcements>(
        toAPIUrl('config_service.ConfigService', 'GetAnnouncements'),
        {}
      )
      .then((resp) => {
        form.setFieldsValue({
          announcements: resp.data.announcements,
        });
      });
  }, [form]);

  const onFinish = (vals: Store) => {
    console.log('vals', vals);
    axios
      .post<Announcements>(
        toAPIUrl('config_service.ConfigService', 'SetAnnouncements'),
        {
          announcements: vals.announcements,
        }
      )
      .then((resp) => {
        message.info({
          content: 'Updated announcements on front page',
          duration: 3,
        });
      })
      .catch((err) => {
        message.error({
          content: 'Error ' + err.response?.data?.msg,
          duration: 5,
        });
      });
  };

  return (
    <>
      <Form {...layout} onFinish={onFinish} form={form}>
        <Form.List name="announcements">
          {(fields, { add, remove, move }) => (
            <>
              {fields.map(({ key, name, ...restField }, index, arr) => (
                <>
                  <Form.Item
                    {...restField}
                    name={[name, 'title']}
                    label="Title"
                    rules={[{ required: true, message: 'Missing title' }]}
                  >
                    <Input placeholder="Title" />
                  </Form.Item>
                  <Form.Item
                    {...restField}
                    name={[name, 'link']}
                    label="Link"
                    rules={[{ required: true, message: 'Missing link' }]}
                  >
                    <Input placeholder="https://" />
                  </Form.Item>
                  <Form.Item
                    {...restField}
                    name={[name, 'body']}
                    label="Body"
                    rules={[{ required: true, message: 'Missing body' }]}
                  >
                    <Input.TextArea
                      rows={4}
                      placeholder="Body - you can use Markdown, but avoid links in here."
                    />
                  </Form.Item>
                  <Button
                    onClick={() => remove(name)}
                    icon={<MinusCircleOutlined />}
                  >
                    Delete
                  </Button>
                  {index > 0 ? (
                    <Button
                      onClick={() => move(index, index - 1)}
                      icon={<UpOutlined />}
                    >
                      Move up
                    </Button>
                  ) : null}
                  {index < arr.length - 1 ? (
                    <Button
                      onClick={() => move(index, index + 1)}
                      icon={<DownOutlined />}
                    >
                      Move down
                    </Button>
                  ) : null}
                  <Divider />
                </>
              ))}
              <Form.Item>
                <Button
                  type="dashed"
                  onClick={() => add()}
                  block
                  icon={<PlusOutlined />}
                >
                  Add field
                </Button>
              </Form.Item>
            </>
          )}
        </Form.List>

        <Form.Item wrapperCol={{ ...layout.wrapperCol }}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};
