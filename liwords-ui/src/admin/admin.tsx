import { Layout, Menu } from "antd";
import { MenuInfo } from "rc-menu/lib/interface";
import React, { useState } from "react";
import { TopBar } from "../navigation/topbar";
import { TourneyEditor } from "./tourney_editor";
// import { UserEditor } from './user_editor';
import { AnnouncementEditor } from "./announcement_editor";
import "./admin.scss";
import { PuzzleGenerator } from "./puzzle_generator";
import { UserDetails } from "./user_details";
import { PermsAndRoles } from "./perms_and_roles";
import { UserBadges } from "./user_badges";
import { VerificationQueue } from "./verification_queue";
import { ManualOrgAssignment } from "./manual_org_assignment";
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
        { key: "user-details", label: "See User Details" },
        { key: "perms-and-roles", label: "User Permissions and Roles" },
        { key: "badges", label: "Assign User Badges" },
        { key: "verification-queue", label: "Identity Verification Queue" },
        {
          key: "manual-org-assignment",
          label: "Manual Organization Assignment",
        },
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
        <Layout style={{ marginTop: 20 }}>
          <Layout.Sider width={250}>
            <Sider setVisibleTab={setVisibleTab} />
          </Layout.Sider>
          <Layout.Content style={{ marginLeft: 20 }}>
            {visibleTab === "edit-tournament" && <TourneyEditor mode="edit" />}
            {visibleTab === "new-tournament" && <TourneyEditor mode="new" />}
            {visibleTab === "announcement-editor" && <AnnouncementEditor />}
            {visibleTab === "puzzle-generator" && <PuzzleGenerator />}
            {visibleTab === "user-details" && <UserDetails />}
            {visibleTab === "perms-and-roles" && <PermsAndRoles />}
            {visibleTab === "badges" && <UserBadges />}
            {visibleTab === "verification-queue" && <VerificationQueue />}
            {visibleTab === "manual-org-assignment" && <ManualOrgAssignment />}
            {/* {visibleTab === 'user-editor' && <UserEditor />} */}
            {/* {visibleTab === 'manage-tournament' && <TourneyManager />} */}
          </Layout.Content>
        </Layout>
      </Layout>
    </>
  );
};
