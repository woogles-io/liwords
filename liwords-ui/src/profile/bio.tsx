import React, { useCallback } from 'react';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { useParams, Link } from 'react-router-dom';
import { useLoginStateStoreContext } from '../store/store';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { notification, Card, Modal, Form, Input, Alert } from 'antd';

type BioProps = {
  bio: string;
  bioLoaded: boolean;
};

export const BioCard = React.memo((props: BioProps) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { username: viewer } = loginState;
  const { username } = useParams();
  const { TextArea } = Input;
  const [err, setErr] = useState('');

  const [latestBio, setLatestBio] = useState('');

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [candidateBio, setCandidateBio] = useState('');

  React.useEffect(() => {
    setLatestBio(props.bio);
    console.log('useEffect');
  }, [props.bio]);

  const onChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setCandidateBio(e.target.value);
  }, []);

  return viewer === username || latestBio !== '' ? (
    <Card title="Bio">
      <ReactMarkdown>
        {latestBio ? latestBio : "You haven't yet provided your bio."}
      </ReactMarkdown>
    </Card>
  ) : (
    <React.Fragment />
  );
});
