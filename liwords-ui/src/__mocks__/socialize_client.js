export default {
  getActiveChatChannels: vi.fn().mockResolvedValue({ data: {} }),
  getChatsForChannel: vi.fn().mockResolvedValue({ data: {} }),
  markChannelRead: vi.fn().mockResolvedValue({}),
};
