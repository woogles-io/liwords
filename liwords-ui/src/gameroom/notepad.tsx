import React, { useRef, useState } from 'react';
import { Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

type NotepadProps = {
  style?: React.CSSProperties;
};

export const Notepad = React.memo((props: NotepadProps) => {
  const notepad = useRef<HTMLTextAreaElement>(null);
  const [curNotepad, setCurNotepad] = useState('');
  return (
    <div className="notepad-container">
      <textarea
        className="notepad"
        value={curNotepad}
        ref={notepad}
        spellCheck={false}
        style={props.style}
        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
          setCurNotepad(e.target.value);
        }}
      />
      <Button
        className="add-play"
        shape="circle"
        icon={<PlusOutlined />}
        type="primary"
        onClick={() => {}}
      />
    </div>
  );
});
