// Definitions for user-related matters

// @generated by protoc-gen-es v2.2.0 with parameter "target=ts"
// @generated from file proto/ipc/users.proto (package ipc, syntax proto3)
/* eslint-disable */

import type { GenEnum, GenFile, GenMessage } from "@bufbuild/protobuf/codegenv1";
import { enumDesc, fileDesc, messageDesc } from "@bufbuild/protobuf/codegenv1";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file proto/ipc/users.proto.
 */
export const file_proto_ipc_users: GenFile = /*@__PURE__*/
  fileDesc("ChVwcm90by9pcGMvdXNlcnMucHJvdG8SA2lwYyLKAQoNUHJvZmlsZVVwZGF0ZRIPCgd1c2VyX2lkGAEgASgJEjAKB3JhdGluZ3MYAiADKAsyHy5pcGMuUHJvZmlsZVVwZGF0ZS5SYXRpbmdzRW50cnkaKwoGUmF0aW5nEg4KBnJhdGluZxgBIAEoARIRCglkZXZpYXRpb24YAiABKAEaSQoMUmF0aW5nc0VudHJ5EgsKA2tleRgBIAEoCRIoCgV2YWx1ZRgCIAEoCzIZLmlwYy5Qcm9maWxlVXBkYXRlLlJhdGluZzoCOAEqNAoLQ2hpbGRTdGF0dXMSCQoFQ0hJTEQQABINCglOT1RfQ0hJTEQQARILCgdVTktOT1dOEAJCcgoHY29tLmlwY0IKVXNlcnNQcm90b1ABWi9naXRodWIuY29tL3dvb2dsZXMtaW8vbGl3b3Jkcy9ycGMvYXBpL3Byb3RvL2lwY6ICA0lYWKoCA0lwY8oCA0lwY+ICD0lwY1xHUEJNZXRhZGF0YeoCA0lwY2IGcHJvdG8z");

/**
 * @generated from message ipc.ProfileUpdate
 */
export type ProfileUpdate = Message<"ipc.ProfileUpdate"> & {
  /**
   * @generated from field: string user_id = 1;
   */
  userId: string;

  /**
   * map of variant name to rating
   *
   * @generated from field: map<string, ipc.ProfileUpdate.Rating> ratings = 2;
   */
  ratings: { [key: string]: ProfileUpdate_Rating };
};

/**
 * Describes the message ipc.ProfileUpdate.
 * Use `create(ProfileUpdateSchema)` to create a new message.
 */
export const ProfileUpdateSchema: GenMessage<ProfileUpdate> = /*@__PURE__*/
  messageDesc(file_proto_ipc_users, 0);

/**
 * @generated from message ipc.ProfileUpdate.Rating
 */
export type ProfileUpdate_Rating = Message<"ipc.ProfileUpdate.Rating"> & {
  /**
   * @generated from field: double rating = 1;
   */
  rating: number;

  /**
   * @generated from field: double deviation = 2;
   */
  deviation: number;
};

/**
 * Describes the message ipc.ProfileUpdate.Rating.
 * Use `create(ProfileUpdate_RatingSchema)` to create a new message.
 */
export const ProfileUpdate_RatingSchema: GenMessage<ProfileUpdate_Rating> = /*@__PURE__*/
  messageDesc(file_proto_ipc_users, 0, 0);

/**
 * @generated from enum ipc.ChildStatus
 */
export enum ChildStatus {
  /**
   * @generated from enum value: CHILD = 0;
   */
  CHILD = 0,

  /**
   * @generated from enum value: NOT_CHILD = 1;
   */
  NOT_CHILD = 1,

  /**
   * @generated from enum value: UNKNOWN = 2;
   */
  UNKNOWN = 2,
}

/**
 * Describes the enum ipc.ChildStatus.
 */
export const ChildStatusSchema: GenEnum<ChildStatus> = /*@__PURE__*/
  enumDesc(file_proto_ipc_users, 0);
