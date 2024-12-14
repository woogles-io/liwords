import React from "react";
import { useBriefProfile } from "../utils/brief_profiles";

type DisplayFlagProps = {
  className?: string;
  countryCode?: string;
};

export const DisplayFlag = (props: DisplayFlagProps) => (
  <>
    {props.countryCode ? (
      <img
        className={`country-flag ${props.className ? props.className : ""}`}
        src={`https://woogles-flags.s3.us-east-2.amazonaws.com/${props.countryCode}.png`}
        alt={`${props.countryCode.toUpperCase()} flag`}
        title={`${props.countryCode.toUpperCase()} flag`}
      />
    ) : null}
  </>
);

export const DisplayUserFlag = ({ uuid }: { uuid: string | undefined }) => {
  const briefProfile = useBriefProfile(uuid);

  return (
    <React.Fragment>
      {briefProfile && <DisplayFlag countryCode={briefProfile.countryCode} />}
    </React.Fragment>
  );
};
