import React from "react";
import type { MessageInstance } from "antd/es/message/interface";
import type { HookAPI } from "antd/lib/modal/useModal";
import type { Client } from "@connectrpc/connect";
import { ModService } from "../gen/api/proto/mod_service/mod_service_pb";
import { flashError } from "../utils/hooks/connect";

// Asks whether to report a message to the moderators, and sends the report.
// Anything more to tell them goes by email.
export const reportMessage = (
  app: { modal: HookAPI; message: MessageInstance },
  modClient: Client<typeof ModService>,
  msg: { id: string; sender: string; message: string },
) => {
  app.modal.confirm({
    title: (
      <p className="readable-text-color">
        Report this message to the moderators?
      </p>
    ),
    content: (
      <div className="report-message">
        <p className="readable-text-color">
          {msg.sender}: {msg.message}
        </p>
        <p className="readable-text-color">
          To tell the moderators more, write to{" "}
          <a href="mailto:conduct@woogles.io">conduct@woogles.io</a>.
        </p>
      </div>
    ),
    okText: "Report",
    async onOk() {
      try {
        await modClient.reportChatMessage({ messageId: msg.id });
        app.message.success("Report sent");
      } catch (e) {
        flashError(e);
      }
    },
  });
};
