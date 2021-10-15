import React from 'react';

type Props = {
  children?: React.ReactNode;
  className?: string;
  open?: boolean;
};
export const Sidebar = React.memo((props: Props) => {
  const calculatedClassName = `sidebar ${props.className}`;
  return <aside className={calculatedClassName}>{props.children}</aside>;
});
