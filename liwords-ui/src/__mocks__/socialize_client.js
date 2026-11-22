export default {
  getActiveChatChannels: vi
    .fn()
    .mockResolvedValue({ channels: [], hasMore: false, unreadChannels: [] }),
  getChatsForChannel: vi.fn().mockResolvedValue({ data: {} }),
  markChannelRead: vi.fn().mockResolvedValue({}),
};
