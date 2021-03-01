import React from 'react';

import { Button, Modal, Upload, Alert } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';

type Props = {
  visible: boolean;
  error: string;
  onOk: (avatarFile: File) => void;
  onCancel: () => void;
};

export const AvatarEditModal = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [avatarFile, setAvatarFile] = useState(new File([''], ''));

  let okButtonDisabled = avatarFile == null || avatarFile.name.length === 0;

  const fileProps = {
    beforeUpload: (file: File) => {
      return false;
    },
    maxCount: 1,
    onChange: (info: any) => {
      if (info.fileList.length > 0) {
        setAvatarFile(info.fileList.slice(-1)[0].originFileObj);
      } else {
        setAvatarFile(new File([''], ''));
      }
    },
    accept: 'image/jpeg',
    showUploadList: false,
  };

  return (
    <Modal
      className="avatar-update-modal"
      title="Update avatar"
      visible={props.visible}
      okText="Upload"
      okButtonProps={{ disabled: okButtonDisabled }}
      onCancel={() => {
        props.onCancel();
      }}
      onOk={() => {
        props.onOk(avatarFile);
      }}
    >
      <Upload {...fileProps}>
        <Button icon={<UploadOutlined />}>Select avatar</Button>
      </Upload>
      <>{avatarFile.name}</>
      {props.error !== '' ? <Alert message={props.error} type="error" /> : null}
    </Modal>
  );
});
