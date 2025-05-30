// @generated by protoc-gen-es v2.5.1 with parameter "target=ts"
// @generated from file proto/config_service/config_service.proto (package config_service, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import { fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { file_google_protobuf_timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file proto/config_service/config_service.proto.
 */
export const file_proto_config_service_config_service: GenFile = /*@__PURE__*/
  fileDesc("Cilwcm90by9jb25maWdfc2VydmljZS9jb25maWdfc2VydmljZS5wcm90bxIOY29uZmlnX3NlcnZpY2UiJQoSRW5hYmxlR2FtZXNSZXF1ZXN0Eg8KB2VuYWJsZWQYASABKAgiIAoQU2V0RkVIYXNoUmVxdWVzdBIMCgRoYXNoGAEgASgJIhAKDkNvbmZpZ1Jlc3BvbnNlIjkKDEFubm91bmNlbWVudBINCgV0aXRsZRgBIAEoCRIMCgRsaW5rGAIgASgJEgwKBGJvZHkYAyABKAkiTgoXU2V0QW5ub3VuY2VtZW50c1JlcXVlc3QSMwoNYW5ub3VuY2VtZW50cxgBIAMoCzIcLmNvbmZpZ19zZXJ2aWNlLkFubm91bmNlbWVudCIZChdHZXRBbm5vdW5jZW1lbnRzUmVxdWVzdCJMChVBbm5vdW5jZW1lbnRzUmVzcG9uc2USMwoNYW5ub3VuY2VtZW50cxgBIAMoCzIcLmNvbmZpZ19zZXJ2aWNlLkFubm91bmNlbWVudCJuChxTZXRTaW5nbGVBbm5vdW5jZW1lbnRSZXF1ZXN0EjIKDGFubm91bmNlbWVudBgBIAEoCzIcLmNvbmZpZ19zZXJ2aWNlLkFubm91bmNlbWVudBIaChJsaW5rX3NlYXJjaF9zdHJpbmcYAiABKAkiSgobU2V0R2xvYmFsSW50ZWdyYXRpb25SZXF1ZXN0EhgKEGludGVncmF0aW9uX25hbWUYASABKAkSEQoJanNvbl9kYXRhGAIgASgJIjQKD0FkZEJhZGdlUmVxdWVzdBIMCgRjb2RlGAEgASgJEhMKC2Rlc2NyaXB0aW9uGAIgASgJIjQKEkFzc2lnbkJhZGdlUmVxdWVzdBIQCgh1c2VybmFtZRgBIAEoCRIMCgRjb2RlGAIgASgJIicKF0dldFVzZXJzRm9yQmFkZ2VSZXF1ZXN0EgwKBGNvZGUYASABKAkiKQoVR2V0VXNlckRldGFpbHNSZXF1ZXN0EhAKCHVzZXJuYW1lGAEgASgJIoUBChNVc2VyRGV0YWlsc1Jlc3BvbnNlEgwKBHV1aWQYASABKAkSDQoFZW1haWwYAiABKAkSKwoHY3JlYXRlZBgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXASEgoKYmlydGhfZGF0ZRgEIAEoCRIQCgh1c2VybmFtZRgFIAEoCSIrChJTZWFyY2hFbWFpbFJlcXVlc3QSFQoNcGFydGlhbF9lbWFpbBgBIAEoCSJJChNTZWFyY2hFbWFpbFJlc3BvbnNlEjIKBXVzZXJzGAEgAygLMiMuY29uZmlnX3NlcnZpY2UuVXNlckRldGFpbHNSZXNwb25zZSIeCglVc2VybmFtZXMSEQoJdXNlcm5hbWVzGAEgAygJMtkICg1Db25maWdTZXJ2aWNlElUKD1NldEdhbWVzRW5hYmxlZBIiLmNvbmZpZ19zZXJ2aWNlLkVuYWJsZUdhbWVzUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlEk0KCVNldEZFSGFzaBIgLmNvbmZpZ19zZXJ2aWNlLlNldEZFSGFzaFJlcXVlc3QaHi5jb25maWdfc2VydmljZS5Db25maWdSZXNwb25zZRJbChBTZXRBbm5vdW5jZW1lbnRzEicuY29uZmlnX3NlcnZpY2UuU2V0QW5ub3VuY2VtZW50c1JlcXVlc3QaHi5jb25maWdfc2VydmljZS5Db25maWdSZXNwb25zZRJnChBHZXRBbm5vdW5jZW1lbnRzEicuY29uZmlnX3NlcnZpY2UuR2V0QW5ub3VuY2VtZW50c1JlcXVlc3QaJS5jb25maWdfc2VydmljZS5Bbm5vdW5jZW1lbnRzUmVzcG9uc2UiA5ACARJlChVTZXRTaW5nbGVBbm5vdW5jZW1lbnQSLC5jb25maWdfc2VydmljZS5TZXRTaW5nbGVBbm5vdW5jZW1lbnRSZXF1ZXN0Gh4uY29uZmlnX3NlcnZpY2UuQ29uZmlnUmVzcG9uc2USYwoUU2V0R2xvYmFsSW50ZWdyYXRpb24SKy5jb25maWdfc2VydmljZS5TZXRHbG9iYWxJbnRlZ3JhdGlvblJlcXVlc3QaHi5jb25maWdfc2VydmljZS5Db25maWdSZXNwb25zZRJLCghBZGRCYWRnZRIfLmNvbmZpZ19zZXJ2aWNlLkFkZEJhZGdlUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlElEKC0Fzc2lnbkJhZGdlEiIuY29uZmlnX3NlcnZpY2UuQXNzaWduQmFkZ2VSZXF1ZXN0Gh4uY29uZmlnX3NlcnZpY2UuQ29uZmlnUmVzcG9uc2USUwoNVW5hc3NpZ25CYWRnZRIiLmNvbmZpZ19zZXJ2aWNlLkFzc2lnbkJhZGdlUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlElsKEEdldFVzZXJzRm9yQmFkZ2USJy5jb25maWdfc2VydmljZS5HZXRVc2Vyc0ZvckJhZGdlUmVxdWVzdBoZLmNvbmZpZ19zZXJ2aWNlLlVzZXJuYW1lcyIDkAIBEmEKDkdldFVzZXJEZXRhaWxzEiUuY29uZmlnX3NlcnZpY2UuR2V0VXNlckRldGFpbHNSZXF1ZXN0GiMuY29uZmlnX3NlcnZpY2UuVXNlckRldGFpbHNSZXNwb25zZSIDkAIBElsKC1NlYXJjaEVtYWlsEiIuY29uZmlnX3NlcnZpY2UuU2VhcmNoRW1haWxSZXF1ZXN0GiMuY29uZmlnX3NlcnZpY2UuU2VhcmNoRW1haWxSZXNwb25zZSIDkAIBQrgBChJjb20uY29uZmlnX3NlcnZpY2VCEkNvbmZpZ1NlcnZpY2VQcm90b1ABWjpnaXRodWIuY29tL3dvb2dsZXMtaW8vbGl3b3Jkcy9ycGMvYXBpL3Byb3RvL2NvbmZpZ19zZXJ2aWNlogIDQ1hYqgINQ29uZmlnU2VydmljZcoCDUNvbmZpZ1NlcnZpY2XiAhlDb25maWdTZXJ2aWNlXEdQQk1ldGFkYXRh6gINQ29uZmlnU2VydmljZWIGcHJvdG8z", [file_google_protobuf_timestamp]);

/**
 * @generated from message config_service.EnableGamesRequest
 */
export type EnableGamesRequest = Message<"config_service.EnableGamesRequest"> & {
  /**
   * @generated from field: bool enabled = 1;
   */
  enabled: boolean;
};

/**
 * Describes the message config_service.EnableGamesRequest.
 * Use `create(EnableGamesRequestSchema)` to create a new message.
 */
export const EnableGamesRequestSchema: GenMessage<EnableGamesRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 0);

/**
 * @generated from message config_service.SetFEHashRequest
 */
export type SetFEHashRequest = Message<"config_service.SetFEHashRequest"> & {
  /**
   * @generated from field: string hash = 1;
   */
  hash: string;
};

/**
 * Describes the message config_service.SetFEHashRequest.
 * Use `create(SetFEHashRequestSchema)` to create a new message.
 */
export const SetFEHashRequestSchema: GenMessage<SetFEHashRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 1);

/**
 * @generated from message config_service.ConfigResponse
 */
export type ConfigResponse = Message<"config_service.ConfigResponse"> & {
};

/**
 * Describes the message config_service.ConfigResponse.
 * Use `create(ConfigResponseSchema)` to create a new message.
 */
export const ConfigResponseSchema: GenMessage<ConfigResponse> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 2);

/**
 * @generated from message config_service.Announcement
 */
export type Announcement = Message<"config_service.Announcement"> & {
  /**
   * @generated from field: string title = 1;
   */
  title: string;

  /**
   * @generated from field: string link = 2;
   */
  link: string;

  /**
   * @generated from field: string body = 3;
   */
  body: string;
};

/**
 * Describes the message config_service.Announcement.
 * Use `create(AnnouncementSchema)` to create a new message.
 */
export const AnnouncementSchema: GenMessage<Announcement> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 3);

/**
 * @generated from message config_service.SetAnnouncementsRequest
 */
export type SetAnnouncementsRequest = Message<"config_service.SetAnnouncementsRequest"> & {
  /**
   * @generated from field: repeated config_service.Announcement announcements = 1;
   */
  announcements: Announcement[];
};

/**
 * Describes the message config_service.SetAnnouncementsRequest.
 * Use `create(SetAnnouncementsRequestSchema)` to create a new message.
 */
export const SetAnnouncementsRequestSchema: GenMessage<SetAnnouncementsRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 4);

/**
 * @generated from message config_service.GetAnnouncementsRequest
 */
export type GetAnnouncementsRequest = Message<"config_service.GetAnnouncementsRequest"> & {
};

/**
 * Describes the message config_service.GetAnnouncementsRequest.
 * Use `create(GetAnnouncementsRequestSchema)` to create a new message.
 */
export const GetAnnouncementsRequestSchema: GenMessage<GetAnnouncementsRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 5);

/**
 * @generated from message config_service.AnnouncementsResponse
 */
export type AnnouncementsResponse = Message<"config_service.AnnouncementsResponse"> & {
  /**
   * @generated from field: repeated config_service.Announcement announcements = 1;
   */
  announcements: Announcement[];
};

/**
 * Describes the message config_service.AnnouncementsResponse.
 * Use `create(AnnouncementsResponseSchema)` to create a new message.
 */
export const AnnouncementsResponseSchema: GenMessage<AnnouncementsResponse> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 6);

/**
 * @generated from message config_service.SetSingleAnnouncementRequest
 */
export type SetSingleAnnouncementRequest = Message<"config_service.SetSingleAnnouncementRequest"> & {
  /**
   * @generated from field: config_service.Announcement announcement = 1;
   */
  announcement?: Announcement;

  /**
   * @generated from field: string link_search_string = 2;
   */
  linkSearchString: string;
};

/**
 * Describes the message config_service.SetSingleAnnouncementRequest.
 * Use `create(SetSingleAnnouncementRequestSchema)` to create a new message.
 */
export const SetSingleAnnouncementRequestSchema: GenMessage<SetSingleAnnouncementRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 7);

/**
 * @generated from message config_service.SetGlobalIntegrationRequest
 */
export type SetGlobalIntegrationRequest = Message<"config_service.SetGlobalIntegrationRequest"> & {
  /**
   * @generated from field: string integration_name = 1;
   */
  integrationName: string;

  /**
   * @generated from field: string json_data = 2;
   */
  jsonData: string;
};

/**
 * Describes the message config_service.SetGlobalIntegrationRequest.
 * Use `create(SetGlobalIntegrationRequestSchema)` to create a new message.
 */
export const SetGlobalIntegrationRequestSchema: GenMessage<SetGlobalIntegrationRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 8);

/**
 * @generated from message config_service.AddBadgeRequest
 */
export type AddBadgeRequest = Message<"config_service.AddBadgeRequest"> & {
  /**
   * @generated from field: string code = 1;
   */
  code: string;

  /**
   * @generated from field: string description = 2;
   */
  description: string;
};

/**
 * Describes the message config_service.AddBadgeRequest.
 * Use `create(AddBadgeRequestSchema)` to create a new message.
 */
export const AddBadgeRequestSchema: GenMessage<AddBadgeRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 9);

/**
 * @generated from message config_service.AssignBadgeRequest
 */
export type AssignBadgeRequest = Message<"config_service.AssignBadgeRequest"> & {
  /**
   * @generated from field: string username = 1;
   */
  username: string;

  /**
   * @generated from field: string code = 2;
   */
  code: string;
};

/**
 * Describes the message config_service.AssignBadgeRequest.
 * Use `create(AssignBadgeRequestSchema)` to create a new message.
 */
export const AssignBadgeRequestSchema: GenMessage<AssignBadgeRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 10);

/**
 * @generated from message config_service.GetUsersForBadgeRequest
 */
export type GetUsersForBadgeRequest = Message<"config_service.GetUsersForBadgeRequest"> & {
  /**
   * @generated from field: string code = 1;
   */
  code: string;
};

/**
 * Describes the message config_service.GetUsersForBadgeRequest.
 * Use `create(GetUsersForBadgeRequestSchema)` to create a new message.
 */
export const GetUsersForBadgeRequestSchema: GenMessage<GetUsersForBadgeRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 11);

/**
 * @generated from message config_service.GetUserDetailsRequest
 */
export type GetUserDetailsRequest = Message<"config_service.GetUserDetailsRequest"> & {
  /**
   * @generated from field: string username = 1;
   */
  username: string;
};

/**
 * Describes the message config_service.GetUserDetailsRequest.
 * Use `create(GetUserDetailsRequestSchema)` to create a new message.
 */
export const GetUserDetailsRequestSchema: GenMessage<GetUserDetailsRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 12);

/**
 * @generated from message config_service.UserDetailsResponse
 */
export type UserDetailsResponse = Message<"config_service.UserDetailsResponse"> & {
  /**
   * @generated from field: string uuid = 1;
   */
  uuid: string;

  /**
   * @generated from field: string email = 2;
   */
  email: string;

  /**
   * @generated from field: google.protobuf.Timestamp created = 3;
   */
  created?: Timestamp;

  /**
   * @generated from field: string birth_date = 4;
   */
  birthDate: string;

  /**
   * @generated from field: string username = 5;
   */
  username: string;
};

/**
 * Describes the message config_service.UserDetailsResponse.
 * Use `create(UserDetailsResponseSchema)` to create a new message.
 */
export const UserDetailsResponseSchema: GenMessage<UserDetailsResponse> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 13);

/**
 * @generated from message config_service.SearchEmailRequest
 */
export type SearchEmailRequest = Message<"config_service.SearchEmailRequest"> & {
  /**
   * @generated from field: string partial_email = 1;
   */
  partialEmail: string;
};

/**
 * Describes the message config_service.SearchEmailRequest.
 * Use `create(SearchEmailRequestSchema)` to create a new message.
 */
export const SearchEmailRequestSchema: GenMessage<SearchEmailRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 14);

/**
 * @generated from message config_service.SearchEmailResponse
 */
export type SearchEmailResponse = Message<"config_service.SearchEmailResponse"> & {
  /**
   * @generated from field: repeated config_service.UserDetailsResponse users = 1;
   */
  users: UserDetailsResponse[];
};

/**
 * Describes the message config_service.SearchEmailResponse.
 * Use `create(SearchEmailResponseSchema)` to create a new message.
 */
export const SearchEmailResponseSchema: GenMessage<SearchEmailResponse> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 15);

/**
 * @generated from message config_service.Usernames
 */
export type Usernames = Message<"config_service.Usernames"> & {
  /**
   * @generated from field: repeated string usernames = 1;
   */
  usernames: string[];
};

/**
 * Describes the message config_service.Usernames.
 * Use `create(UsernamesSchema)` to create a new message.
 */
export const UsernamesSchema: GenMessage<Usernames> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 16);

/**
 * ConfigService requires admin authentication, except for the GetAnnouncements
 * endpoint.
 *
 * @generated from service config_service.ConfigService
 */
export const ConfigService: GenService<{
  /**
   * @generated from rpc config_service.ConfigService.SetGamesEnabled
   */
  setGamesEnabled: {
    methodKind: "unary";
    input: typeof EnableGamesRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.SetFEHash
   */
  setFEHash: {
    methodKind: "unary";
    input: typeof SetFEHashRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.SetAnnouncements
   */
  setAnnouncements: {
    methodKind: "unary";
    input: typeof SetAnnouncementsRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.GetAnnouncements
   */
  getAnnouncements: {
    methodKind: "unary";
    input: typeof GetAnnouncementsRequestSchema;
    output: typeof AnnouncementsResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.SetSingleAnnouncement
   */
  setSingleAnnouncement: {
    methodKind: "unary";
    input: typeof SetSingleAnnouncementRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.SetGlobalIntegration
   */
  setGlobalIntegration: {
    methodKind: "unary";
    input: typeof SetGlobalIntegrationRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.AddBadge
   */
  addBadge: {
    methodKind: "unary";
    input: typeof AddBadgeRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.AssignBadge
   */
  assignBadge: {
    methodKind: "unary";
    input: typeof AssignBadgeRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.UnassignBadge
   */
  unassignBadge: {
    methodKind: "unary";
    input: typeof AssignBadgeRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.GetUsersForBadge
   */
  getUsersForBadge: {
    methodKind: "unary";
    input: typeof GetUsersForBadgeRequestSchema;
    output: typeof UsernamesSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.GetUserDetails
   */
  getUserDetails: {
    methodKind: "unary";
    input: typeof GetUserDetailsRequestSchema;
    output: typeof UserDetailsResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.SearchEmail
   */
  searchEmail: {
    methodKind: "unary";
    input: typeof SearchEmailRequestSchema;
    output: typeof SearchEmailResponseSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_proto_config_service_config_service, 0);

