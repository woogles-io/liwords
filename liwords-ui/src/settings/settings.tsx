import React, { useEffect, useCallback, useState } from "react";
import { useParams } from "react-router";
import { App } from "antd";
import { TopBar } from "../navigation/topbar";
import { ChangePassword } from "./change_password";
import { PersonalInfoWidget } from "./personal_info";
import { CloseAccount } from "./close_account";
import { ClosedAccount } from "./closed_account";
import { Preferences } from "./preferences";
import { BlockedPlayers } from "./blocked_players";
import { LogOut } from "./log_out_woogles";
import { Support } from "./support_woogles";
import { useLoginStateStoreContext } from "../store/store";
import { useNavigate } from "react-router";
import { useResetStoreContext } from "../store/store";

import "./settings.scss";
import { Secret } from "./secret";
import { HeartFilled } from "@ant-design/icons";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import {
  connectErrorMessage,
  flashError,
  useClient,
} from "../utils/hooks/connect";
import {
  AuthenticationService,
  PersonalInfoRequestSchema,
  PersonalInfoResponseSchema,
  ProfileService,
} from "../gen/api/proto/user_service/user_service_pb";
import { PersonalInfoResponse } from "../gen/api/proto/user_service/user_service_pb";
import { API } from "./api";
import { create } from "@bufbuild/protobuf";
import { Integrations } from "./integrations";
import { RolesPermissions } from "./roles_permissions";

enum Category {
  PersonalInfo = 1,
  ChangePassword,
  Preferences,
  Integrations,
  BlockedPlayers,
  Secret,
  RolesPermissions,
  LogOut,
  Support,
  NoUser,
  API,
}

const getInitialCategory = (categoryShortcut: string, loggedIn: boolean) => {
  // We don't want to keep /donate or any other shortcuts in the url after reading it on first load
  // These are just shortcuts for backwards compatibility so we can send existing urls to the new
  // settings pages
  if (!loggedIn && categoryShortcut === "donate") {
    // Don't redirect if they aren't logged in and just donating
    return Category.Support;
  }
  window.history.replaceState({}, "settings", "/settings");
  switch (categoryShortcut) {
    case "donate":
    case "support":
      return Category.Support;
    case "personal":
      return Category.PersonalInfo;
    case "password":
      return Category.ChangePassword;
    case "preferences":
      return Category.Preferences;
    case "integrations":
      return Category.Integrations;
    case "secret":
      return Category.Secret;
    case "blocked":
      return Category.BlockedPlayers;
    case "logout":
      return Category.LogOut;
    case "api":
      return Category.API;
    case "roles":
      return Category.RolesPermissions;
  }
  // to be streaming-friendly, PersonalInfo should not be the default tab.
  return Category.Preferences;
};

export const Settings = React.memo(() => {
  const { loginState } = useLoginStateStoreContext();
  const { userID, username: viewer, loggedIn } = loginState;
  const { resetStore } = useResetStoreContext();
  const { notification } = App.useApp();
  let { section } = useParams();
  if (!section) {
    section = "";
  }
  const [category, setCategory] = useState(
    getInitialCategory(section, loggedIn),
  );
  const [player, setPlayer] = useState<Partial<PlayerInfo> | undefined>(
    undefined,
  );
  const [personalInfo, setPersonalInfo] = useState<
    PersonalInfoResponse | undefined
  >(undefined);
  const [showCloseAccount, setShowCloseAccount] = useState(false);
  const [showClosedAccount, setShowClosedAccount] = useState(false);
  const [accountClosureError, setAccountClosureError] = useState("");
  const navigate = useNavigate();

  const profileClient = useClient(ProfileService);
  const authClient = useClient(AuthenticationService);

  useEffect(() => {
    if (viewer === "" || (!loggedIn && category === Category.Support)) return;
    if (!loggedIn) {
      setCategory(Category.NoUser);
      return;
    }
    (async () => {
      try {
        const resp = await profileClient.getPersonalInfo(
          create(PersonalInfoRequestSchema, {}),
        );

        setPlayer({
          fullName: resp.fullName,
          userId: userID,
        });

        setPersonalInfo(resp);
      } catch (e) {
        flashError(e);
      }
    })();
  }, [viewer, loggedIn, category, profileClient, userID]);

  type CategoryProps = {
    title: string | React.ReactNode;
    category: Category;
  };

  const CategoryChoice = React.memo((props: CategoryProps) => {
    return (
      <div
        className={category === props.category ? "choice active" : "choice"}
        onClick={() => {
          setCategory(props.category);
        }}
      >
        {props.title}
      </div>
    );
  });

  const handleContribute = useCallback(() => {
    navigate("/settings");
  }, [navigate]);

  const handleLogout = useCallback(async () => {
    try {
      await authClient.logout({});
      notification.info({
        message: "Success",
        description: "You have been logged out.",
      });
      resetStore();
      navigate("/");
    } catch (e) {
      flashError(e);
    }
  }, [authClient, navigate, resetStore, notification]);

  const updatedAvatar = useCallback(
    (avatarUrl: string) => {
      if (!personalInfo) {
        return;
      }
      setPersonalInfo(
        create(PersonalInfoResponseSchema, {
          ...personalInfo,
          avatarUrl: avatarUrl,
        }),
      );
    },
    [personalInfo],
  );

  const startClosingAccount = useCallback(() => {
    setAccountClosureError("");
    setShowCloseAccount(true);
  }, []);

  const closeAccountNow = useCallback(
    async (pw: string) => {
      try {
        await authClient.notifyAccountClosure({ password: pw });
        setShowCloseAccount(false);
        setShowClosedAccount(true);
        handleLogout();
      } catch (e) {
        setAccountClosureError(connectErrorMessage(e));
      }
    },
    [authClient, handleLogout],
  );

  const logIn = <div className="log-in">Log in to see your settings</div>;

  const categoriesColumn = (
    <div className="categories">
      <CategoryChoice title="Personal info" category={Category.PersonalInfo} />
      <CategoryChoice
        title="Change password"
        category={Category.ChangePassword}
      />
      <CategoryChoice title="Integrations" category={Category.Integrations} />
      <CategoryChoice title="Preferences" category={Category.Preferences} />
      <CategoryChoice
        title="Blocked players list"
        category={Category.BlockedPlayers}
      />
      <CategoryChoice title="Secret features" category={Category.Secret} />
      <CategoryChoice title="API" category={Category.API} />
      <CategoryChoice
        title="Roles & Permissions"
        category={Category.RolesPermissions}
      />
      <CategoryChoice title="Log out" category={Category.LogOut} />
      <CategoryChoice
        title={
          <>
            <HeartFilled />
            Support Woogles
          </>
        }
        category={Category.Support}
      />
    </div>
  );

  return (
    <>
      <TopBar />
      {loggedIn ? (
        <div className="settings">
          {categoriesColumn}
          <div className="category">
            {category === Category.PersonalInfo ? (
              showCloseAccount ? (
                <CloseAccount
                  closeAccountNow={closeAccountNow}
                  player={player}
                  err={accountClosureError}
                  cancel={() => {
                    setShowCloseAccount(false);
                  }}
                />
              ) : showClosedAccount ? (
                <ClosedAccount />
              ) : personalInfo ? (
                <PersonalInfoWidget
                  player={player}
                  personalInfo={personalInfo}
                  updatedAvatar={updatedAvatar}
                  startClosingAccount={startClosingAccount}
                />
              ) : null
            ) : null}
            {category === Category.ChangePassword ? <ChangePassword /> : null}
            {category === Category.Preferences ? <Preferences /> : null}
            {category === Category.Integrations ? <Integrations /> : null}
            {category === Category.Secret ? <Secret /> : null}
            {category === Category.BlockedPlayers ? <BlockedPlayers /> : null}
            {category === Category.API ? <API /> : null}
            {category === Category.RolesPermissions ? (
              <RolesPermissions />
            ) : null}
            {category === Category.LogOut ? (
              <LogOut player={player} handleLogout={handleLogout} />
            ) : null}
            {category === Category.Support ? (
              <Support handleContribute={handleContribute} />
            ) : null}
            {category === Category.NoUser ? logIn : null}
          </div>
        </div>
      ) : null}
      {!loggedIn && category === Category.Support ? (
        <div className="settings stand-alone">
          <Support handleContribute={handleContribute} />
        </div>
      ) : (
        !loggedIn && <div className="settings loggedOut">{logIn}</div>
      )}
    </>
  );
});
