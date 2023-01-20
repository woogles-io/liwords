// @generated by protoc-gen-connect-web v0.3.1
// @generated from file puzzle_service/puzzle_service.proto (package puzzle_service, syntax proto3)
/* eslint-disable */
/* @ts-nocheck */

import {AnswerResponse, APIPuzzleGenerationJobRequest, APIPuzzleGenerationJobResponse, NextClosestRatingPuzzleIdRequest, NextClosestRatingPuzzleIdResponse, NextPuzzleIdRequest, NextPuzzleIdResponse, PreviousPuzzleRequest, PreviousPuzzleResponse, PuzzleJobLogsRequest, PuzzleJobLogsResponse, PuzzleRequest, PuzzleResponse, PuzzleVoteRequest, PuzzleVoteResponse, StartPuzzleIdRequest, StartPuzzleIdResponse, SubmissionRequest, SubmissionResponse} from "./puzzle_service_pb.js";
import {MethodKind} from "@bufbuild/protobuf";

/**
 * @generated from service puzzle_service.PuzzleService
 */
export declare const PuzzleService: {
  readonly typeName: "puzzle_service.PuzzleService",
  readonly methods: {
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetStartPuzzleId
     */
    readonly getStartPuzzleId: {
      readonly name: "GetStartPuzzleId",
      readonly I: typeof StartPuzzleIdRequest,
      readonly O: typeof StartPuzzleIdResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetNextPuzzleId
     */
    readonly getNextPuzzleId: {
      readonly name: "GetNextPuzzleId",
      readonly I: typeof NextPuzzleIdRequest,
      readonly O: typeof NextPuzzleIdResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetNextClosestRatingPuzzleId
     */
    readonly getNextClosestRatingPuzzleId: {
      readonly name: "GetNextClosestRatingPuzzleId",
      readonly I: typeof NextClosestRatingPuzzleIdRequest,
      readonly O: typeof NextClosestRatingPuzzleIdResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetPuzzle
     */
    readonly getPuzzle: {
      readonly name: "GetPuzzle",
      readonly I: typeof PuzzleRequest,
      readonly O: typeof PuzzleResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.SubmitAnswer
     */
    readonly submitAnswer: {
      readonly name: "SubmitAnswer",
      readonly I: typeof SubmissionRequest,
      readonly O: typeof SubmissionResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * GetPuzzleAnswer just gets the answer of the puzzle without a submission.
     * It will not work if the user has not tried the puzzle at least once.
     *
     * @generated from rpc puzzle_service.PuzzleService.GetPuzzleAnswer
     */
    readonly getPuzzleAnswer: {
      readonly name: "GetPuzzleAnswer",
      readonly I: typeof PuzzleRequest,
      readonly O: typeof AnswerResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetPreviousPuzzleId
     */
    readonly getPreviousPuzzleId: {
      readonly name: "GetPreviousPuzzleId",
      readonly I: typeof PreviousPuzzleRequest,
      readonly O: typeof PreviousPuzzleResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.SetPuzzleVote
     */
    readonly setPuzzleVote: {
      readonly name: "SetPuzzleVote",
      readonly I: typeof PuzzleVoteRequest,
      readonly O: typeof PuzzleVoteResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.StartPuzzleGenJob
     */
    readonly startPuzzleGenJob: {
      readonly name: "StartPuzzleGenJob",
      readonly I: typeof APIPuzzleGenerationJobRequest,
      readonly O: typeof APIPuzzleGenerationJobResponse,
      readonly kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc puzzle_service.PuzzleService.GetPuzzleJobLogs
     */
    readonly getPuzzleJobLogs: {
      readonly name: "GetPuzzleJobLogs",
      readonly I: typeof PuzzleJobLogsRequest,
      readonly O: typeof PuzzleJobLogsResponse,
      readonly kind: MethodKind.Unary,
    },
  }
};
