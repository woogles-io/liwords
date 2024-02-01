import React from 'react';
import ReactMarkdown from 'react-markdown';
import { Table } from 'antd';
import { isMac, isWindows } from '../utils/cwgame/common';
import './markdown_tips.scss';

export const MarkdownTips = React.memo(() => {
  const asterisksExample = '*italics* or **bold**';
  const blankLineExample = 'line 1\n\nline 2';
  const linkExample = '[Woogles!](https://woogles.io)';
  const dataSource = [
    {
      key: '1',
      type: 'italics or bold',
      use: 'asterisks',
      example: asterisksExample,
      result: <ReactMarkdown>{asterisksExample}</ReactMarkdown>,
    },
    {
      key: '2',
      type: 'new line',
      use: 'return key twice',
      example: 'line 1⏎⏎line 2',
      result: <ReactMarkdown>{blankLineExample}</ReactMarkdown>,
    },
    {
      key: '3',
      type: 'web link',
      use: '[link title](web address)',
      example: linkExample,
      result: <ReactMarkdown>{linkExample}</ReactMarkdown>,
    },
    {
      key: '4',
      type: 'emoji',
      use: isMac()
        ? 'Command + control + space'
        : isWindows()
          ? 'Windows key + .'
          : 'use keyboard',
      example: '🥰',
      result: '🥰',
    },
  ];

  const columns = [
    { title: 'To get', dataIndex: 'type' },
    { title: 'Use', dataIndex: 'use' },
    { title: 'Example', dataIndex: 'example' },
    { title: 'Result', dataIndex: 'result' },
  ];

  return (
    <div className="markdown-tips">
      <Table
        dataSource={dataSource}
        columns={columns}
        pagination={{ hideOnSinglePage: true }}
      />
    </div>
  );
});
