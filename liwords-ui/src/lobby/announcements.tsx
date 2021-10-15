import React, { useEffect } from 'react';
import { Card } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import ReactMarkdown from 'react-markdown';

type Announcement = {
  title: string;
  link: string;
  body: string;
};

export type Announcements = {
  announcements: Array<Announcement>;
};

export const Announcements = () => {
  const { useState } = useMountedState();

  const [announcements, setAnnouncements] = useState<Array<Announcement>>([]);
  useEffect(() => {
    axios
      .post<Announcements>(
        toAPIUrl('config_service.ConfigService', 'GetAnnouncements'),
        {}
      )
      .then((resp) => {
        setAnnouncements(resp.data.announcements);
      });
  }, []);

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
    <Card title="Announcements">
      <ul className="announcements-list">{renderAnnouncements}</ul>
    </Card>
  );
};
