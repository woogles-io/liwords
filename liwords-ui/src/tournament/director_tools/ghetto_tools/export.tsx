// Export and print utilities

import { Button, Form, Select, Switch } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useState } from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";

export const ExportTournament = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };
  const tClient = useClient(TournamentService);
  const onSubmit = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      format: vals.format,
    };
    try {
      const resp = await tClient.exportTournament(obj);
      const url = window.URL.createObjectURL(new Blob([resp.exported]));
      const link = document.createElement("a");
      link.href = url;
      const tname = tournamentContext.metadata.name;
      let extension;
      switch (vals.format) {
        case "tsh":
          extension = "tsh";
          break;
        case "standingsonly":
          extension = "csv";
          break;
      }
      const downloadFilename = `${tname}.${extension}`;
      link.setAttribute("download", downloadFilename);
      document.body.appendChild(link);
      link.onclick = () => {
        link.remove();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
        }, 1000);
      };
      link.click();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <>
      <Form onFinish={onSubmit}>
        <Form.Item {...formItemLayout} label="Select format" name="format">
          <Select>
            <Select.Option value="tsh">
              NASPA tournament submit format
            </Select.Option>
            {/* <Select.Option value="aupair">AUPair format</Select.Option> */}
            <Select.Option value="standingsonly">
              Standings only (CSV)
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

export const CreatePrintableScorecards = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };
  const tClient = useClient(TournamentService);
  const [isLoading, setIsLoading] = useState(false);
  const onSubmit = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      showOpponents: vals.showOpponents,
      showSeeds: vals.showSeeds,
      showQrCode: vals.showQrCode,
    };
    setIsLoading(true);

    try {
      const resp = await tClient.getTournamentScorecards(obj);
      // @ts-expect-error - TypeScript issue with Uint8Array type
      const url = window.URL.createObjectURL(new Blob([resp.pdfZip]));
      const link = document.createElement("a");
      link.href = url;
      const tname = tournamentContext.metadata.name;
      const extension = "zip";
      const downloadFilename = `${tname}.${extension}`;
      link.setAttribute("download", downloadFilename);
      document.body.appendChild(link);
      link.onclick = () => {
        link.remove();
        setTimeout(() => {
          window.URL.revokeObjectURL(url);
        }, 1000);
      };
      link.click();
    } catch (e) {
      flashError(e);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Form onFinish={onSubmit}>
        <Form.Item
          {...formItemLayout}
          label="Show opponents"
          name="showOpponents"
        >
          <Switch />
        </Form.Item>
        <Form.Item {...formItemLayout} label="Show seeds" name="showSeeds">
          <Switch />
        </Form.Item>

        <Form.Item {...formItemLayout} label="Show QR code" name="showQrCode">
          <Switch />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={isLoading}>
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};
