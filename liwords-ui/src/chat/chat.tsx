import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Card, Input, Tabs, notification } from "antd";
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
import { ChatMessage, ChatMessageSchema } from "../gen/api/proto/ipc/chat_pb";
import { useClient } from "../utils/hooks/connect";
import { SocializeService } from "../gen/api/proto/user_service/user_service_pb";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { useCollectionContext } from "../collections/useCollectionContext";
import { CollectionNavigationTab } from "../collections/CollectionNavigationTab";
import { create, toBinary } from "@bufbuild/protobuf";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";
import { encodeToSocketFmt } from "../utils/protobuf";

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
  leagueID?: string;
  suppressDefault?: boolean;
};

// userid -> channel -> string
let globalUnsentChatCache: { [key: string]: { [key: string]: string } } = {};

const MAX_MESSAGE_LENGTH = 500;
const MAX_WEBSOCKET_MESSAGE_SIZE = 1800; // Conservative limit (backend is 2048)

// Helper function to determine the correct default tab
const getDefaultTab = (
  suppressDefault: boolean,
  loggedIn: boolean,
  collectionContext: unknown,
  defaultChannel: string,
  tournamentID?: string,
): string => {
  if (suppressDefault || !loggedIn) return "CHAT";

  // Collection takes precedence over everything else
  if (collectionContext) return "COLLECTION";

  // Lobby should default to PLAYERS
  if (defaultChannel === "chat.lobby") return "PLAYERS";

  // Check channel type
  const channelType = defaultChannel?.split(".")[1];

  // Tournament channels, league channels, game channels, and gametv channels should default to CHAT
  if (
    tournamentID ||
    channelType === "tournament" ||
    channelType === "league" ||
    channelType === "game" ||
    channelType === "gametv"
  ) {
    return "CHAT";
  }

  // Default to PLAYERS for other channels
  return "PLAYERS";
};

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
  const [selectedChatTab, setSelectedChatTab] = useState(() =>
    getDefaultTab(
      props.suppressDefault || false,
      loggedIn,
      collectionContext,
      defaultChannel,
      props.tournamentID,
    ),
  );
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
  const lastChannel = useRef("");
  const [chatAutoScroll, setChatAutoScroll] = useState(true);
  const [channel, setChannel] = useState<string | undefined>(
    !props.suppressDefault ? defaultChannel : undefined,
  );
  const presenceCount = presences.filter((p) => p.channel === channel).length;
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
    competitorState,
    setHeight,
    presenceCount,
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
        leagueId: props.leagueID || "",
      });

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
    props.leagueID,
  ]);

  useEffect(() => {
    // Initial load of channels
    if (!channelsFetched) {
      fetchChannels();
    }
  }, [fetchChannels, channelsFetched]);

  useEffect(() => {
    if (chatTab) {
      // Designs changes when the chat is long enough to scroll
      if (chatTab.scrollHeight > chatTab.clientHeight) {
        setHasScroll(true);
      }
      doChatAutoScroll();
    }
  }, [chatTab, doChatAutoScroll, chatEntities]);

  // Track previous values to detect actual changes
  const prevDefaultChannelRef = useRef(defaultChannel);
  const prevCollectionContextRef = useRef(collectionContext);
  const prevLoggedInRef = useRef(loggedIn);

  useEffect(() => {
    // Handle default channel/description changes or login state changes
    if (
      defaultChannel !== prevDefaultChannelRef.current ||
      loggedIn !== prevLoggedInRef.current
    ) {
      setChannel(defaultChannel);
      setDescription(defaultDescription);
      prevDefaultChannelRef.current = defaultChannel;
      prevLoggedInRef.current = loggedIn;

      // Determine if we should switch tabs based on the new channel
      if (loggedIn) {
        const newTab = getDefaultTab(
          props.suppressDefault || false,
          loggedIn,
          collectionContext,
          defaultChannel,
          props.tournamentID,
        );
        setSelectedChatTab(newTab);
        setShowChannels(
          props.suppressDefault || defaultChannel === "chat.lobby",
        );
      }
    }

    // Handle collection context changes - only auto-switch when context appears/disappears
    if (collectionContext !== prevCollectionContextRef.current) {
      const wasNull = prevCollectionContextRef.current === null;
      const isNull = collectionContext === null;
      prevCollectionContextRef.current = collectionContext;

      // Only auto-switch to COLLECTION when context appears (null -> truthy)
      if (wasNull && !isNull) {
        setSelectedChatTab("COLLECTION");
      }
      // Only auto-switch away from COLLECTION when context disappears (truthy -> null)
      else if (!wasNull && isNull && selectedChatTab === "COLLECTION") {
        const newTab = getDefaultTab(
          props.suppressDefault || false,
          loggedIn,
          null,
          defaultChannel,
          props.tournamentID,
        );
        setSelectedChatTab(newTab);
      }
    }
  }, [
    defaultChannel,
    defaultDescription,
    collectionContext,
    loggedIn,
    props.suppressDefault,
    props.tournamentID,
    selectedChatTab,
  ]);

  const decoratedDescription = useMemo(() => {
    switch (channelType) {
      case "game":
        return `Game chat with ${description}`;
      case "gametv":
        return `Game chat for ${description}`;
    }
    return description;
  }, [description, channelType]);

  // Calculate defaultLastMessage from chat entities
  const defaultLastMessage = useMemo(() => {
    const lastMessage = chatEntities.reduce(
      (acc: ChatEntityObj | undefined, ch) =>
        ch.channel === defaultChannel &&
        ch.timestamp &&
        (acc === undefined || (acc.timestamp && ch.timestamp > acc.timestamp))
          ? ch
          : acc,
      undefined,
    );
    return lastMessage ? `${lastMessage?.sender}: ${lastMessage?.message}` : "";
  }, [chatEntities, defaultChannel]);

  useEffect(() => {
    if (chatTab || showChannels) {
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

  // Fetch chats when channel changes
  const handleChannelChange = useCallback(
    async (newChannel: string | undefined) => {
      if (newChannel && newChannel !== lastChannel.current) {
        lastChannel.current = newChannel;
        setChannelSelectedTime(Date.now());

        // Clear unseen messages for this channel
        setUnseenMessages((u) => u.filter((ch) => ch.channel !== newChannel));
        setUpdatedChannels((u) => {
          if (u) {
            const initialUpdated = Array.from(u);
            return new Set(initialUpdated.filter((ch) => ch !== newChannel));
          }
          return undefined;
        });

        const chats = await socializeClient.getChatsForChannel({
          channel: newChannel,
        });
        clearChat();
        const messages: Array<ChatMessage> = chats.messages;
        if (messages) {
          addChats(messages.map(chatMessageToChatEntity));
        }
        setHasUnreadChat(false);
        setChatAutoScroll(true);
        setHeight();
      } else {
        setHeight();
      }
    },
    [addChats, clearChat, setHeight, socializeClient],
  );

  useEffect(() => {
    // Only handle initial channel load
    if (channel && channel !== lastChannel.current) {
      handleChannelChange(channel);
    }
  }, [channel, handleChannelChange]);

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
          // HACK: Check for both exact match and :readonly suffix for directors
          // TODO: Replace with proper permissions field when backend schema is updated
          let specialSender = false;
          let isReadOnly = false;
          if (props.highlight) {
            specialSender = props.highlight.some(
              (name) =>
                name === ent.sender || name === `${ent.sender}:readonly`,
            );
            // Check if this sender is a read-only director
            isReadOnly = props.highlight.some(
              (name) => name === `${ent.sender}:readonly`,
            );
          }
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
              highlightText={isReadOnly ? "Invigilator" : props.highlightText}
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

        if (msg === "") {
          return;
        }
        if (!loggedIn) {
          return;
        }

        // Validate message length before clearing input
        if (msg.length > MAX_MESSAGE_LENGTH) {
          notification.error({
            message: "Message too long",
            description: `Messages must be ${MAX_MESSAGE_LENGTH} characters or less. Current length: ${msg.length}`,
          });
          return; // Keep message in input field
        }

        // Validate encoded message size to prevent WebSocket disconnect
        const evt = create(ChatMessageSchema, {
          message: msg,
          channel: channel,
        });
        const encodedMsg = encodeToSocketFmt(
          MessageType.CHAT_MESSAGE,
          toBinary(ChatMessageSchema, evt),
        );

        if (encodedMsg.length > MAX_WEBSOCKET_MESSAGE_SIZE) {
          notification.error({
            message: "Message too large",
            description: `Your message with metadata is ${encodedMsg.length} bytes, which exceeds the ${MAX_WEBSOCKET_MESSAGE_SIZE} byte limit. Please shorten your message.`,
          });
          return; // Keep message in input field
        }

        // Only clear input after validation passes
        setCurMsg("");
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
            onChannelSelect={async (ch: string, desc: string) => {
              if (channel !== ch) {
                setChannel(ch);
                setDescription(desc);
                await handleChannelChange(ch);
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
                    onClick={async () => {
                      setChannel(undefined);
                      setChannelSelectedTime(Date.now());
                      setShowChannels(true);
                      await fetchChannels();
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

  // Ensure selected tab is valid
  const availableKeys = tabItems.map((item) => item.key);
  const validatedSelectedTab = availableKeys.includes(selectedChatTab)
    ? selectedChatTab
    : availableKeys[0] || "CHAT";

  return (
    <Card className="chat" id="chat">
      <Tabs
        activeKey={validatedSelectedTab}
        centered
        onTabClick={handleTabClick}
        animated={false}
        items={tabItemsWithChildren}
        destroyInactiveTabPane={false} // Keep all tab content rendered
      />
    </Card>
  );
});
