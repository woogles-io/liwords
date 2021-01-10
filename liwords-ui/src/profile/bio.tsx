import React, { useCallback } from 'react';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { useParams } from 'react-router-dom';
import { useLoginStateStoreContext } from '../store/store';
import { Card, Modal, Form, Input } from 'antd';
import { MarkdownTips } from './markdown_tips';
import './bio.scss';

type BioProps = {
  bio: string;
};

export const BioCard = React.memo((props: BioProps) => {
  const { useState } = useMountedState();
  const { loginState } = useLoginStateStoreContext();
  const { username: viewer } = loginState;
  const { username } = useParams();
  const { TextArea } = Input;

  const [bioEditModalVisible, setBioEditModalVisible] = useState(false);
  const [candidateBio, setCandidateBio] = useState("");

  const actions = (viewer === username) 
    ? [(
        <div
          className="edit-bio"
          onClick={() => {
            setCandidateBio(props.bio);
            setBioEditModalVisible(true);
          }}
        >
          Edit
        </div>
      )] 
    : []
  
  const onChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setCandidateBio(e.target.value);
  }, []);

  return (
    <Card title="Bio" actions={actions}>
      <ReactMarkdown>{props.bio}</ReactMarkdown>
      <Modal
        title="Edit bio"
        visible={bioEditModalVisible}
        onCancel={() => {
          setBioEditModalVisible(false);
        }}
        className="bio-edit-modal"
      >
        <Form>
          <TextArea 
            rows={4} 
            value={candidateBio}
            onChange={onChange}
          />
        </Form>

      <div className="preview">
        <div>How your bio will look to others:</div>
        <Card className="preview-card">
          <ReactMarkdown>{candidateBio}</ReactMarkdown>
        </Card>
      </div>
      <MarkdownTips/> 
      </Modal>
    </Card>
  );
});

