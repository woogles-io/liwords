// @generated by protoc-gen-es v2.5.1 with parameter "target=ts"
// @generated from file proto/puzzle_service/puzzle_service.proto (package puzzle_service, syntax proto3)
/* eslint-disable */

import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import { enumDesc, fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv2";
import type { GameEvent, GameHistory, PuzzleGenerationRequest } from "../../vendor/macondo/macondo_pb";
import { file_vendor_macondo_macondo } from "../../vendor/macondo/macondo_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { file_google_protobuf_timestamp } from "@bufbuild/protobuf/wkt";
import type { ClientGameplayEvent } from "../ipc/omgwords_pb";
import { file_proto_ipc_omgwords } from "../ipc/omgwords_pb";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file proto/puzzle_service/puzzle_service.proto.
 */
export const file_proto_puzzle_service_puzzle_service: GenFile = /*@__PURE__*/
  fileDesc("Cilwcm90by9wdXp6bGVfc2VydmljZS9wdXp6bGVfc2VydmljZS5wcm90bxIOcHV6emxlX3NlcnZpY2UiJwoUU3RhcnRQdXp6bGVJZFJlcXVlc3QSDwoHbGV4aWNvbhgBIAEoCSJjChVTdGFydFB1enpsZUlkUmVzcG9uc2USEQoJcHV6emxlX2lkGAEgASgJEjcKDHF1ZXJ5X3Jlc3VsdBgCIAEoDjIhLnB1enpsZV9zZXJ2aWNlLlB1enpsZVF1ZXJ5UmVzdWx0IiYKE05leHRQdXp6bGVJZFJlcXVlc3QSDwoHbGV4aWNvbhgBIAEoCSJiChROZXh0UHV6emxlSWRSZXNwb25zZRIRCglwdXp6bGVfaWQYASABKAkSNwoMcXVlcnlfcmVzdWx0GAIgASgOMiEucHV6emxlX3NlcnZpY2UuUHV6emxlUXVlcnlSZXN1bHQiMwogTmV4dENsb3Nlc3RSYXRpbmdQdXp6bGVJZFJlcXVlc3QSDwoHbGV4aWNvbhgBIAEoCSJvCiFOZXh0Q2xvc2VzdFJhdGluZ1B1enpsZUlkUmVzcG9uc2USEQoJcHV6emxlX2lkGAEgASgJEjcKDHF1ZXJ5X3Jlc3VsdBgCIAEoDjIhLnB1enpsZV9zZXJ2aWNlLlB1enpsZVF1ZXJ5UmVzdWx0IiIKDVB1enpsZVJlcXVlc3QSEQoJcHV6emxlX2lkGAEgASgJItkCCg5BbnN3ZXJSZXNwb25zZRIqCg5jb3JyZWN0X2Fuc3dlchgBIAEoCzISLm1hY29uZG8uR2FtZUV2ZW50EiwKBnN0YXR1cxgCIAEoDjIcLnB1enpsZV9zZXJ2aWNlLlB1enpsZVN0YXR1cxIQCghhdHRlbXB0cxgDIAEoBRIPCgdnYW1lX2lkGAQgASgJEhMKC3R1cm5fbnVtYmVyGAUgASgFEhIKCmFmdGVyX3RleHQYBiABKAkSFwoPbmV3X3VzZXJfcmF0aW5nGAcgASgFEhkKEW5ld19wdXp6bGVfcmF0aW5nGAggASgFEjYKEmZpcnN0X2F0dGVtcHRfdGltZRgJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXASNQoRbGFzdF9hdHRlbXB0X3RpbWUYCiABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wInwKDlB1enpsZVJlc3BvbnNlEiUKB2hpc3RvcnkYASABKAsyFC5tYWNvbmRvLkdhbWVIaXN0b3J5EhMKC2JlZm9yZV90ZXh0GAIgASgJEi4KBmFuc3dlchgDIAEoCzIeLnB1enpsZV9zZXJ2aWNlLkFuc3dlclJlc3BvbnNlImcKEVN1Ym1pc3Npb25SZXF1ZXN0EhEKCXB1enpsZV9pZBgBIAEoCRIoCgZhbnN3ZXIYAiABKAsyGC5pcGMuQ2xpZW50R2FtZXBsYXlFdmVudBIVCg1zaG93X3NvbHV0aW9uGAMgASgIIl0KElN1Ym1pc3Npb25SZXNwb25zZRIXCg91c2VyX2lzX2NvcnJlY3QYASABKAgSLgoGYW5zd2VyGAIgASgLMh4ucHV6emxlX3NlcnZpY2UuQW5zd2VyUmVzcG9uc2UiKgoVUHJldmlvdXNQdXp6bGVSZXF1ZXN0EhEKCXB1enpsZV9pZBgBIAEoCSIrChZQcmV2aW91c1B1enpsZVJlc3BvbnNlEhEKCXB1enpsZV9pZBgBIAEoCSI0ChFQdXp6bGVWb3RlUmVxdWVzdBIRCglwdXp6bGVfaWQYASABKAkSDAoEdm90ZRgCIAEoBSIUChJQdXp6bGVWb3RlUmVzcG9uc2UizgIKGlB1enpsZUdlbmVyYXRpb25Kb2JSZXF1ZXN0EhIKCmJvdF92c19ib3QYASABKAgSDwoHbGV4aWNvbhgCIAEoCRIbChNsZXR0ZXJfZGlzdHJpYnV0aW9uGAMgASgJEhYKCnNxbF9vZmZzZXQYBCABKAVCAhgBEiAKGGdhbWVfY29uc2lkZXJhdGlvbl9saW1pdBgFIAEoBRIbChNnYW1lX2NyZWF0aW9uX2xpbWl0GAYgASgFEjEKB3JlcXVlc3QYByABKAsyIC5tYWNvbmRvLlB1enpsZUdlbmVyYXRpb25SZXF1ZXN0EhIKCnN0YXJ0X2RhdGUYCCABKAkSHwoXZXF1aXR5X2xvc3NfdG90YWxfbGltaXQYCSABKA0SFwoPYXZvaWRfYm90X2dhbWVzGAogASgIEhYKDmRheXNfcGVyX2NodW5rGAsgASgNIjEKHkFQSVB1enpsZUdlbmVyYXRpb25Kb2JSZXNwb25zZRIPCgdzdGFydGVkGAEgASgIInAKHUFQSVB1enpsZUdlbmVyYXRpb25Kb2JSZXF1ZXN0EjsKB3JlcXVlc3QYASABKAsyKi5wdXp6bGVfc2VydmljZS5QdXp6bGVHZW5lcmF0aW9uSm9iUmVxdWVzdBISCgpzZWNyZXRfa2V5GAIgASgJIjUKFFB1enpsZUpvYkxvZ3NSZXF1ZXN0Eg4KBm9mZnNldBgBIAEoBRINCgVsaW1pdBgCIAEoBSLiAQoMUHV6emxlSm9iTG9nEgoKAmlkGAEgASgDEjsKB3JlcXVlc3QYAiABKAsyKi5wdXp6bGVfc2VydmljZS5QdXp6bGVHZW5lcmF0aW9uSm9iUmVxdWVzdBIRCglmdWxmaWxsZWQYAyABKAgSFAoMZXJyb3Jfc3RhdHVzGAQgASgJEi4KCmNyZWF0ZWRfYXQYBSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wEjAKDGNvbXBsZXRlZF9hdBgGIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXAiQwoVUHV6emxlSm9iTG9nc1Jlc3BvbnNlEioKBGxvZ3MYASADKAsyHC5wdXp6bGVfc2VydmljZS5QdXp6bGVKb2JMb2cqYgoRUHV6emxlUXVlcnlSZXN1bHQSCgoGVU5TRUVOEAASCwoHVU5SQVRFRBABEg4KClVORklOSVNIRUQQAhINCglFWEhBVVNURUQQAxIKCgZSQU5ET00QBBIJCgVTVEFSVBAFKjoKDFB1enpsZVN0YXR1cxIOCgpVTkFOU1dFUkVEEAASCwoHQ09SUkVDVBABEg0KCUlOQ09SUkVDVBACMtwHCg1QdXp6bGVTZXJ2aWNlEl8KEEdldFN0YXJ0UHV6emxlSWQSJC5wdXp6bGVfc2VydmljZS5TdGFydFB1enpsZUlkUmVxdWVzdBolLnB1enpsZV9zZXJ2aWNlLlN0YXJ0UHV6emxlSWRSZXNwb25zZRJcCg9HZXROZXh0UHV6emxlSWQSIy5wdXp6bGVfc2VydmljZS5OZXh0UHV6emxlSWRSZXF1ZXN0GiQucHV6emxlX3NlcnZpY2UuTmV4dFB1enpsZUlkUmVzcG9uc2USgwEKHEdldE5leHRDbG9zZXN0UmF0aW5nUHV6emxlSWQSMC5wdXp6bGVfc2VydmljZS5OZXh0Q2xvc2VzdFJhdGluZ1B1enpsZUlkUmVxdWVzdBoxLnB1enpsZV9zZXJ2aWNlLk5leHRDbG9zZXN0UmF0aW5nUHV6emxlSWRSZXNwb25zZRJKCglHZXRQdXp6bGUSHS5wdXp6bGVfc2VydmljZS5QdXp6bGVSZXF1ZXN0Gh4ucHV6emxlX3NlcnZpY2UuUHV6emxlUmVzcG9uc2USVQoMU3VibWl0QW5zd2VyEiEucHV6emxlX3NlcnZpY2UuU3VibWlzc2lvblJlcXVlc3QaIi5wdXp6bGVfc2VydmljZS5TdWJtaXNzaW9uUmVzcG9uc2USUAoPR2V0UHV6emxlQW5zd2VyEh0ucHV6emxlX3NlcnZpY2UuUHV6emxlUmVxdWVzdBoeLnB1enpsZV9zZXJ2aWNlLkFuc3dlclJlc3BvbnNlEmQKE0dldFByZXZpb3VzUHV6emxlSWQSJS5wdXp6bGVfc2VydmljZS5QcmV2aW91c1B1enpsZVJlcXVlc3QaJi5wdXp6bGVfc2VydmljZS5QcmV2aW91c1B1enpsZVJlc3BvbnNlElYKDVNldFB1enpsZVZvdGUSIS5wdXp6bGVfc2VydmljZS5QdXp6bGVWb3RlUmVxdWVzdBoiLnB1enpsZV9zZXJ2aWNlLlB1enpsZVZvdGVSZXNwb25zZRJyChFTdGFydFB1enpsZUdlbkpvYhItLnB1enpsZV9zZXJ2aWNlLkFQSVB1enpsZUdlbmVyYXRpb25Kb2JSZXF1ZXN0Gi4ucHV6emxlX3NlcnZpY2UuQVBJUHV6emxlR2VuZXJhdGlvbkpvYlJlc3BvbnNlEl8KEEdldFB1enpsZUpvYkxvZ3MSJC5wdXp6bGVfc2VydmljZS5QdXp6bGVKb2JMb2dzUmVxdWVzdBolLnB1enpsZV9zZXJ2aWNlLlB1enpsZUpvYkxvZ3NSZXNwb25zZUK4AQoSY29tLnB1enpsZV9zZXJ2aWNlQhJQdXp6bGVTZXJ2aWNlUHJvdG9QAVo6Z2l0aHViLmNvbS93b29nbGVzLWlvL2xpd29yZHMvcnBjL2FwaS9wcm90by9wdXp6bGVfc2VydmljZaICA1BYWKoCDVB1enpsZVNlcnZpY2XKAg1QdXp6bGVTZXJ2aWNl4gIZUHV6emxlU2VydmljZVxHUEJNZXRhZGF0YeoCDVB1enpsZVNlcnZpY2ViBnByb3RvMw", [file_vendor_macondo_macondo, file_google_protobuf_timestamp, file_proto_ipc_omgwords]);

/**
 * @generated from message puzzle_service.StartPuzzleIdRequest
 */
export type StartPuzzleIdRequest = Message<"puzzle_service.StartPuzzleIdRequest"> & {
  /**
   * @generated from field: string lexicon = 1;
   */
  lexicon: string;
};

/**
 * Describes the message puzzle_service.StartPuzzleIdRequest.
 * Use `create(StartPuzzleIdRequestSchema)` to create a new message.
 */
export const StartPuzzleIdRequestSchema: GenMessage<StartPuzzleIdRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 0);

/**
 * @generated from message puzzle_service.StartPuzzleIdResponse
 */
export type StartPuzzleIdResponse = Message<"puzzle_service.StartPuzzleIdResponse"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;

  /**
   * @generated from field: puzzle_service.PuzzleQueryResult query_result = 2;
   */
  queryResult: PuzzleQueryResult;
};

/**
 * Describes the message puzzle_service.StartPuzzleIdResponse.
 * Use `create(StartPuzzleIdResponseSchema)` to create a new message.
 */
export const StartPuzzleIdResponseSchema: GenMessage<StartPuzzleIdResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 1);

/**
 * @generated from message puzzle_service.NextPuzzleIdRequest
 */
export type NextPuzzleIdRequest = Message<"puzzle_service.NextPuzzleIdRequest"> & {
  /**
   * @generated from field: string lexicon = 1;
   */
  lexicon: string;
};

/**
 * Describes the message puzzle_service.NextPuzzleIdRequest.
 * Use `create(NextPuzzleIdRequestSchema)` to create a new message.
 */
export const NextPuzzleIdRequestSchema: GenMessage<NextPuzzleIdRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 2);

/**
 * @generated from message puzzle_service.NextPuzzleIdResponse
 */
export type NextPuzzleIdResponse = Message<"puzzle_service.NextPuzzleIdResponse"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;

  /**
   * @generated from field: puzzle_service.PuzzleQueryResult query_result = 2;
   */
  queryResult: PuzzleQueryResult;
};

/**
 * Describes the message puzzle_service.NextPuzzleIdResponse.
 * Use `create(NextPuzzleIdResponseSchema)` to create a new message.
 */
export const NextPuzzleIdResponseSchema: GenMessage<NextPuzzleIdResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 3);

/**
 * @generated from message puzzle_service.NextClosestRatingPuzzleIdRequest
 */
export type NextClosestRatingPuzzleIdRequest = Message<"puzzle_service.NextClosestRatingPuzzleIdRequest"> & {
  /**
   * @generated from field: string lexicon = 1;
   */
  lexicon: string;
};

/**
 * Describes the message puzzle_service.NextClosestRatingPuzzleIdRequest.
 * Use `create(NextClosestRatingPuzzleIdRequestSchema)` to create a new message.
 */
export const NextClosestRatingPuzzleIdRequestSchema: GenMessage<NextClosestRatingPuzzleIdRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 4);

/**
 * @generated from message puzzle_service.NextClosestRatingPuzzleIdResponse
 */
export type NextClosestRatingPuzzleIdResponse = Message<"puzzle_service.NextClosestRatingPuzzleIdResponse"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;

  /**
   * @generated from field: puzzle_service.PuzzleQueryResult query_result = 2;
   */
  queryResult: PuzzleQueryResult;
};

/**
 * Describes the message puzzle_service.NextClosestRatingPuzzleIdResponse.
 * Use `create(NextClosestRatingPuzzleIdResponseSchema)` to create a new message.
 */
export const NextClosestRatingPuzzleIdResponseSchema: GenMessage<NextClosestRatingPuzzleIdResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 5);

/**
 * @generated from message puzzle_service.PuzzleRequest
 */
export type PuzzleRequest = Message<"puzzle_service.PuzzleRequest"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;
};

/**
 * Describes the message puzzle_service.PuzzleRequest.
 * Use `create(PuzzleRequestSchema)` to create a new message.
 */
export const PuzzleRequestSchema: GenMessage<PuzzleRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 6);

/**
 * @generated from message puzzle_service.AnswerResponse
 */
export type AnswerResponse = Message<"puzzle_service.AnswerResponse"> & {
  /**
   * @generated from field: macondo.GameEvent correct_answer = 1;
   */
  correctAnswer?: GameEvent;

  /**
   * @generated from field: puzzle_service.PuzzleStatus status = 2;
   */
  status: PuzzleStatus;

  /**
   * @generated from field: int32 attempts = 3;
   */
  attempts: number;

  /**
   * @generated from field: string game_id = 4;
   */
  gameId: string;

  /**
   * @generated from field: int32 turn_number = 5;
   */
  turnNumber: number;

  /**
   * @generated from field: string after_text = 6;
   */
  afterText: string;

  /**
   * @generated from field: int32 new_user_rating = 7;
   */
  newUserRating: number;

  /**
   * @generated from field: int32 new_puzzle_rating = 8;
   */
  newPuzzleRating: number;

  /**
   * @generated from field: google.protobuf.Timestamp first_attempt_time = 9;
   */
  firstAttemptTime?: Timestamp;

  /**
   * @generated from field: google.protobuf.Timestamp last_attempt_time = 10;
   */
  lastAttemptTime?: Timestamp;
};

/**
 * Describes the message puzzle_service.AnswerResponse.
 * Use `create(AnswerResponseSchema)` to create a new message.
 */
export const AnswerResponseSchema: GenMessage<AnswerResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 7);

/**
 * @generated from message puzzle_service.PuzzleResponse
 */
export type PuzzleResponse = Message<"puzzle_service.PuzzleResponse"> & {
  /**
   * @generated from field: macondo.GameHistory history = 1;
   */
  history?: GameHistory;

  /**
   * @generated from field: string before_text = 2;
   */
  beforeText: string;

  /**
   * @generated from field: puzzle_service.AnswerResponse answer = 3;
   */
  answer?: AnswerResponse;
};

/**
 * Describes the message puzzle_service.PuzzleResponse.
 * Use `create(PuzzleResponseSchema)` to create a new message.
 */
export const PuzzleResponseSchema: GenMessage<PuzzleResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 8);

/**
 * @generated from message puzzle_service.SubmissionRequest
 */
export type SubmissionRequest = Message<"puzzle_service.SubmissionRequest"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;

  /**
   * @generated from field: ipc.ClientGameplayEvent answer = 2;
   */
  answer?: ClientGameplayEvent;

  /**
   * @generated from field: bool show_solution = 3;
   */
  showSolution: boolean;
};

/**
 * Describes the message puzzle_service.SubmissionRequest.
 * Use `create(SubmissionRequestSchema)` to create a new message.
 */
export const SubmissionRequestSchema: GenMessage<SubmissionRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 9);

/**
 * @generated from message puzzle_service.SubmissionResponse
 */
export type SubmissionResponse = Message<"puzzle_service.SubmissionResponse"> & {
  /**
   * @generated from field: bool user_is_correct = 1;
   */
  userIsCorrect: boolean;

  /**
   * @generated from field: puzzle_service.AnswerResponse answer = 2;
   */
  answer?: AnswerResponse;
};

/**
 * Describes the message puzzle_service.SubmissionResponse.
 * Use `create(SubmissionResponseSchema)` to create a new message.
 */
export const SubmissionResponseSchema: GenMessage<SubmissionResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 10);

/**
 * @generated from message puzzle_service.PreviousPuzzleRequest
 */
export type PreviousPuzzleRequest = Message<"puzzle_service.PreviousPuzzleRequest"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;
};

/**
 * Describes the message puzzle_service.PreviousPuzzleRequest.
 * Use `create(PreviousPuzzleRequestSchema)` to create a new message.
 */
export const PreviousPuzzleRequestSchema: GenMessage<PreviousPuzzleRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 11);

/**
 * @generated from message puzzle_service.PreviousPuzzleResponse
 */
export type PreviousPuzzleResponse = Message<"puzzle_service.PreviousPuzzleResponse"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;
};

/**
 * Describes the message puzzle_service.PreviousPuzzleResponse.
 * Use `create(PreviousPuzzleResponseSchema)` to create a new message.
 */
export const PreviousPuzzleResponseSchema: GenMessage<PreviousPuzzleResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 12);

/**
 * @generated from message puzzle_service.PuzzleVoteRequest
 */
export type PuzzleVoteRequest = Message<"puzzle_service.PuzzleVoteRequest"> & {
  /**
   * @generated from field: string puzzle_id = 1;
   */
  puzzleId: string;

  /**
   * @generated from field: int32 vote = 2;
   */
  vote: number;
};

/**
 * Describes the message puzzle_service.PuzzleVoteRequest.
 * Use `create(PuzzleVoteRequestSchema)` to create a new message.
 */
export const PuzzleVoteRequestSchema: GenMessage<PuzzleVoteRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 13);

/**
 * @generated from message puzzle_service.PuzzleVoteResponse
 */
export type PuzzleVoteResponse = Message<"puzzle_service.PuzzleVoteResponse"> & {
};

/**
 * Describes the message puzzle_service.PuzzleVoteResponse.
 * Use `create(PuzzleVoteResponseSchema)` to create a new message.
 */
export const PuzzleVoteResponseSchema: GenMessage<PuzzleVoteResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 14);

/**
 * @generated from message puzzle_service.PuzzleGenerationJobRequest
 */
export type PuzzleGenerationJobRequest = Message<"puzzle_service.PuzzleGenerationJobRequest"> & {
  /**
   * @generated from field: bool bot_vs_bot = 1;
   */
  botVsBot: boolean;

  /**
   * @generated from field: string lexicon = 2;
   */
  lexicon: string;

  /**
   * @generated from field: string letter_distribution = 3;
   */
  letterDistribution: string;

  /**
   * @generated from field: int32 sql_offset = 4 [deprecated = true];
   * @deprecated
   */
  sqlOffset: number;

  /**
   * @generated from field: int32 game_consideration_limit = 5;
   */
  gameConsiderationLimit: number;

  /**
   * @generated from field: int32 game_creation_limit = 6;
   */
  gameCreationLimit: number;

  /**
   * @generated from field: macondo.PuzzleGenerationRequest request = 7;
   */
  request?: PuzzleGenerationRequest;

  /**
   * start_date is just a YYYY-MM-DD date at which we should
   * start looking for games (in non bot_v_bot), and go backwards
   * from there.
   *
   * @generated from field: string start_date = 8;
   */
  startDate: string;

  /**
   * @generated from field: uint32 equity_loss_total_limit = 9;
   */
  equityLossTotalLimit: number;

  /**
   * @generated from field: bool avoid_bot_games = 10;
   */
  avoidBotGames: boolean;

  /**
   * @generated from field: uint32 days_per_chunk = 11;
   */
  daysPerChunk: number;
};

/**
 * Describes the message puzzle_service.PuzzleGenerationJobRequest.
 * Use `create(PuzzleGenerationJobRequestSchema)` to create a new message.
 */
export const PuzzleGenerationJobRequestSchema: GenMessage<PuzzleGenerationJobRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 15);

/**
 * @generated from message puzzle_service.APIPuzzleGenerationJobResponse
 */
export type APIPuzzleGenerationJobResponse = Message<"puzzle_service.APIPuzzleGenerationJobResponse"> & {
  /**
   * @generated from field: bool started = 1;
   */
  started: boolean;
};

/**
 * Describes the message puzzle_service.APIPuzzleGenerationJobResponse.
 * Use `create(APIPuzzleGenerationJobResponseSchema)` to create a new message.
 */
export const APIPuzzleGenerationJobResponseSchema: GenMessage<APIPuzzleGenerationJobResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 16);

/**
 * @generated from message puzzle_service.APIPuzzleGenerationJobRequest
 */
export type APIPuzzleGenerationJobRequest = Message<"puzzle_service.APIPuzzleGenerationJobRequest"> & {
  /**
   * @generated from field: puzzle_service.PuzzleGenerationJobRequest request = 1;
   */
  request?: PuzzleGenerationJobRequest;

  /**
   * @generated from field: string secret_key = 2;
   */
  secretKey: string;
};

/**
 * Describes the message puzzle_service.APIPuzzleGenerationJobRequest.
 * Use `create(APIPuzzleGenerationJobRequestSchema)` to create a new message.
 */
export const APIPuzzleGenerationJobRequestSchema: GenMessage<APIPuzzleGenerationJobRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 17);

/**
 * @generated from message puzzle_service.PuzzleJobLogsRequest
 */
export type PuzzleJobLogsRequest = Message<"puzzle_service.PuzzleJobLogsRequest"> & {
  /**
   * @generated from field: int32 offset = 1;
   */
  offset: number;

  /**
   * @generated from field: int32 limit = 2;
   */
  limit: number;
};

/**
 * Describes the message puzzle_service.PuzzleJobLogsRequest.
 * Use `create(PuzzleJobLogsRequestSchema)` to create a new message.
 */
export const PuzzleJobLogsRequestSchema: GenMessage<PuzzleJobLogsRequest> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 18);

/**
 * @generated from message puzzle_service.PuzzleJobLog
 */
export type PuzzleJobLog = Message<"puzzle_service.PuzzleJobLog"> & {
  /**
   * @generated from field: int64 id = 1;
   */
  id: bigint;

  /**
   * @generated from field: puzzle_service.PuzzleGenerationJobRequest request = 2;
   */
  request?: PuzzleGenerationJobRequest;

  /**
   * @generated from field: bool fulfilled = 3;
   */
  fulfilled: boolean;

  /**
   * @generated from field: string error_status = 4;
   */
  errorStatus: string;

  /**
   * @generated from field: google.protobuf.Timestamp created_at = 5;
   */
  createdAt?: Timestamp;

  /**
   * @generated from field: google.protobuf.Timestamp completed_at = 6;
   */
  completedAt?: Timestamp;
};

/**
 * Describes the message puzzle_service.PuzzleJobLog.
 * Use `create(PuzzleJobLogSchema)` to create a new message.
 */
export const PuzzleJobLogSchema: GenMessage<PuzzleJobLog> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 19);

/**
 * @generated from message puzzle_service.PuzzleJobLogsResponse
 */
export type PuzzleJobLogsResponse = Message<"puzzle_service.PuzzleJobLogsResponse"> & {
  /**
   * @generated from field: repeated puzzle_service.PuzzleJobLog logs = 1;
   */
  logs: PuzzleJobLog[];
};

/**
 * Describes the message puzzle_service.PuzzleJobLogsResponse.
 * Use `create(PuzzleJobLogsResponseSchema)` to create a new message.
 */
export const PuzzleJobLogsResponseSchema: GenMessage<PuzzleJobLogsResponse> = /*@__PURE__*/
  messageDesc(file_proto_puzzle_service_puzzle_service, 20);

/**
 * @generated from enum puzzle_service.PuzzleQueryResult
 */
export enum PuzzleQueryResult {
  /**
   * @generated from enum value: UNSEEN = 0;
   */
  UNSEEN = 0,

  /**
   * @generated from enum value: UNRATED = 1;
   */
  UNRATED = 1,

  /**
   * @generated from enum value: UNFINISHED = 2;
   */
  UNFINISHED = 2,

  /**
   * @generated from enum value: EXHAUSTED = 3;
   */
  EXHAUSTED = 3,

  /**
   * @generated from enum value: RANDOM = 4;
   */
  RANDOM = 4,

  /**
   * @generated from enum value: START = 5;
   */
  START = 5,
}

/**
 * Describes the enum puzzle_service.PuzzleQueryResult.
 */
export const PuzzleQueryResultSchema: GenEnum<PuzzleQueryResult> = /*@__PURE__*/
  enumDesc(file_proto_puzzle_service_puzzle_service, 0);

/**
 * @generated from enum puzzle_service.PuzzleStatus
 */
export enum PuzzleStatus {
  /**
   * @generated from enum value: UNANSWERED = 0;
   */
  UNANSWERED = 0,

  /**
   * @generated from enum value: CORRECT = 1;
   */
  CORRECT = 1,

  /**
   * @generated from enum value: INCORRECT = 2;
   */
  INCORRECT = 2,
}

/**
 * Describes the enum puzzle_service.PuzzleStatus.
 */
export const PuzzleStatusSchema: GenEnum<PuzzleStatus> = /*@__PURE__*/
  enumDesc(file_proto_puzzle_service_puzzle_service, 1);

/**
 * @generated from service puzzle_service.PuzzleService
 */
export const PuzzleService: GenService<{
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetStartPuzzleId
   */
  getStartPuzzleId: {
    methodKind: "unary";
    input: typeof StartPuzzleIdRequestSchema;
    output: typeof StartPuzzleIdResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetNextPuzzleId
   */
  getNextPuzzleId: {
    methodKind: "unary";
    input: typeof NextPuzzleIdRequestSchema;
    output: typeof NextPuzzleIdResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetNextClosestRatingPuzzleId
   */
  getNextClosestRatingPuzzleId: {
    methodKind: "unary";
    input: typeof NextClosestRatingPuzzleIdRequestSchema;
    output: typeof NextClosestRatingPuzzleIdResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetPuzzle
   */
  getPuzzle: {
    methodKind: "unary";
    input: typeof PuzzleRequestSchema;
    output: typeof PuzzleResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.SubmitAnswer
   */
  submitAnswer: {
    methodKind: "unary";
    input: typeof SubmissionRequestSchema;
    output: typeof SubmissionResponseSchema;
  },
  /**
   * GetPuzzleAnswer just gets the answer of the puzzle without a submission.
   * It will not work if the user has not tried the puzzle at least once.
   *
   * @generated from rpc puzzle_service.PuzzleService.GetPuzzleAnswer
   */
  getPuzzleAnswer: {
    methodKind: "unary";
    input: typeof PuzzleRequestSchema;
    output: typeof AnswerResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetPreviousPuzzleId
   */
  getPreviousPuzzleId: {
    methodKind: "unary";
    input: typeof PreviousPuzzleRequestSchema;
    output: typeof PreviousPuzzleResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.SetPuzzleVote
   */
  setPuzzleVote: {
    methodKind: "unary";
    input: typeof PuzzleVoteRequestSchema;
    output: typeof PuzzleVoteResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.StartPuzzleGenJob
   */
  startPuzzleGenJob: {
    methodKind: "unary";
    input: typeof APIPuzzleGenerationJobRequestSchema;
    output: typeof APIPuzzleGenerationJobResponseSchema;
  },
  /**
   * @generated from rpc puzzle_service.PuzzleService.GetPuzzleJobLogs
   */
  getPuzzleJobLogs: {
    methodKind: "unary";
    input: typeof PuzzleJobLogsRequestSchema;
    output: typeof PuzzleJobLogsResponseSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_proto_puzzle_service_puzzle_service, 0);

