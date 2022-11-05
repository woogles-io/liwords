import {
  DownOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  UpOutlined,
} from '@ant-design/icons';
import { Button, Divider, Form, Input, message } from 'antd';
import { Store } from 'rc-field-form/lib/interface';
import React, { useEffect } from 'react';
import { ConnectError } from '@bufbuild/connect-web';
import { ConfigService } from '../gen/api/proto/config_service/config_service_connectweb';
import { useClient } from '../utils/hooks/connect';

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
  const configService = useClient(ConfigService);
  useEffect(() => {
    const fetchAnnouncements = async () => {
      const resp = await configService.getAnnouncements({});
      form.setFieldsValue({
        announcements: resp.announcements,
      });
    };
    fetchAnnouncements();
  }, [configService, form]);

  const onFinish = async (vals: Store) => {
    console.log('vals', vals);
    try {
      await configService.setAnnouncements({
        announcements: vals.announcements,
      });
      message.info({
        content: 'Updated announcements on front page',
        duration: 3,
      });
    } catch (err) {
      if (err instanceof ConnectError) {
        message.error({
          content: err.message,
          duration: 3,
        });
      } else {
        console.log(err);
      }
    }
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
