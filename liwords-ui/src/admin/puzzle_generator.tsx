import React, { useCallback, useMemo, useState } from 'react';

import {
  Button,
  Divider,
  Form,
  Input,
  InputNumber,
  List,
  message,
  Select,
  Switch,
} from 'antd';
import { Store } from 'antd/lib/form/interface';
import { excludedLexica, LexiconFormItem } from '../shared/lexicon_display';
import {
  PuzzleBucket,
  PuzzleGenerationRequest,
  PuzzleTag,
} from '../gen/api/proto/macondo/macondo_pb';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import {
  APIPuzzleGenerationJobRequest,
  PuzzleGenerationJobRequest,
  PuzzleJobLog,
  PuzzleJobLogsRequest,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import moment from 'moment';
import { proto3 } from '@bufbuild/protobuf';
import { flashError, useClient } from '../utils/hooks/connect';
import { PuzzleService } from '../gen/api/proto/puzzle_service/puzzle_service_connectweb';

const layout = {
  labelCol: {
    span: 6,
  },
  wrapperCol: {
    span: 16,
  },
};

type formBucket = {
  size: number;
  includes: Array<PuzzleTag>;
  excludes: Array<PuzzleTag>;
};

export const PuzzleGenerator = () => {
  const [logs, setLogs] = useState<Array<PuzzleJobLog>>([]);

  const renderedLogs = useMemo(() => {
    return (
      <List
        itemLayout="horizontal"
        dataSource={logs}
        renderItem={(item) => {
          const createdAt = item.createdAt;
          const completedAt = item.completedAt;
          let dcreate, dcomplete;
          if (createdAt) {
            dcreate = moment(createdAt.toDate()).fromNow();
          }
          if (completedAt) {
            dcomplete = moment(completedAt.toDate()).fromNow();
          }

          return (
            <List.Item>
              <List.Item.Meta
                title={`Job ${item.id} - created: ${dcreate}`}
                description={
                  <div className="readable-text-color">
                    <p>Completed: {dcomplete}</p>
                    <p>Fulfilled: {`${item.fulfilled}`}</p>
                    <p>Error status: {item.errorStatus}</p>
                    <p>Request:</p>
                    <pre>{item.request?.toJsonString({ prettySpaces: 2 })}</pre>
                  </div>
                }
              />
            </List.Item>
          );
        }}
      />
    );
  }, [logs]);

  const puzzleClient = useClient(PuzzleService);

  const onFinish = useCallback(
    async (vals: Store) => {
      console.log('vals', vals);

      const apireq = new APIPuzzleGenerationJobRequest();

      const req = new PuzzleGenerationJobRequest();
      req.botVsBot = vals.bvb;
      req.lexicon = vals.lexicon;
      req.letterDistribution = vals.letterdist;
      req.sqlOffset = vals.sqlOffset;
      req.gameConsiderationLimit = vals.gameConsiderationLimit;
      req.gameCreationLimit = vals.gameCreationLimit;

      const bucketReq = new PuzzleGenerationRequest();

      const buckets = new Array<PuzzleBucket>();
      vals.buckets.forEach((bucket: formBucket) => {
        const pb = new PuzzleBucket();
        pb.size = bucket.size;

        pb.includes = bucket.includes ?? new Array<PuzzleTag>();
        pb.excludes = bucket.excludes ?? new Array<PuzzleTag>();
        buckets.push(pb);
      });

      bucketReq.buckets = buckets;

      req.request = bucketReq;

      apireq.secretKey = vals.secretKey;
      apireq.request = req;

      try {
        await puzzleClient.startPuzzleGenJob(apireq);
        message.info({ content: 'Submitted job' });
      } catch (e) {
        flashError(e);
      }
    },
    [puzzleClient]
  );

  const fetchRecentLogs = useCallback(async () => {
    const req = new PuzzleJobLogsRequest();
    req.offset = 0;
    req.limit = 20;

    try {
      const resp = await puzzleClient.getPuzzleJobLogs(req);
      setLogs(resp.logs);
    } catch (e) {
      flashError(e);
    }
  }, [puzzleClient]);

  const puzzleTags = useMemo(
    () =>
      proto3.getEnumType(PuzzleTag).values.map((val) => {
        return <Select.Option key={val.no}>{val.name}</Select.Option>;
      }),
    []
  );

  return (
    <div className="puzzle-generator">
      <Form
        {...layout}
        onFinish={onFinish}
        initialValues={{
          letterdist: 'english',
          lexicon: 'CSW21',
        }}
      >
        <Form.Item name="secretKey" label="Secret Key for Puzzle Generation">
          <Input type="password" />
        </Form.Item>
        <Form.Item
          name="bvb"
          /* go dortmund */ label="Bot vs Bot"
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <LexiconFormItem excludedLexica={excludedLexica(false, false)} />
        <Form.Item name="letterdist" label="Letter distribution">
          <Select>
            <Select.Option value="english">English</Select.Option>
            <Select.Option value="french">French</Select.Option>;
            <Select.Option value="german">German</Select.Option>
            <Select.Option value="norwegian">Norwegian</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item name="sqlOffset" label="SQL Offset">
          <InputNumber />
        </Form.Item>
        <Form.Item
          name="gameConsiderationLimit"
          label="Game Consideration Limit"
        >
          <InputNumber />
        </Form.Item>
        <Form.Item
          name="gameCreationLimit"
          label="Game Creation Limit (only for bot v bot games)"
        >
          <InputNumber />
        </Form.Item>

        <Form.List name="buckets">
          {(fields, { add, remove }) => (
            <>
              {fields.map((field) => (
                <>
                  <Form.Item
                    {...field}
                    name={[field.name, 'size']}
                    label="Size"
                    rules={[{ required: true, message: 'Missing bucket size' }]}
                  >
                    <InputNumber />
                  </Form.Item>
                  <Form.Item
                    {...field}
                    name={[field.name, 'includes']}
                    label="Includes"
                  >
                    <Select mode="multiple" allowClear>
                      {puzzleTags}
                    </Select>
                  </Form.Item>
                  <Form.Item
                    {...field}
                    name={[field.name, 'excludes']}
                    label="Excludes"
                  >
                    <Select mode="multiple" allowClear>
                      {puzzleTags}
                    </Select>
                  </Form.Item>
                  <Button
                    onClick={() => remove(field.name)}
                    icon={<MinusCircleOutlined />}
                  >
                    Delete
                  </Button>
                </>
              ))}
              <Form.Item>
                <Button
                  type="dashed"
                  onClick={() => add()}
                  block
                  icon={<PlusOutlined />}
                >
                  Add bucket
                </Button>
              </Form.Item>
            </>
          )}
        </Form.List>

        <Form.Item wrapperCol={{ ...layout.wrapperCol, offset: 3 }}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
      <Divider />
      <Button type="default" onClick={fetchRecentLogs}>
        Fetch recent generation logs
      </Button>
      {renderedLogs}
    </div>
  );
};
