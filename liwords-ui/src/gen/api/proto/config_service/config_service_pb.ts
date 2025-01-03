// @generated by protoc-gen-es v2.2.0 with parameter "target=ts"
// @generated from file proto/config_service/config_service.proto (package config_service, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv1";
import { fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv1";
import { file_google_protobuf_wrappers } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file proto/config_service/config_service.proto.
 */
export const file_proto_config_service_config_service: GenFile = /*@__PURE__*/
  fileDesc("Cilwcm90by9jb25maWdfc2VydmljZS9jb25maWdfc2VydmljZS5wcm90bxIOY29uZmlnX3NlcnZpY2UiJQoSRW5hYmxlR2FtZXNSZXF1ZXN0Eg8KB2VuYWJsZWQYASABKAgiIAoQU2V0RkVIYXNoUmVxdWVzdBIMCgRoYXNoGAEgASgJItEBChJQZXJtaXNzaW9uc1JlcXVlc3QSEAoIdXNlcm5hbWUYASABKAkSLAoIZGlyZWN0b3IYAiABKAsyGi5nb29nbGUucHJvdG9idWYuQm9vbFZhbHVlEikKBWFkbWluGAMgASgLMhouZ29vZ2xlLnByb3RvYnVmLkJvb2xWYWx1ZRInCgNtb2QYBCABKAsyGi5nb29nbGUucHJvdG9idWYuQm9vbFZhbHVlEicKA2JvdBgFIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5Cb29sVmFsdWUiHwoLVXNlclJlcXVlc3QSEAoIdXNlcm5hbWUYASABKAkihAEKDFVzZXJSZXNwb25zZRIQCgh1c2VybmFtZRgBIAEoCRIMCgR1dWlkGAIgASgJEg0KBWVtYWlsGAMgASgJEg4KBmlzX2JvdBgEIAEoCBITCgtpc19kaXJlY3RvchgFIAEoCBIOCgZpc19tb2QYBiABKAgSEAoIaXNfYWRtaW4YByABKAgiEAoOQ29uZmlnUmVzcG9uc2UiOQoMQW5ub3VuY2VtZW50Eg0KBXRpdGxlGAEgASgJEgwKBGxpbmsYAiABKAkSDAoEYm9keRgDIAEoCSJOChdTZXRBbm5vdW5jZW1lbnRzUmVxdWVzdBIzCg1hbm5vdW5jZW1lbnRzGAEgAygLMhwuY29uZmlnX3NlcnZpY2UuQW5ub3VuY2VtZW50IhkKF0dldEFubm91bmNlbWVudHNSZXF1ZXN0IkwKFUFubm91bmNlbWVudHNSZXNwb25zZRIzCg1hbm5vdW5jZW1lbnRzGAEgAygLMhwuY29uZmlnX3NlcnZpY2UuQW5ub3VuY2VtZW50Im4KHFNldFNpbmdsZUFubm91bmNlbWVudFJlcXVlc3QSMgoMYW5ub3VuY2VtZW50GAEgASgLMhwuY29uZmlnX3NlcnZpY2UuQW5ub3VuY2VtZW50EhoKEmxpbmtfc2VhcmNoX3N0cmluZxgCIAEoCSJKChtTZXRHbG9iYWxJbnRlZ3JhdGlvblJlcXVlc3QSGAoQaW50ZWdyYXRpb25fbmFtZRgBIAEoCRIRCglqc29uX2RhdGEYAiABKAky6QUKDUNvbmZpZ1NlcnZpY2USVQoPU2V0R2FtZXNFbmFibGVkEiIuY29uZmlnX3NlcnZpY2UuRW5hYmxlR2FtZXNSZXF1ZXN0Gh4uY29uZmlnX3NlcnZpY2UuQ29uZmlnUmVzcG9uc2USTQoJU2V0RkVIYXNoEiAuY29uZmlnX3NlcnZpY2UuU2V0RkVIYXNoUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlElgKElNldFVzZXJQZXJtaXNzaW9ucxIiLmNvbmZpZ19zZXJ2aWNlLlBlcm1pc3Npb25zUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlEksKDkdldFVzZXJEZXRhaWxzEhsuY29uZmlnX3NlcnZpY2UuVXNlclJlcXVlc3QaHC5jb25maWdfc2VydmljZS5Vc2VyUmVzcG9uc2USWwoQU2V0QW5ub3VuY2VtZW50cxInLmNvbmZpZ19zZXJ2aWNlLlNldEFubm91bmNlbWVudHNSZXF1ZXN0Gh4uY29uZmlnX3NlcnZpY2UuQ29uZmlnUmVzcG9uc2USYgoQR2V0QW5ub3VuY2VtZW50cxInLmNvbmZpZ19zZXJ2aWNlLkdldEFubm91bmNlbWVudHNSZXF1ZXN0GiUuY29uZmlnX3NlcnZpY2UuQW5ub3VuY2VtZW50c1Jlc3BvbnNlEmUKFVNldFNpbmdsZUFubm91bmNlbWVudBIsLmNvbmZpZ19zZXJ2aWNlLlNldFNpbmdsZUFubm91bmNlbWVudFJlcXVlc3QaHi5jb25maWdfc2VydmljZS5Db25maWdSZXNwb25zZRJjChRTZXRHbG9iYWxJbnRlZ3JhdGlvbhIrLmNvbmZpZ19zZXJ2aWNlLlNldEdsb2JhbEludGVncmF0aW9uUmVxdWVzdBoeLmNvbmZpZ19zZXJ2aWNlLkNvbmZpZ1Jlc3BvbnNlQrgBChJjb20uY29uZmlnX3NlcnZpY2VCEkNvbmZpZ1NlcnZpY2VQcm90b1ABWjpnaXRodWIuY29tL3dvb2dsZXMtaW8vbGl3b3Jkcy9ycGMvYXBpL3Byb3RvL2NvbmZpZ19zZXJ2aWNlogIDQ1hYqgINQ29uZmlnU2VydmljZcoCDUNvbmZpZ1NlcnZpY2XiAhlDb25maWdTZXJ2aWNlXEdQQk1ldGFkYXRh6gINQ29uZmlnU2VydmljZWIGcHJvdG8z", [file_google_protobuf_wrappers]);

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
 * @generated from message config_service.PermissionsRequest
 */
export type PermissionsRequest = Message<"config_service.PermissionsRequest"> & {
  /**
   * @generated from field: string username = 1;
   */
  username: string;

  /**
   * @generated from field: google.protobuf.BoolValue director = 2;
   */
  director?: boolean;

  /**
   * @generated from field: google.protobuf.BoolValue admin = 3;
   */
  admin?: boolean;

  /**
   * @generated from field: google.protobuf.BoolValue mod = 4;
   */
  mod?: boolean;

  /**
   * @generated from field: google.protobuf.BoolValue bot = 5;
   */
  bot?: boolean;
};

/**
 * Describes the message config_service.PermissionsRequest.
 * Use `create(PermissionsRequestSchema)` to create a new message.
 */
export const PermissionsRequestSchema: GenMessage<PermissionsRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 2);

/**
 * @generated from message config_service.UserRequest
 */
export type UserRequest = Message<"config_service.UserRequest"> & {
  /**
   * @generated from field: string username = 1;
   */
  username: string;
};

/**
 * Describes the message config_service.UserRequest.
 * Use `create(UserRequestSchema)` to create a new message.
 */
export const UserRequestSchema: GenMessage<UserRequest> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 3);

/**
 * @generated from message config_service.UserResponse
 */
export type UserResponse = Message<"config_service.UserResponse"> & {
  /**
   * @generated from field: string username = 1;
   */
  username: string;

  /**
   * @generated from field: string uuid = 2;
   */
  uuid: string;

  /**
   * @generated from field: string email = 3;
   */
  email: string;

  /**
   * @generated from field: bool is_bot = 4;
   */
  isBot: boolean;

  /**
   * @generated from field: bool is_director = 5;
   */
  isDirector: boolean;

  /**
   * @generated from field: bool is_mod = 6;
   */
  isMod: boolean;

  /**
   * @generated from field: bool is_admin = 7;
   */
  isAdmin: boolean;
};

/**
 * Describes the message config_service.UserResponse.
 * Use `create(UserResponseSchema)` to create a new message.
 */
export const UserResponseSchema: GenMessage<UserResponse> = /*@__PURE__*/
  messageDesc(file_proto_config_service_config_service, 4);

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
  messageDesc(file_proto_config_service_config_service, 5);

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
  messageDesc(file_proto_config_service_config_service, 6);

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
  messageDesc(file_proto_config_service_config_service, 7);

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
  messageDesc(file_proto_config_service_config_service, 8);

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
  messageDesc(file_proto_config_service_config_service, 9);

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
  messageDesc(file_proto_config_service_config_service, 10);

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
  messageDesc(file_proto_config_service_config_service, 11);

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
   * @generated from rpc config_service.ConfigService.SetUserPermissions
   */
  setUserPermissions: {
    methodKind: "unary";
    input: typeof PermissionsRequestSchema;
    output: typeof ConfigResponseSchema;
  },
  /**
   * @generated from rpc config_service.ConfigService.GetUserDetails
   */
  getUserDetails: {
    methodKind: "unary";
    input: typeof UserRequestSchema;
    output: typeof UserResponseSchema;
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
}> = /*@__PURE__*/
  serviceDesc(file_proto_config_service_config_service, 0);

