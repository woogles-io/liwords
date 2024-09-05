import { EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { Popconfirm, Tooltip, Typography } from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import { AuthenticationService } from '../gen/api/proto/user_service/user_service_connect';
import { flashError, useClient } from '../utils/hooks/connect';

export const API = () => {
  const authClient = useClient(AuthenticationService);
  const [apikey, setapikey] = useState('');
  const [keyhidden, setkeyhidden] = useState(true);
  const [confirmResetVisible, setConfirmResetVisible] = useState(false);
  useEffect(() => {
    (async () => {
      try {
        const resp = await authClient.getAPIKey({});
        setapikey(resp.key);
      } catch (e) {
        flashError(e);
      }
    })();
  }, [authClient]);

  const generateNewAPIKey = async () => {
    try {
      const resp = await authClient.getAPIKey({
        reset: true,
      });
      setapikey(resp.key);
      setkeyhidden(false);
      setConfirmResetVisible(false);
    } catch (e) {
      flashError(e);
    }
  };

  const toggleHidden = () => {
    setkeyhidden((v: boolean) => !v);
  };

  const keyDisplay = useMemo(() => {
    if (!apikey) {
      return 'You have not generated an API key';
    }
    if (keyhidden) {
      return (
        <Tooltip
          title="Click the eye icon to view full API key"
          placement="bottom"
        >
          {apikey.substring(0, 10) + '...'}
        </Tooltip>
      );
    }
    return (
      <Typography.Paragraph copyable className="readable-text-color">
        {apikey}
      </Typography.Paragraph>
    );
  }, [apikey, keyhidden]);

  let resetHeader = 'This will create a new API key.';
  if (apikey) {
    resetHeader = 'Resetting this API key will invalidate your last key.';
  }

  const apiKeyDisplay = (
    <div className="api-settings">
      {apikey && (
        <Tooltip title="View API key">
          <EyeOutlined onClick={toggleHidden} style={{ marginRight: 20 }} />
        </Tooltip>
      )}
      <Popconfirm
        title={`${resetHeader} Are you sure you wish to do this?`}
        onConfirm={generateNewAPIKey}
        onCancel={() => setConfirmResetVisible(false)}
        open={confirmResetVisible}
        okText="Yes"
        cancelText="No"
      >
        <Tooltip title={apikey ? 'Reset API key' : 'Generate API key'}>
          <ReloadOutlined onClick={() => setConfirmResetVisible(true)} />
        </Tooltip>
      </Popconfirm>
      <div style={{ marginTop: 10 }}>{keyDisplay}</div>
    </div>
  );

  return (
    <>
      <Typography.Paragraph type="danger">
        This is for coders only! If someone asks you to make an API key, do not
        do this unless you know exactly what you're doing.
      </Typography.Paragraph>
      <p style={{ marginBottom: 10 }}>
        Our API is a Protobuf API, using the Connect framework. It is used by
        this front end (the Woogles web app) to communicate with the backend.
      </p>
      <p style={{ marginBottom: 10 }}>
        You can check our API documentation at{' '}
        <a
          href="https://buf.build/domino14/liwords"
          target="_blank"
          rel="noreferrer"
        >
          https://buf.build/domino14/liwords
        </a>
        .
      </p>
      <div className="section-header">API Key</div>
      <p style={{ marginBottom: 10 }}>
        Your API key should be kept secret.{' '}
        <em>Please do not share it with anyone!</em>
      </p>
      <p>{apiKeyDisplay}</p>
    </>
  );
};
