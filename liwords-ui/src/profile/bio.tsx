import React, { useCallback, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { useParams } from 'react-router-dom';
import { useLoginStateStoreContext } from '../store/store';
import { Card, Modal, Row, Col, Form, Input, Alert, Button } from 'antd';
import { TopBar } from '../topbar/topbar';

type BioProps = {
  bio: string;
};

export const BioCard = React.memo((props: BioProps) => {
  const { useState } = useMountedState();
  const [bioEditVisible, setBioEditVisible] = useState(false);
  const { loginState } = useLoginStateStoreContext();
  const { username: viewer } = loginState;
  const { username } = useParams();

  const actions = (viewer === username) 
    ? [(
        <div
          className="edit-bio"
          onClick={() => {
            setBioEditVisible(true);
          }}
        >
          Edit
        </div>
      )] 
    : []
  
  return (
    <Card title="Bio" actions={actions}>
      <ReactMarkdown>{props.bio}</ReactMarkdown>
      <Modal
        title="Edit bio"
        visible={bioEditVisible}
        onCancel={() => {
          setBioEditVisible(false);
        }}
      >
        <Form>
          <Form.Item
            label="Bio"
            name="bioMarkup"
          >
            <Input
              placeholder="Enter bio in Markdown form"
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
});

