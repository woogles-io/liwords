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

  const [bioEditVisible, setBioEditVisible] = useState(true);

  const actions = (viewer === username) 
    ? [(
        <div
          className="edit-bio"
          onClick={() => {
            setBioEditVisible(true);
            //candidateBio = props.bio;
          }}
        >
          Edit
        </div>
      )] 
    : []
  
  const onChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    console.log(e.target.value);
  }, []);

  return (
    <Card title="Bio" actions={actions}>
      <ReactMarkdown>{props.bio}</ReactMarkdown>
      <Modal
        title="Edit bio"
        visible={bioEditVisible}
        onCancel={() => {
          setBioEditVisible(false);
        }}
        className="bio-edit-modal"
      >
        <Form initialValues={{bio: props.bio}}>
          <TextArea 
            rows={4} 
            value={props.bio}
            onChange={onChange}
          />
        </Form>

      <div className="preview">
        <div>How your bio will look to others:</div>
        <Card className="preview-card">
        <ReactMarkdown>{props.bio}</ReactMarkdown>
        </Card>
      </div>
      <MarkdownTips/> 
      </Modal>
    </Card>
  );
});

