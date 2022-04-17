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
import { LiwordsAPIError, postProto } from '../api/api';
import { Store } from 'antd/lib/form/interface';
import { excludedLexica, LexiconFormItem } from '../shared/lexicon_display';
import {
  PuzzleBucket,
  PuzzleGenerationRequest,
  PuzzleTag,
  PuzzleTagMap,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import {
  APIPuzzleGenerationJobRequest,
  APIPuzzleGenerationJobResponse,
  PuzzleGenerationJobRequest,
  PuzzleJobLog,
  PuzzleJobLogsRequest,
  PuzzleJobLogsResponse,
} from '../gen/api/proto/puzzle_service/puzzle_service_pb';
import moment from 'moment';

const layout = {
  labelCol: {
    span: 3,
  },
  wrapperCol: {
    span: 16,
  },
};

// don't code like this:
const bucketToProto = (strlist: Array<keyof PuzzleTagMap>) => {
  return strlist?.map((val) => {
    switch (val) {
      case 'EQUITY':
        return PuzzleTag.EQUITY;
      case 'BINGO':
        return PuzzleTag.BINGO;
      case 'ONLY_BINGO':
        return PuzzleTag.ONLY_BINGO;
      case 'BLANK_BINGO':
        return PuzzleTag.BLANK_BINGO;
      case 'NON_BINGO':
        return PuzzleTag.NON_BINGO;
      case 'POWER_TILE':
        return PuzzleTag.POWER_TILE;
      case 'BINGO_NINE_OR_ABOVE':
        return PuzzleTag.BINGO_NINE_OR_ABOVE;
      case 'CEL_ONLY':
        return PuzzleTag.CEL_ONLY;
    }
  });
};

type formBucket = {
  size: number;
  includes: Array<keyof PuzzleTagMap>;
  excludes: Array<keyof PuzzleTagMap>;
};

export const PuzzleGenerator = () => {
  const [logs, setLogs] = useState<Array<PuzzleJobLog>>([]);

  const renderedLogs = useMemo(() => {
    return (
      <List
        itemLayout="horizontal"
        dataSource={logs}
        renderItem={(item) => {
          const createdAt = item.getCreatedAt();
          const completedAt = item.getCompletedAt();
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
                title={`Job ${item.getId()} - created: ${dcreate}`}
                description={
                  <div className="readable-text-color">
                    <p>Completed: {dcomplete}</p>
                    <p>Fulfilled: {`${item.getFulfilled()}`}</p>
                    <p>Error status: {item.getErrorStatus()}</p>
                    <p>Request:</p>
                    <pre>
                      {JSON.stringify(
                        item.getRequest()?.toObject(),
                        undefined,
                        2
                      )}
                    </pre>
                  </div>
                }
              />
            </List.Item>
          );
        }}
      />
    );
  }, [logs]);

  const onFinish = useCallback(async (vals: Store) => {
    console.log('vals', vals);

    const apireq = new APIPuzzleGenerationJobRequest();

    const req = new PuzzleGenerationJobRequest();
    req.setBotVsBot(vals.bvb);
    req.setLexicon(vals.lexicon);
    req.setLetterDistribution(vals.letterdist);
    req.setSqlOffset(vals.sqlOffset);
    req.setGameConsiderationLimit(vals.gameConsiderationLimit);
    req.setGameCreationLimit(vals.gameCreationLimit);

    const bucketReq = new PuzzleGenerationRequest();

    const buckets = new Array<PuzzleBucket>();
    vals.buckets.forEach((bucket: formBucket) => {
      const pb = new PuzzleBucket();
      pb.setSize(bucket.size);

      pb.setIncludesList(bucketToProto(bucket.includes));
      pb.setExcludesList(bucketToProto(bucket.excludes));
      buckets.push(pb);
    });

    bucketReq.setBucketsList(buckets);

    req.setRequest(bucketReq);

    apireq.setSecretKey(vals.secretKey);
    apireq.setRequest(req);

    try {
      await postProto(
        APIPuzzleGenerationJobResponse,
        'puzzle_service.PuzzleService',
        'StartPuzzleGenJob',
        apireq
      );
      message.info({ content: 'Submitted job' });
    } catch (e) {
      message.error({
        content: (e as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, []);

  const fetchRecentLogs = useCallback(async () => {
    const req = new PuzzleJobLogsRequest();
    req.setOffset(0);
    req.setLimit(20);
    // Add pagination later.
    try {
      const resp = await postProto(
        PuzzleJobLogsResponse,
        'puzzle_service.PuzzleService',
        'GetPuzzleJobLogs',
        req
      );
      setLogs(resp.getLogsList());
    } catch (e) {
      message.error({
        content: (e as LiwordsAPIError).message,
        duration: 5,
      });
    }
  }, []);

  // TODO: figure out how to import this from the protobuf.
  const puzzleTags = useMemo(
    () =>
      [
        'EQUITY',
        'BINGO',
        'ONLY_BINGO',
        'BLANK_BINGO',
        'NON_BINGO',
        'POWER_TILE',
        'BINGO_NINE_OR_ABOVE',
        'CEL_ONLY',
      ].map((name) => {
        return <Select.Option key={name}>{name}</Select.Option>;
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
// postJsonObj
