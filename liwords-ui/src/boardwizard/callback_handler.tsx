import { Divider, Typography } from 'antd';
import { GCGProcessForm } from './gcg_process_form';

export const CallbackHandler = () => {
  const urlParams = new URLSearchParams(window.location.search);

  return (
    <div style={{ padding: 20, maxWidth: 400 }}>
      <Typography.Text>
        You're almost done. Select the lexicon and challenge rule for your
        annotated game and click Create new annotated game.
      </Typography.Text>
      <Divider />
      <GCGProcessForm gcg={urlParams.get('gcg') ?? ''} showUpload={false} />
    </div>
  );
};
