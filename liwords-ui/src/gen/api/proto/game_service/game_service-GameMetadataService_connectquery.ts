// @generated by protoc-gen-connect-query v2.1.0 with parameter "target=ts"
// @generated from file proto/game_service/game_service.proto (package game_service, syntax proto3)
/* eslint-disable */

import { GameMetadataService } from "./game_service_pb";

/**
 * @generated from rpc game_service.GameMetadataService.GetMetadata
 */
export const getMetadata = GameMetadataService.method.getMetadata;

/**
 * GetGCG gets a GCG string for the given game ID.
 *
 * @generated from rpc game_service.GameMetadataService.GetGCG
 */
export const getGCG = GameMetadataService.method.getGCG;

/**
 * GetGameHistory gets a GameHistory for the given game ID. GameHistory
 * is our internal representation of a game's state.
 *
 * @generated from rpc game_service.GameMetadataService.GetGameHistory
 */
export const getGameHistory = GameMetadataService.method.getGameHistory;

/**
 * GetRecentGames gets recent games for a user.
 *
 * @generated from rpc game_service.GameMetadataService.GetRecentGames
 */
export const getRecentGames = GameMetadataService.method.getRecentGames;

/**
 * @generated from rpc game_service.GameMetadataService.GetRematchStreak
 */
export const getRematchStreak = GameMetadataService.method.getRematchStreak;

/**
 * GetGameDocument gets a Game Document. This will eventually obsolete
 * GetGameHistory. Does not work with annotated games for now.
 *
 * @generated from rpc game_service.GameMetadataService.GetGameDocument
 */
export const getGameDocument = GameMetadataService.method.getGameDocument;
