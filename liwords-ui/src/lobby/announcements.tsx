import React, { useEffect } from 'react';
import { Card } from 'antd';
import { useMountedState } from '../utils/mounted';
import ReactMarkdown from 'react-markdown';
import { Announcement } from '../gen/api/proto/config_service/config_service_pb';
import { useClient } from '../utils/hooks/connect';
import { ConfigService } from '../gen/api/proto/config_service/config_service_connectweb';

export type Announcements = {
  announcements: Array<Announcement>;
};

export const AnnouncementsWidget = () => {
  const { useState } = useMountedState();
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
          <ReactMarkdown>{a.body}</ReactMarkdown>
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
