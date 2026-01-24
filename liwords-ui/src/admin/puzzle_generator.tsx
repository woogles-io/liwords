import React, { useCallback, useMemo, useState } from "react";

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
} from "antd";
import { Store } from "antd/lib/form/interface";
import { excludedLexica, LexiconFormItem } from "../shared/lexicon_display";
import {
  PuzzleBucket,
  PuzzleBucketSchema,
  PuzzleGenerationRequestSchema,
  PuzzleTag,
} from "../gen/api/vendor/macondo/macondo_pb";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import {
  APIPuzzleGenerationJobRequestSchema,
  PuzzleGenerationJobRequestSchema,
  PuzzleJobLog,
  PuzzleJobLogsRequestSchema,
} from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import moment from "moment";
import { create, toJsonString } from "@bufbuild/protobuf";
import { flashError, useClient } from "../utils/hooks/connect";
import { PuzzleService } from "../gen/api/proto/puzzle_service/puzzle_service_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { enumToOptions } from "../utils/protobuf";

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
            dcreate = moment(timestampDate(createdAt)).fromNow();
          }
          if (completedAt) {
            dcomplete = moment(timestampDate(completedAt)).toISOString();
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
                    <pre>
                      {item.request
                        ? toJsonString(
                            PuzzleGenerationJobRequestSchema,
                            item.request,
                            { prettySpaces: 2 },
                          )
                        : ""}
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

  const puzzleClient = useClient(PuzzleService);

  const onFinish = useCallback(
    async (vals: Store) => {
      console.log("vals", vals);

      const apireq = create(APIPuzzleGenerationJobRequestSchema, {});

      const req = create(PuzzleGenerationJobRequestSchema, {});
      req.botVsBot = vals.bvb;
      req.lexicon = vals.lexicon;
      req.letterDistribution = vals.letterdist;
      req.gameConsiderationLimit = vals.gameConsiderationLimit;
      req.gameCreationLimit = vals.gameCreationLimit;
      req.avoidBotGames = vals.avoidBotGames;
      req.daysPerChunk = vals.daysPerChunk;
      req.equityLossTotalLimit = vals.equityLossTotalLimit;
      req.startDate = vals.startDate;
      const bucketReq = create(PuzzleGenerationRequestSchema, {});

      const buckets = new Array<PuzzleBucket>();
      vals.buckets.forEach((bucket: formBucket) => {
        const pb = create(PuzzleBucketSchema, {});
        pb.size = bucket.size;

        pb.includes =
          bucket.includes?.map((v) => Number(v)) ?? new Array<PuzzleTag>();
        pb.excludes =
          bucket.excludes?.map((v) => Number(v)) ?? new Array<PuzzleTag>();
        buckets.push(pb);
      });

      bucketReq.buckets = buckets;

      req.request = bucketReq;

      apireq.secretKey = vals.secretKey;
      apireq.request = req;

      try {
        await puzzleClient.startPuzzleGenJob(apireq);
        message.info({ content: "Submitted job" });
      } catch (e) {
        flashError(e);
      }
    },
    [puzzleClient],
  );

  const fetchRecentLogs = useCallback(async () => {
    const req = create(PuzzleJobLogsRequestSchema, {});
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
      enumToOptions(PuzzleTag).map((key) => {
        return (
          <Select.Option key={key.value} value={key.value}>
            {key.label}
          </Select.Option>
        );
      }),
    [],
  );

  return (
    <div className="puzzle-generator" style={{ padding: 24 }}>
      <Form
        // {...layout}
        onFinish={onFinish}
        initialValues={{
          letterdist: "english",
          lexicon: "CSW24",
        }}
        layout="vertical"
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
            <Select.Option value="catalan">Catalan</Select.Option>
            <Select.Option value="spanish">Spanish</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item
          name="gameConsiderationLimit"
          label="Game Consideration Limit (total for bots, per chunk for real games)"
          initialValue={6000}
        >
          <InputNumber inputMode="numeric" />
        </Form.Item>
        <Form.Item
          name="gameCreationLimit"
          label="Game Creation Limit (only for bot v bot games)"
        >
          <InputNumber inputMode="numeric" />
        </Form.Item>

        <Form.Item
          name="startDate"
          label="Start date in YYYY-MM-DD format. Puzzles will be created with this date as the latest date."
        >
          <Input />
        </Form.Item>

        <Form.Item
          name="equityLossTotalLimit"
          label="Total equity loss limit"
          initialValue={150}
        >
          <InputNumber inputMode="numeric" />
        </Form.Item>

        <Form.Item
          name="avoidBotGames"
          label="Avoid games where a bot is involved"
          initialValue="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="daysPerChunk"
          label="How many days to search at a time"
          initialValue={1}
        >
          <InputNumber inputMode="numeric" />
        </Form.Item>

        <Form.List name="buckets">
          {(fields, { add, remove }) => (
            <>
              {fields.map((field) => (
                <React.Fragment key={field.key}>
                  <Form.Item
                    name={[field.name, "size"]}
                    label="Size"
                    rules={[{ required: true, message: "Missing bucket size" }]}
                  >
                    <InputNumber inputMode="numeric" />
                  </Form.Item>
                  <Form.Item name={[field.name, "includes"]} label="Includes">
                    <Select mode="multiple" allowClear>
                      {puzzleTags}
                    </Select>
                  </Form.Item>
                  <Form.Item name={[field.name, "excludes"]} label="Excludes">
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
                </React.Fragment>
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
