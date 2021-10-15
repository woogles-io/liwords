import React from 'react';

type Props = {
  children?: React.ReactNode;
  className?: string;
};
export const Main = React.memo((props: Props) => {
  return (
    <main className={`main ${props.className || ''}`}>{props.children}</main>
  );
});
