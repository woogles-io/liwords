import React, { useEffect, useState } from "react";
import { Card } from "antd";
import ReactMarkdown from "react-markdown";
import { Announcement } from "../gen/api/proto/config_service/config_service_pb";
import { useClient } from "../utils/hooks/connect";
import { ConfigService } from "../gen/api/proto/config_service/config_service_pb";

export type Announcements = {
  announcements: Array<Announcement>;
};

export const AnnouncementsWidget = () => {
  const [announcements, setAnnouncements] = useState<Array<Announcement>>([]);
  const configClient = useClient(ConfigService);
  useEffect(() => {
    (async () => {
      const resp = await configClient.getAnnouncements({});
      setAnnouncements(resp.announcements);
    })();
  }, [configClient]);

  const renderAnnouncements = announcements.map((a, idx) => (
    <a href={a.link} target="_blank" rel="noopener noreferrer" key={idx}>
      <li>
        <h4>{a.title}</h4>
        <div>
          <ReactMarkdown
            components={{
              img: ({ src }) => <img src={src} style={{ maxWidth: 300 }} />,
            }}
          >
            {a.body}
          </ReactMarkdown>
        </div>
      </li>
    </a>
  ));
  return (
    <Card title="Announcements" className="announcements-card">
      <ul className="announcements-list">{renderAnnouncements}</ul>
    </Card>
  );
};
