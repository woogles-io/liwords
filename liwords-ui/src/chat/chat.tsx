import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Card, Input, Tabs } from "antd";
import { LeftOutlined } from "@ant-design/icons";
import { singularCount } from "../utils/plural";
import { ChatEntity } from "./chat_entity";
import {
  useChatStoreContext,
  useLoginStateStoreContext,
  usePresenceStoreContext,
} from "../store/store";
import "./chat.scss";
import { Presences } from "./presences";
import { ChatChannels } from "./chat_channels";
import {
  ChatEntityObj,
  ChatEntityType,
  chatMessageToChatEntity,
} from "../store/constants";
import { Players } from "./players";
import { ChatMessage } from "../gen/api/proto/ipc/chat_pb";
import { useClient } from "../utils/hooks/connect";
import { SocializeService } from "../gen/api/proto/user_service/user_service_pb";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { useCollectionContext } from "../collections/useCollectionContext";
import { CollectionNavigationTab } from "../collections/CollectionNavigationTab";

export type Props = {
  sendChat: (msg: string, chan: string) => void;
  defaultChannel: string;
  defaultDescription: string;
  DISCONNECT?: () => void;
  //For chat accessible places without channels
  channelTypeOverride?: string;
  highlight?: Array<string>;
  highlightText?: string;
  tournamentID?: string;
  suppressDefault?: boolean;
};

// userid -> channel -> string
let globalUnsentChatCache: { [key: string]: { [key: string]: string } } = {};

export const Chat = React.memo((props: Props) => {
  const { loginState } = useLoginStateStoreContext();
  const competitorState = useTournamentCompetitorState();
  const { loggedIn, userID } = loginState;
  const collectionContext = useCollectionContext();
  const [hasScroll, setHasScroll] = useState(false);
  const [channelsFetched, setChannelsFetched] = useState(false);
  const [presenceVisible, setPresenceVisible] = useState(false);
  // We cannot useRef because we need to awoken effects relying on the element.
  const [tabContainerElement, setTabContainerElement] =
    useState<HTMLDivElement | null>(null);
  const { defaultChannel, defaultDescription } = props;
  const [showChannels, setShowChannels] = useState(props.suppressDefault);
  const propsSendChat = useMemo(() => props.sendChat, [props.sendChat]);
  const [selectedChatTab, setSelectedChatTab] = useState(() => {
    if (props.suppressDefault || !loggedIn) return "CHAT";
    // Collection takes precedence over game chat
    if (collectionContext) return "COLLECTION";
    // Check if we're in a game or gametv channel
    const channelType = defaultChannel?.split(".")[1];
    if (channelType === "game" || channelType === "gametv") return "CHAT";
    return "PLAYERS";
  });
  const chatTab = selectedChatTab === "CHAT" ? tabContainerElement : null;
  const socializeClient = useClient(SocializeService);
  // Chat auto-scrolls when the last entity is visible.
  const [hasUnreadChat, setHasUnreadChat] = useState(false);
  const {
    chat: chatEntities,
    clearChat,
    addChat,
    addChats,
    setChatChannels,
  } = useChatStoreContext();
  const { presences } = usePresenceStoreContext();
  const [presenceCount, setPresenceCount] = useState(0);
  const lastChannel = useRef("");
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
  const [channel, setChannel] = useState<string | undefined>(
    !props.suppressDefault ? defaultChannel : undefined,
  );
  const [, setRefreshCurMsg] = useState(0);
  const channelType = useMemo(() => {
    return channel?.split(".")[1] || "";
  }, [channel]);

  const canonicalizedChannel = useMemo(() => {
    switch (channelType) {
      case "gametv":
        return "gametv";
      case "game":
        return "game";
      default:
        return channel;
    }
  }, [channelType, channel]);

  const setCurMsg = useCallback(
    (x: string) => {
      if (!canonicalizedChannel) {
        // cannot store it
        return;
      }
      if (!loggedIn) {
        // do not clear cache if they briefly disconnect
        return;
      }
      if (!(userID in globalUnsentChatCache)) {
        // if they log in as someone else, flush the former user's cache
        globalUnsentChatCache = { [userID]: {} };
      }
      globalUnsentChatCache[userID][canonicalizedChannel] = x;
      setRefreshCurMsg((n) => (n + 1) | 0); // trigger refresh
    },
    [loggedIn, userID, canonicalizedChannel],
  );
  const curMsg =
    (loggedIn &&
      userID &&
      canonicalizedChannel &&
      globalUnsentChatCache[userID]?.[canonicalizedChannel]) ||
    "";
  const [maxEntitiesHeight, setMaxEntitiesHeight] = useState<
    number | undefined
  >(undefined);
  const [description, setDescription] = useState(defaultDescription);
  const [defaultLastMessage, setDefaultLastMessage] = useState("");
  const [channelSelectedTime, setChannelSelectedTime] = useState(Date.now());
  const [channelReadTime, setChannelReadTime] = useState(Date.now());
  const [notificationCount, setNotificationCount] = useState<number>(0);
  const [lastNotificationTimestamp, setLastNotificationTimestamp] =
    useState<number>(Date.now());
  // Messages that come in for other channels
  const [unseenMessages, setUnseenMessages] = useState(
    new Array<ChatEntityObj>(),
  );
  // Channels other than the current that are flagged hasUpdate. Each one is removed
  // if we switch to it
  const [updatedChannels, setUpdatedChannels] = useState<
    Set<string> | undefined
  >(undefined);
  const onChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setCurMsg(e.target.value);
    },
    [setCurMsg],
  );

  const autoScrollTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const doChatAutoScroll = useCallback(
    (force = false) => {
      if ((chatAutoScroll || force) && chatTab) {
        // Clear any existing timeout
        if (autoScrollTimeoutRef.current) {
          clearTimeout(autoScrollTimeoutRef.current);
        }
        // Slight delay on this to let entities load, now that they're xhr
        autoScrollTimeoutRef.current = setTimeout(() => {
          if (chatTab.scrollHeight > chatTab.clientHeight) {
            setHasScroll(true);
          }
          const desiredScrollTop = chatTab.scrollHeight - chatTab.clientHeight;
          // Slight wiggle room, since close enough to the bottom is close enough.
          // Otherwise it bounces on rounding errors for later changes to things that call this
          if (chatTab.scrollTop < desiredScrollTop - 6) {
            chatTab.scrollTop = desiredScrollTop;
          }
          setHasUnreadChat(false);
          autoScrollTimeoutRef.current = null;
        }, 100);
      }
    },
    [chatAutoScroll, chatTab],
  );

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById("chat")?.clientHeight;
    const contextPanelHeight =
      document.getElementById("chat-context")?.clientHeight;
    const calculatedEntitiesHeight =
      contextPanelHeight && tabPaneHeight
        ? tabPaneHeight -
          (contextPanelHeight > 106 ? contextPanelHeight : 106) -
          100
        : undefined;
    if (maxEntitiesHeight !== calculatedEntitiesHeight) {
      setMaxEntitiesHeight(calculatedEntitiesHeight);
      doChatAutoScroll();
    }
  }, [doChatAutoScroll, maxEntitiesHeight]);

  const sendNewMessage = useCallback(
    (receiverId: string, username: string) => {
      const calculatePMChannel = (receiverID: string) => {
        let u1 = loginState.userID;
        let u2 = receiverID;
        if (u2 < u1) {
          [u1, u2] = [u2, u1];
        }
        return `chat.pm.${u1}_${u2}`;
      };
      if (loginState.loggedIn) {
        setSelectedChatTab("CHAT");
        setChannel(calculatePMChannel(receiverId));
        setDescription(`Chat with ${username}`);
        setShowChannels(false);
      } else {
        console.log(loginState);
      }
    },
    [loginState],
  );

  useEffect(() => {
    setHeight();
  }, [
    updatedChannels,
    unseenMessages,
    presenceCount,
    competitorState,
    setHeight,
  ]);

  // When window is shrunk, auto-scroll may be enabled. This is one-way.
  // Hiding bookmarks bar or downloaded files bar should keep it enabled.
  const enableChatAutoScroll = useCallback(() => {
    const tab = document.getElementById("chat");
    if (tab && tab?.scrollTop >= tab?.scrollHeight - tab?.clientHeight) {
      setChatAutoScroll(true);
    }
    // When window is shrunk, keep the bottom entity instead of the top.
    doChatAutoScroll();
    setHeight();
  }, [doChatAutoScroll, setHeight]);

  const fetchChannels = useCallback(async () => {
    if (loggedIn) {
      const initial = !channelsFetched;
      if (initial) {
        setChannelsFetched(true);
      }
      const resp = await socializeClient.getActiveChatChannels({
        number: 20,
        offset: 0,
        tournamentId: props.tournamentID || "",
      });

      console.log("Fetched channels:", resp.channels);
      setChatChannels(resp);
      enableChatAutoScroll();
      if (initial) {
        // If these were set already, just return that list,
        // otherwise respect the hasUpdate fields
        const newUpdatedChannels = new Set(
          resp?.channels
            ?.filter((ch) => ch.hasUpdate)
            ?.map((ch) => {
              return ch.name;
            }),
        );
        setUpdatedChannels(newUpdatedChannels);
        setNotificationCount(newUpdatedChannels.size);
      }
    }
  }, [
    setChatChannels,
    enableChatAutoScroll,
    channelsFetched,
    loggedIn,
    setNotificationCount,
    socializeClient,
    props.tournamentID,
  ]);

  useEffect(() => {
    // Initial load of channels
    if (!channelsFetched) {
      fetchChannels();
    }
  }, [fetchChannels, channelsFetched]);

  useEffect(() => {
    setPresenceCount(presences.filter((p) => p.channel === channel).length);
  }, [presences, channel]);

  useEffect(() => {
    if (chatTab) {
      // Designs changes when the chat is long enough to scroll
      if (chatTab.scrollHeight > chatTab.clientHeight) {
        setHasScroll(true);
      }
      doChatAutoScroll();
    }
  }, [chatTab, doChatAutoScroll, chatEntities]);

  useEffect(() => {
    setChannel(defaultChannel);
    setDescription(defaultDescription);
    if (
      loggedIn &&
      (defaultChannel === "chat.lobby" || props.suppressDefault)
    ) {
      setSelectedChatTab("PLAYERS");
      setShowChannels(true);
    }
  }, [defaultChannel, defaultDescription, loggedIn, props.suppressDefault]);

  // Track context to detect actual changes (not just re-renders)
  const [prevContext, setPrevContext] = useState({
    tournamentID: props.tournamentID,
    channelType,
    hasCollection: !!collectionContext,
  });

  // Only auto-switch tabs when context actually changes
  useEffect(() => {
    const currentContext = {
      tournamentID: props.tournamentID,
      channelType,
      hasCollection: !!collectionContext,
    };

    // Check if context actually changed
    const contextChanged =
      prevContext.tournamentID !== currentContext.tournamentID ||
      prevContext.channelType !== currentContext.channelType ||
      prevContext.hasCollection !== currentContext.hasCollection;

    if (contextChanged) {
      if (
        props.tournamentID ||
        channelType === "game" ||
        channelType === "gametv"
      ) {
        setSelectedChatTab("CHAT");
      } else if (collectionContext) {
        setSelectedChatTab("COLLECTION");
      }
      setPrevContext(currentContext);
    }
  }, [props.tournamentID, channelType, collectionContext, prevContext]);

  const decoratedDescription = useMemo(() => {
    switch (channelType) {
      case "game":
        return `Game chat with ${description}`;
      case "gametv":
        return `Game chat for ${description}`;
    }
    return description;
  }, [description, channelType]);

  useEffect(() => {
    // Remove this channel's messages from the unseen list when we switch back to message view
    setUnseenMessages((u) => u.filter((ch) => ch.channel !== channel));
    setUpdatedChannels((u) => {
      if (u) {
        const initialUpdated = Array.from(u);
        return new Set(initialUpdated.filter((ch) => ch !== channel));
      }
      return undefined;
    });
  }, [channel, showChannels]);

  useEffect(() => {
    if (chatTab || showChannels) {
      // chat entities changed.
      // Update defaultLastMessage

      const lastMessage = chatEntities.reduce(
        (acc: ChatEntityObj | undefined, ch) =>
          ch.channel === defaultChannel &&
          ch.timestamp &&
          (acc === undefined || (acc.timestamp && ch.timestamp > acc.timestamp))
            ? ch
            : acc,
        undefined,
      );

      setDefaultLastMessage((u) =>
        lastMessage ? `${lastMessage?.sender}: ${lastMessage?.message}` : u,
      );
      // If there are new messages in this
      // channel and we've scrolled up, mark this chat unread,
      const currentUnread = chatEntities
        .filter((ch) => {
          return (
            ch.channel === channel &&
            ch.timestamp &&
            ch.timestamp > channelReadTime
          );
        })
        .sort((chA, chB) => {
          if (chB.timestamp && chA.timestamp) {
            return chB.timestamp > chA.timestamp
              ? 1
              : chB.timestamp < chA.timestamp
                ? -1
                : 0;
          }
          return 0;
        });
      if (currentUnread.length && chatTab) {
        const hasUnread =
          chatTab.scrollTop + 6 < chatTab.scrollHeight - chatTab.clientHeight;
        setHasUnreadChat(hasUnread);
        setChannelReadTime(
          (u) => Number(currentUnread[currentUnread.length - 1].timestamp) || u,
        );
      }
      // If they're for other channels or we're on the channel screen
      // add them to the unseenMessage,
      // to indicate unread in the channels component
      const otherUnreads = chatEntities
        .filter(
          // Only the ones since we switched to this channel
          (ch) =>
            (ch.channel !== channel || showChannels) &&
            ch.timestamp &&
            ch.timestamp > channelSelectedTime,
        )
        .filter(
          (ch) =>
            // And not our own
            ch.senderId !== userID,
        );
      if (otherUnreads.length) {
        setUnseenMessages((u) =>
          u.concat(otherUnreads).filter((ch) => ch.channel !== channel),
        );
        if (showChannels) {
          // if we have unread messages while looking at the channels, refetch them
          fetchChannels();
        }
      }
    }
  }, [
    chatTab,
    chatEntities,
    channel,
    defaultChannel,
    fetchChannels,
    channelReadTime,
    channelSelectedTime,
    showChannels,
    userID,
  ]);

  useEffect(() => {
    if (selectedChatTab === "PLAYERS") {
      const lastMessage = chatEntities.reduce(
        (acc: ChatEntityObj | undefined, ch) =>
          ch.timestamp &&
          (acc === undefined || (acc.timestamp && ch.timestamp > acc.timestamp))
            ? ch
            : acc,
        undefined,
      );
      if (
        lastMessage?.channel !== "chat.lobby" &&
        (lastMessage?.timestamp || 0) > lastNotificationTimestamp
      ) {
        setNotificationCount((x) => x + 1);
        setLastNotificationTimestamp((x) =>
          (Number(lastMessage?.timestamp) || 0) > x
            ? Number(lastMessage?.timestamp) || x
            : x,
        );
      }
    }
  }, [
    chatEntities,
    selectedChatTab,
    lastNotificationTimestamp,
    setNotificationCount,
  ]);

  useEffect(() => {
    window.addEventListener("resize", enableChatAutoScroll);
    return () => {
      window.removeEventListener("resize", enableChatAutoScroll);
    };
  }, [enableChatAutoScroll]);

  // Cleanup timeout on component unmount
  useEffect(() => {
    return () => {
      if (autoScrollTimeoutRef.current) {
        clearTimeout(autoScrollTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    // If we actually changed the channel, get the new messages
    if (channel && channel !== lastChannel.current) {
      lastChannel.current = channel || "";
      setChannelSelectedTime(Date.now());
      const fetchChats = async () => {
        const chats = await socializeClient.getChatsForChannel({ channel });

        clearChat();
        const messages: Array<ChatMessage> = chats.messages;
        if (messages) {
          addChats(messages.map(chatMessageToChatEntity));
        }
        setHasUnreadChat(false);
        setChatAutoScroll(true);
        setHeight();
      };
      fetchChats();
    } else {
      setHeight();
    }
  }, [channel, addChats, clearChat, loggedIn, setHeight, socializeClient]);

  // When user is scrolling, auto-scroll may be enabled or disabled.
  // This handler is set through onScroll.
  const handleChatScrolled = useCallback(() => {
    if (chatTab) {
      // Allow for 12 pixels of wiggle room for enabling auto scroll
      if (
        chatTab.scrollTop + 12 >=
        chatTab.scrollHeight - chatTab.clientHeight
      ) {
        setHasUnreadChat(false);
        setChatAutoScroll(true);
      } else {
        setChatAutoScroll(false);
      }
    }
  }, [chatTab]);

  const entities = useMemo(
    () =>
      chatEntities
        ?.filter((ent: ChatEntityObj) => {
          if (ent.entityType === ChatEntityType.UserChat) {
            return ent.channel === channel;
          }
          return true;
        })
        .map((ent) => {
          const specialSender = props.highlight
            ? props.highlight.includes(ent.sender)
            : false;
          if (!ent.id) {
            return null;
          }
          return (
            <ChatEntity
              entityType={ent.entityType}
              key={ent.id}
              msgID={ent.id}
              sender={ent.sender}
              senderId={ent.senderId}
              message={ent.message}
              channel={ent.channel}
              timestamp={ent.timestamp}
              highlight={specialSender}
              highlightText={props.highlightText}
              sendMessage={sendNewMessage}
            />
          );
        }),
    [
      chatEntities,
      props.highlight,
      props.highlightText,
      channel,
      sendNewMessage,
    ],
  );

  const handleTabClick = useCallback((key: string) => {
    setSelectedChatTab(key);
    setNotificationCount(0);
    setLastNotificationTimestamp(Date.now());
  }, []);
  const handleHideList = useCallback(() => {
    setPresenceVisible(false);
  }, []);
  const handleShowList = useCallback(() => {
    setPresenceVisible(true);
  }, []);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter" && channel) {
        e.preventDefault();
        // Send if non-trivial
        const msg = curMsg.trim();
        setCurMsg("");

        if (msg === "") {
          return;
        }
        if (!loggedIn) {
          return;
        }
        propsSendChat(msg, channel);
        // This may not be a good idea. User will miss unread messages.
        setChatAutoScroll(true);
        doChatAutoScroll();
      }
    },
    [curMsg, doChatAutoScroll, loggedIn, propsSendChat, channel, setCurMsg],
  );

  const gameChannel = useMemo(
    () =>
      defaultChannel?.startsWith("chat.game.") ? defaultChannel : undefined,
    [defaultChannel],
  );
  const gameChannelPresenceCount = useMemo(
    () =>
      gameChannel
        ? presences.reduce(
            (count, p) => count + +!!(p.channel === gameChannel),
            0,
          )
        : 0,
    [presences, gameChannel],
  );
  const [laggedGameChannelPresenceCount, setLaggedGameChannelPresenceCount] =
    useState(0);
  useEffect(() => {
    const t = setTimeout(() => {
      // lag this update to allow opponent to refresh the window.
      setLaggedGameChannelPresenceCount(gameChannelPresenceCount);
    }, 5000);
    return () => {
      clearTimeout(t);
    };
  }, [gameChannelPresenceCount]);
  const checkOpponentPresence = useRef<{
    channel: string | undefined;
    count: number;
  }>({ channel: undefined, count: 0 });
  useEffect(() => {
    if (
      laggedGameChannelPresenceCount > 0 &&
      checkOpponentPresence.current.channel !== gameChannel
    ) {
      checkOpponentPresence.current = {
        channel: gameChannel,
        count: laggedGameChannelPresenceCount,
      };
    }
  }, [gameChannel, laggedGameChannelPresenceCount]);
  useEffect(() => {
    if (gameChannel && checkOpponentPresence.current.channel === gameChannel) {
      const additionalCount =
        laggedGameChannelPresenceCount - checkOpponentPresence.current.count;
      // XXX: if this happens while browsing available chat channels, the message is lost.
      if (additionalCount > 0) {
        addChat({
          entityType: ChatEntityType.ServerMsg,
          sender: "",
          message: "Opponent has returned to this room.",
          channel: "server",
        });
      } else if (additionalCount < 0) {
        addChat({
          entityType: ChatEntityType.ErrorMsg,
          sender: "",
          message: "Opponent is no longer in this room.",
          channel: "server",
        });
      }
      checkOpponentPresence.current.count = laggedGameChannelPresenceCount;
    }
  }, [addChat, gameChannel, laggedGameChannelPresenceCount]);
  const peopleOnlineCounter = useMemo(
    () =>
      channel?.startsWith("chat.gametv.")
        ? singularCount(presenceCount, "Observer", "Observers")
        : singularCount(presenceCount, "Player", "Players"),
    [channel, presenceCount],
  );

  // Memoize just the tab keys/labels structure, render content inline
  const tabItems = useMemo(() => {
    const items = [];

    if (collectionContext) {
      items.push({
        label: "Collection",
        key: "COLLECTION",
      });
    }

    items.push({
      label: "Players",
      key: "PLAYERS",
    });

    items.push({
      label: (
        <>
          Chat
          {notificationCount > 0 && (
            <span className="notification">
              {notificationCount < 10 ? notificationCount : "!"}
            </span>
          )}
        </>
      ),
      key: "CHAT",
    });

    return items;
  }, [collectionContext, notificationCount]);

  // Add children based on selected tab to avoid re-rendering all tabs
  const tabItemsWithChildren = tabItems.map((item) => {
    if (item.key === "COLLECTION") {
      return {
        ...item,
        children: (
          <div className="collection-pane">
            <CollectionNavigationTab />
          </div>
        ),
      };
    }

    if (item.key === "PLAYERS") {
      return {
        ...item,
        children: (
          <div className="player-pane">
            <Players
              sendMessage={sendNewMessage}
              defaultChannelType={
                props.channelTypeOverride || defaultChannel.split(".")[1] || ""
              }
            />
          </div>
        ),
      };
    }

    if (item.key === "CHAT") {
      return {
        ...item,
        children: showChannels ? (
          <ChatChannels
            defaultChannel={defaultChannel}
            defaultDescription={defaultDescription}
            defaultLastMessage={defaultLastMessage}
            updatedChannels={updatedChannels}
            unseenMessages={unseenMessages}
            onChannelSelect={(ch: string, desc: string) => {
              if (channel !== ch) {
                setChannel(ch);
                setDescription(desc);
              }
              setShowChannels(false);
            }}
            sendMessage={sendNewMessage}
            suppressDefault={props.suppressDefault}
            tournamentID={props.tournamentID}
          />
        ) : (
          channel && (
            <>
              <div
                id="chat-context"
                className={`chat-context${hasScroll ? " scrolling" : ""}`}
              >
                {loggedIn ? (
                  <p
                    className={`breadcrumb clickable${
                      (updatedChannels && updatedChannels.size > 0) ||
                      unseenMessages.length > 0
                        ? " unread"
                        : ""
                    }`}
                    onClick={() => {
                      setChannel(undefined);
                      setChannelSelectedTime(Date.now());
                      setShowChannels(true);
                      fetchChannels();
                    }}
                  >
                    <LeftOutlined /> All chats
                    {((updatedChannels && updatedChannels.size > 0) ||
                      unseenMessages.length > 0) && (
                      <span className="unread-marker">•</span>
                    )}
                  </p>
                ) : null}
                <p data-testid="description" className="description">
                  {decoratedDescription}
                  {hasUnreadChat && (
                    <span className="unread-marker" data-testid="description">
                      •
                    </span>
                  )}
                </p>
                {presenceCount && !channel.startsWith("chat.pm.") ? (
                  <>
                    <p className="presence-count">
                      <span>{peopleOnlineCounter}</span>
                      {presenceVisible ? (
                        <span className="list-trigger" onClick={handleHideList}>
                          Hide list
                        </span>
                      ) : (
                        <span className="list-trigger" onClick={handleShowList}>
                          Show list
                        </span>
                      )}
                    </p>
                    {presenceVisible ? (
                      <p className="presence">
                        <Presences
                          players={presences}
                          channel={channel}
                          sendMessage={sendNewMessage}
                        />
                      </p>
                    ) : null}
                  </>
                ) : null}
              </div>
              {defaultChannel === "chat.lobby" && channel === "chat.lobby" ? (
                <React.Fragment key="chat-disabled">
                  <p className="disabled-message">Help chat is disabled.</p>
                </React.Fragment>
              ) : (
                <React.Fragment key="chat-enabled">
                  <div
                    className={`entities ${channelType}`}
                    style={
                      maxEntitiesHeight
                        ? {
                            maxHeight: maxEntitiesHeight,
                          }
                        : undefined
                    }
                    ref={setTabContainerElement}
                    onScroll={handleChatScrolled}
                  >
                    {entities}
                  </div>
                  <form>
                    <Input
                      autoFocus={!defaultChannel.startsWith("chat.game")}
                      autoComplete="off"
                      placeholder={
                        channel === "chat.lobby"
                          ? "Ask or answer question..."
                          : "chat..."
                      }
                      name="chat-input"
                      disabled={!loggedIn}
                      onKeyDown={onKeyDown}
                      onChange={onChange}
                      value={curMsg}
                      spellCheck={false}
                    />
                  </form>
                </React.Fragment>
              )}
            </>
          )
        ),
      };
    }

    return item;
  });

  // Ensure selected tab is valid when tabItems change
  useEffect(() => {
    const availableKeys = tabItems.map((item) => item.key);
    if (!availableKeys.includes(selectedChatTab)) {
      // If current selected tab is not available, pick first available tab
      setSelectedChatTab(availableKeys[0] || "CHAT");
    }
  }, [tabItems, selectedChatTab]);

  return (
    <Card className="chat" id="chat">
      <Tabs
        activeKey={selectedChatTab}
        centered
        onTabClick={handleTabClick}
        animated={false}
        items={tabItemsWithChildren}
        destroyInactiveTabPane={false} // Keep all tab content rendered
      />
    </Card>
  );
});
