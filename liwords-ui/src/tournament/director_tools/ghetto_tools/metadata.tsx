// Tournament metadata/settings form

import { Button, DatePicker, Form, Input, message, Switch } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useEffect } from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { isClubType } from "../../../store/constants";
import { useTournamentStoreContext } from "../../../store/store";
import {
  dayjsToProtobufTimestampIgnoringNanos,
  doesCurrentUserUse24HourTime,
  protobufTimestampToDayjsIgnoringNanos,
} from "../../../utils/datetime";
import { flashError, useClient } from "../../../utils/hooks/connect";

export const EditDescription = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  useEffect(() => {
    const metadata = tournamentContext.metadata;
    const scheduledStartTime = metadata.scheduledStartTime
      ? protobufTimestampToDayjsIgnoringNanos(metadata.scheduledStartTime)
      : null;
    const scheduledEndTime = metadata.scheduledEndTime
      ? protobufTimestampToDayjsIgnoringNanos(metadata.scheduledEndTime)
      : null;

    form.setFieldsValue({
      name: metadata.name,
      description: metadata.description,
      logo: metadata.logo,
      color: metadata.color,
      scheduledTime: {
        range: [scheduledStartTime, scheduledEndTime],
      },
      irlMode: metadata.irlMode,
      monitored: metadata.monitored,
    });
  }, [form, tournamentContext.metadata]);

  const onSubmit = async (vals: Store) => {
    const [scheduledStartTime, scheduledEndTime] = vals.scheduledTime
      ?.range ?? [null, null];
    const obj = {
      metadata: {
        name: vals.name,
        description: vals.description,
        logo: vals.logo,
        color: vals.color,
        id: props.tournamentID,
        scheduledStartTime: scheduledStartTime
          ? dayjsToProtobufTimestampIgnoringNanos(scheduledStartTime)
          : undefined,
        scheduledEndTime: scheduledEndTime
          ? dayjsToProtobufTimestampIgnoringNanos(scheduledEndTime)
          : undefined,
        irlMode: vals.irlMode,
        monitored: vals.monitored,
      },
      setOnlySpecified: true,
    };
    try {
      await tClient.setTournamentMetadata(obj);
      message.info({
        content: "Set tournament metadata successfully.",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const timeFormat = doesCurrentUserUse24HourTime() ? "HH:mm" : "hh:mm A";

  return (
    <>
      <Form form={form} onFinish={onSubmit} layout="vertical">
        <Form.Item name="name" label="Club or tournament name">
          <Input />
        </Form.Item>
        <Form.Item label="Tournament Start and End Times">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Use your local time zone. Times are used for tournament listing. The
            tournament will still only start/end when the director does so
            manually.
          </div>
          <Form.Item name={["scheduledTime", "range"]} noStyle>
            <DatePicker.RangePicker
              style={{ width: "100%" }}
              showTime={{ format: timeFormat }}
              format={`YYYY-MM-DD ${timeFormat}`}
              showNow={false}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item name="description" label="Description">
          <Input.TextArea rows={12} />
        </Form.Item>
        <Form.Item label="Logo URL">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Optional, requires refresh
          </div>
          <Form.Item name="logo" noStyle>
            <Input />
          </Form.Item>
        </Form.Item>
        <Form.Item label="Hex Color">
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Optional, requires refresh
          </div>

          <Form.Item name="color" noStyle>
            <Input placeholder="#00bdff" />
          </Form.Item>
        </Form.Item>
        <Form.Item>
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            IRL (In-Real-Life) Mode is used for real-life tournaments - games
            being played with a physical board and tiles. Once you turn this
            mode on, and click Submit, <em>you cannot turn it off</em>.
          </div>
          <Form.Item name="irlMode" label="IRL Mode">
            <Switch
              disabled={
                tournamentContext.metadata.irlMode ||
                isClubType(tournamentContext.metadata.type)
              }
            />
          </Form.Item>
        </Form.Item>
        <Form.Item>
          <div style={{ fontSize: "12px", color: "#666", marginBottom: "8px" }}>
            Monitoring/Invigilation mode requires participants to share their
            camera and screen via vdo.ninja for tournament oversight.
          </div>
          <Form.Item name="monitored" label="Monitoring Enabled">
            <Switch />
          </Form.Item>
        </Form.Item>
        <Form.Item style={{ paddingBottom: 20 }}>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};
