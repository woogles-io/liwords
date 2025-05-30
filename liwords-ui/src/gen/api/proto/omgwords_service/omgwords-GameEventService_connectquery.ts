// @generated by protoc-gen-connect-query v2.1.0 with parameter "target=ts"
// @generated from file proto/omgwords_service/omgwords.proto (package omgwords_service, syntax proto3)
/* eslint-disable */

import { GameEventService } from "./omgwords_pb";

/**
 * CreateBroadcastGame will create a game for Woogles broadcast
 *
 * @generated from rpc omgwords_service.GameEventService.CreateBroadcastGame
 */
export const createBroadcastGame = GameEventService.method.createBroadcastGame;

/**
 * DeleteBroadcastGame deletes a Woogles annotated game.
 *
 * @generated from rpc omgwords_service.GameEventService.DeleteBroadcastGame
 */
export const deleteBroadcastGame = GameEventService.method.deleteBroadcastGame;

/**
 * SendGameEvent is how one sends game events to the Woogles API.
 *
 * @generated from rpc omgwords_service.GameEventService.SendGameEvent
 */
export const sendGameEvent = GameEventService.method.sendGameEvent;

/**
 * SetRacks sets the rack for the players of the game.
 *
 * @generated from rpc omgwords_service.GameEventService.SetRacks
 */
export const setRacks = GameEventService.method.setRacks;

/**
 * @generated from rpc omgwords_service.GameEventService.ReplaceGameDocument
 */
export const replaceGameDocument = GameEventService.method.replaceGameDocument;

/**
 * PatchGameDocument merges in the passed-in GameDocument with what's on the
 * server. The passed-in GameDocument should be a partial document
 *
 * @generated from rpc omgwords_service.GameEventService.PatchGameDocument
 */
export const patchGameDocument = GameEventService.method.patchGameDocument;

/**
 * @generated from rpc omgwords_service.GameEventService.SetBroadcastGamePrivacy
 */
export const setBroadcastGamePrivacy = GameEventService.method.setBroadcastGamePrivacy;

/**
 * @generated from rpc omgwords_service.GameEventService.GetGamesForEditor
 */
export const getGamesForEditor = GameEventService.method.getGamesForEditor;

/**
 * @generated from rpc omgwords_service.GameEventService.GetMyUnfinishedGames
 */
export const getMyUnfinishedGames = GameEventService.method.getMyUnfinishedGames;

/**
 * GetGameDocument fetches the latest GameDocument for the passed-in ID.
 *
 * @generated from rpc omgwords_service.GameEventService.GetGameDocument
 */
export const getGameDocument = GameEventService.method.getGameDocument;

/**
 * @generated from rpc omgwords_service.GameEventService.GetRecentAnnotatedGames
 */
export const getRecentAnnotatedGames = GameEventService.method.getRecentAnnotatedGames;

/**
 * @generated from rpc omgwords_service.GameEventService.GetCGP
 */
export const getCGP = GameEventService.method.getCGP;

/**
 * @generated from rpc omgwords_service.GameEventService.ImportGCG
 */
export const importGCG = GameEventService.method.importGCG;
