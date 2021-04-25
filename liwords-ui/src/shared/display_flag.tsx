import React from 'react';

type DisplayFlagProps = {
  className?: string;
  countryCode?: string;
};

export const DisplayFlag = (props: DisplayFlagProps) => (
  <>
    {props.countryCode ? (
      <img
        className={`country-flag ${props.className ? props.className : ''}`}
        src={`https://woogles-flags.s3.us-east-2.amazonaws.com/${props.countryCode}.png`}
        alt={`${props.countryCode.toUpperCase()} flag`}
      />
    ) : null}
  </>
);
