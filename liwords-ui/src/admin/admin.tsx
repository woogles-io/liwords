import { Menu } from 'antd';
import React from 'react';
import { TopBar } from '../topbar/topbar';
import { TourneyEditor } from './tourney_editor';
// import { UserEditor } from './user_editor';
import { useMountedState } from '../utils/mounted';

type Props = {};

type SiderProps = {
  setVisibleTab: React.Dispatch<React.SetStateAction<string>>;
};

const Sider = (props: SiderProps) => {
  const handleClick = (info: any) => {
    props.setVisibleTab(info.key);
  };
  return (
    <Menu onClick={handleClick} style={{ width: 200 }} mode="inline">
      {/* <Menu.Item>Options</Menu.Item> */}
      {/* <SubMenu key="tournaments" title="Tournaments"> */}
      <Menu.Item key="edit-tournament">Edit Tournament</Menu.Item>
      <Menu.Item key="new-tournament">New Tournament</Menu.Item>
      {/* <Menu.Item key="manage-tournament">Tournament Manager</Menu.Item> */}
      {/* <Menu.Item key="user-editor">User Editor</Menu.Item> */}
      {/* </SubMenu> */}
    </Menu>
  );
};

export const Admin = () => {
  const { useState } = useMountedState();
  const [visibleTab, setVisibleTab] = useState('');
  return (
    <>
      <TopBar />
      {/* <TourneyEditor /> */}
      <Sider setVisibleTab={setVisibleTab} />
      {visibleTab === 'edit-tournament' && <TourneyEditor mode="edit" />}
      {visibleTab === 'new-tournament' && <TourneyEditor mode="new" />}
      {/* {visibleTab === 'user-editor' && <UserEditor />} */}
      {/* {visibleTab === 'manage-tournament' && <TourneyManager />} */}
    </>
  );
};
