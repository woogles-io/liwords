import { Layout, Menu } from "antd";
import { MenuInfo } from "rc-menu/lib/interface";
import React, { useState } from "react";
import { TopBar } from "../navigation/topbar";
import { TourneyEditor } from "./tourney_editor";
// import { UserEditor } from './user_editor';
import { AnnouncementEditor } from "./announcement_editor";
import "./admin.scss";
import { PuzzleGenerator } from "./puzzle_generator";
// import { TourneyManager } from './tourney_manager';

type SiderProps = {
  setVisibleTab: React.Dispatch<React.SetStateAction<string>>;
};

const Sider = (props: SiderProps) => {
  const handleClick = (info: MenuInfo) => {
    props.setVisibleTab(info.key);
  };
  return (
    <Menu
      onClick={handleClick}
      style={{ width: 200 }}
      mode="inline"
      items={[
        //{ label: 'Options' },
        //{
        //  key: 'tournaments',
        //  label: 'Tournaments',
        //  children: [
        { key: "edit-tournament", label: "Edit Tournament" },
        { key: "new-tournament", label: "New Tournament" },
        { key: "announcement-editor", label: "Edit Announcements" },
        { key: "puzzle-generator", label: "Puzzle Generator" },
        //    { key: 'manage-tournament', label: 'Tournament Manager' },
        //    { key: 'user-editor', label: 'User Editor' },
        //  ],
        //},
      ]}
    />
  );
};

export const Admin = () => {
  const [visibleTab, setVisibleTab] = useState("");
  return (
    <>
      <Layout>
        <TopBar />
        <Layout>
          <Layout.Sider>
            <Sider setVisibleTab={setVisibleTab} />
          </Layout.Sider>
          <Layout.Content>
            {visibleTab === "edit-tournament" && <TourneyEditor mode="edit" />}
            {visibleTab === "new-tournament" && <TourneyEditor mode="new" />}
            {visibleTab === "announcement-editor" && <AnnouncementEditor />}
            {visibleTab === "puzzle-generator" && <PuzzleGenerator />}
            {/* {visibleTab === 'user-editor' && <UserEditor />} */}
            {/* {visibleTab === 'manage-tournament' && <TourneyManager />} */}
          </Layout.Content>
        </Layout>
      </Layout>
    </>
  );
};
