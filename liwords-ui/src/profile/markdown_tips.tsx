import React from 'react';
import ReactMarkdown from 'react-markdown';
import { Table } from 'antd';
import './markdown_tips.scss';

export const MarkdownTips = React.memo(() => {
  const italicsExample = '*hello*';
  const boldExample ='**hello**';

  const dataSource = [
    {
      key: '1',
      type: 'Italics',
      use: 'single asterisks',
      example: italicsExample,
      result: <ReactMarkdown>{italicsExample}</ReactMarkdown>
    },
    {
      key: '2',
      type: 'Bold',
      use: 'double asterisks',
      example: boldExample,
      result: <ReactMarkdown>{boldExample}</ReactMarkdown>
    },
  ];

  const columns = [
    { title: 'To get', dataIndex: 'type' },
    { title: 'Use', dataIndex: 'use' },
    { title: 'Example', dataIndex: 'example' },
    { title: 'Result', dataIndex: 'result' }
  ];

  return (
    <Table
      className = "markdown-tips"
      title={() => 'Markdown Tips'}
      dataSource={dataSource} 
      columns={columns} 
      pagination={{hideOnSinglePage: true}}
    />
  );
});

