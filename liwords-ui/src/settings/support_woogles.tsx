import React from 'react';
import { Button } from 'antd';
import woogles from '../assets/woogles.png';

type Props = {
  handleContribute?: () => void;
};

export const Support = React.memo((props: Props) => {
  return (
    <>
      <h3>Support Woogles.io</h3>
      <div className="support-woogles">
        <img src={woogles} className="woogles" alt="Woogles" />
        <div className="right-column">
          <div className="title">Help us keep Woogles.io going!</div>
          <div>
            We’re an entirely volunteer-run 503(c) NFP. If you’re enjoying the
            site, please feel free to contribute a few dollars to us!
          </div>
          <Button onClick={props.handleContribute}>Contribute</Button>
        </div>
      </div>
    </>
  );
});
