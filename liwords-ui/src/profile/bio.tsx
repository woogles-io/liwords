import React, { useCallback, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { useMountedState } from '../utils/mounted';
import { useParams } from 'react-router-dom';
import { useLoginStateStoreContext } from '../store/store';
import { Card, Modal, Table, Row, Col, Form, Input, Alert, Button } from 'antd';
import { TopBar } from '../topbar/topbar';
import './bio.scss';

type BioProps = {
  bio: string;
};

export const BioCard = React.memo((props: BioProps) => {
  const { useState } = useMountedState();
  const [bioEditVisible, setBioEditVisible] = useState(true);
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
  
  const dataSource = [
    {
      key: '1',
      type: 'Italics',
      use: 'single asterisks',
      example: '*hello*',
      result: <ReactMarkdown>*hello*</ReactMarkdown>
    },
    {
      key: '2',
      type: 'Bold',
      use: 'double asterisks',
      example: '**hello**',
      result: <ReactMarkdown>**hello**</ReactMarkdown>
    },
  ];

  const columns = [
    { title: 'To get', dataIndex: 'type' },
    { title: 'Use', dataIndex: 'use' },
    { title: 'Example', dataIndex: 'example' },
    { title: 'Result', dataIndex: 'result' }
  ];

  const foobar="hello!";

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
        <Form>
          <Form.Item
            label="Bio"
            name="bioMarkup"
          >
            <Input
              value='Blah'
              placeholder="Enter bio in Markdown form"
            />
          </Form.Item>
        </Form>

      <div className="preview">
        <div>How your bio will look to others:</div>
        <Card className="preview-card">
        <ReactMarkdown>{props.bio}</ReactMarkdown>
        </Card>
      </div>
      <Table 
        title={() => 'Markdown Tips'}
        dataSource={dataSource} 
        columns={columns} 
        pagination={{hideOnSinglePage: true}}
      />
      </Modal>
    </Card>
  );
});

