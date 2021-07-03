// source: api/proto/realtime/realtime.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = Function('return this')();

var macondo_api_proto_macondo_macondo_pb = require('../../../macondo/api/proto/macondo/macondo_pb.js');
goog.object.extend(proto, macondo_api_proto_macondo_macondo_pb);
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
goog.object.extend(proto, google_protobuf_timestamp_pb);
goog.exportSymbol('proto.liwords.ActiveGameEntry', null, global);
goog.exportSymbol('proto.liwords.ActiveGamePlayer', null, global);
goog.exportSymbol('proto.liwords.ChatMessage', null, global);
goog.exportSymbol('proto.liwords.ChatMessageDeleted', null, global);
goog.exportSymbol('proto.liwords.ChatMessages', null, global);
goog.exportSymbol('proto.liwords.ChildStatus', null, global);
goog.exportSymbol('proto.liwords.ClientGameplayEvent', null, global);
goog.exportSymbol('proto.liwords.ClientGameplayEvent.EventType', null, global);
goog.exportSymbol('proto.liwords.DeclineMatchRequest', null, global);
goog.exportSymbol('proto.liwords.DivisionControls', null, global);
goog.exportSymbol('proto.liwords.DivisionControlsResponse', null, global);
goog.exportSymbol('proto.liwords.DivisionPairingsDeletedResponse', null, global);
goog.exportSymbol('proto.liwords.DivisionPairingsResponse', null, global);
goog.exportSymbol('proto.liwords.DivisionRoundControls', null, global);
goog.exportSymbol('proto.liwords.ErrorMessage', null, global);
goog.exportSymbol('proto.liwords.FirstMethod', null, global);
goog.exportSymbol('proto.liwords.FullTournamentDivisions', null, global);
goog.exportSymbol('proto.liwords.GameDeletion', null, global);
goog.exportSymbol('proto.liwords.GameEndReason', null, global);
goog.exportSymbol('proto.liwords.GameEndedEvent', null, global);
goog.exportSymbol('proto.liwords.GameHistoryRefresher', null, global);
goog.exportSymbol('proto.liwords.GameMetaEvent', null, global);
goog.exportSymbol('proto.liwords.GameMetaEvent.EventType', null, global);
goog.exportSymbol('proto.liwords.GameMode', null, global);
goog.exportSymbol('proto.liwords.GameRequest', null, global);
goog.exportSymbol('proto.liwords.GameRules', null, global);
goog.exportSymbol('proto.liwords.JoinPath', null, global);
goog.exportSymbol('proto.liwords.LagMeasurement', null, global);
goog.exportSymbol('proto.liwords.MatchRequest', null, global);
goog.exportSymbol('proto.liwords.MatchRequestCancellation', null, global);
goog.exportSymbol('proto.liwords.MatchRequests', null, global);
goog.exportSymbol('proto.liwords.MatchUser', null, global);
goog.exportSymbol('proto.liwords.MessageType', null, global);
goog.exportSymbol('proto.liwords.NewGameEvent', null, global);
goog.exportSymbol('proto.liwords.Pairing', null, global);
goog.exportSymbol('proto.liwords.PairingMethod', null, global);
goog.exportSymbol('proto.liwords.PlayerStanding', null, global);
goog.exportSymbol('proto.liwords.PlayersAddedOrRemovedResponse', null, global);
goog.exportSymbol('proto.liwords.PresenceEntry', null, global);
goog.exportSymbol('proto.liwords.RatingMode', null, global);
goog.exportSymbol('proto.liwords.ReadyForGame', null, global);
goog.exportSymbol('proto.liwords.ReadyForTournamentGame', null, global);
goog.exportSymbol('proto.liwords.RematchStartedEvent', null, global);
goog.exportSymbol('proto.liwords.RoundControl', null, global);
goog.exportSymbol('proto.liwords.RoundStandings', null, global);
goog.exportSymbol('proto.liwords.SeekRequest', null, global);
goog.exportSymbol('proto.liwords.SeekRequests', null, global);
goog.exportSymbol('proto.liwords.ServerChallengeResultEvent', null, global);
goog.exportSymbol('proto.liwords.ServerGameplayEvent', null, global);
goog.exportSymbol('proto.liwords.ServerMessage', null, global);
goog.exportSymbol('proto.liwords.SoughtGameProcessEvent', null, global);
goog.exportSymbol('proto.liwords.TimedOut', null, global);
goog.exportSymbol('proto.liwords.TournamentDataResponse', null, global);
goog.exportSymbol('proto.liwords.TournamentDivisionDataResponse', null, global);
goog.exportSymbol('proto.liwords.TournamentDivisionDeletedResponse', null, global);
goog.exportSymbol('proto.liwords.TournamentFinishedResponse', null, global);
goog.exportSymbol('proto.liwords.TournamentGame', null, global);
goog.exportSymbol('proto.liwords.TournamentGameEndedEvent', null, global);
goog.exportSymbol('proto.liwords.TournamentGameEndedEvent.Player', null, global);
goog.exportSymbol('proto.liwords.TournamentGameResult', null, global);
goog.exportSymbol('proto.liwords.TournamentPerson', null, global);
goog.exportSymbol('proto.liwords.TournamentPersons', null, global);
goog.exportSymbol('proto.liwords.TournamentRoundStarted', null, global);
goog.exportSymbol('proto.liwords.UnjoinRealm', null, global);
goog.exportSymbol('proto.liwords.UserPresence', null, global);
goog.exportSymbol('proto.liwords.UserPresences', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameRules = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameRules, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameRules.displayName = 'proto.liwords.GameRules';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameRequest.displayName = 'proto.liwords.GameRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.MatchUser = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.MatchUser, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.MatchUser.displayName = 'proto.liwords.MatchUser';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameDeletion = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameDeletion, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameDeletion.displayName = 'proto.liwords.GameDeletion';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ActiveGamePlayer = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ActiveGamePlayer, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ActiveGamePlayer.displayName = 'proto.liwords.ActiveGamePlayer';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ActiveGameEntry = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.ActiveGameEntry.repeatedFields_, null);
};
goog.inherits(proto.liwords.ActiveGameEntry, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ActiveGameEntry.displayName = 'proto.liwords.ActiveGameEntry';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.LagMeasurement = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.LagMeasurement, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.LagMeasurement.displayName = 'proto.liwords.LagMeasurement';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ChatMessage = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ChatMessage, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ChatMessage.displayName = 'proto.liwords.ChatMessage';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ChatMessages = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.ChatMessages.repeatedFields_, null);
};
goog.inherits(proto.liwords.ChatMessages, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ChatMessages.displayName = 'proto.liwords.ChatMessages';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.UserPresence = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.UserPresence, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.UserPresence.displayName = 'proto.liwords.UserPresence';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.UserPresences = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.UserPresences.repeatedFields_, null);
};
goog.inherits(proto.liwords.UserPresences, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.UserPresences.displayName = 'proto.liwords.UserPresences';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.PresenceEntry = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.PresenceEntry.repeatedFields_, null);
};
goog.inherits(proto.liwords.PresenceEntry, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.PresenceEntry.displayName = 'proto.liwords.PresenceEntry';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.SeekRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.SeekRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.SeekRequest.displayName = 'proto.liwords.SeekRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.MatchRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.MatchRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.MatchRequest.displayName = 'proto.liwords.MatchRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ReadyForGame = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ReadyForGame, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ReadyForGame.displayName = 'proto.liwords.ReadyForGame';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.SoughtGameProcessEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.SoughtGameProcessEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.SoughtGameProcessEvent.displayName = 'proto.liwords.SoughtGameProcessEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.MatchRequestCancellation = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.MatchRequestCancellation, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.MatchRequestCancellation.displayName = 'proto.liwords.MatchRequestCancellation';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.SeekRequests = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.SeekRequests.repeatedFields_, null);
};
goog.inherits(proto.liwords.SeekRequests, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.SeekRequests.displayName = 'proto.liwords.SeekRequests';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.MatchRequests = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.MatchRequests.repeatedFields_, null);
};
goog.inherits(proto.liwords.MatchRequests, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.MatchRequests.displayName = 'proto.liwords.MatchRequests';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ServerGameplayEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ServerGameplayEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ServerGameplayEvent.displayName = 'proto.liwords.ServerGameplayEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ServerChallengeResultEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ServerChallengeResultEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ServerChallengeResultEvent.displayName = 'proto.liwords.ServerChallengeResultEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameEndedEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameEndedEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameEndedEvent.displayName = 'proto.liwords.GameEndedEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameMetaEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameMetaEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameMetaEvent.displayName = 'proto.liwords.GameMetaEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentGameEndedEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.TournamentGameEndedEvent.repeatedFields_, null);
};
goog.inherits(proto.liwords.TournamentGameEndedEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentGameEndedEvent.displayName = 'proto.liwords.TournamentGameEndedEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentGameEndedEvent.Player = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentGameEndedEvent.Player, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentGameEndedEvent.Player.displayName = 'proto.liwords.TournamentGameEndedEvent.Player';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentRoundStarted = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentRoundStarted, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentRoundStarted.displayName = 'proto.liwords.TournamentRoundStarted';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.RematchStartedEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.RematchStartedEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.RematchStartedEvent.displayName = 'proto.liwords.RematchStartedEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.GameHistoryRefresher = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.GameHistoryRefresher, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.GameHistoryRefresher.displayName = 'proto.liwords.GameHistoryRefresher';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.NewGameEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.NewGameEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.NewGameEvent.displayName = 'proto.liwords.NewGameEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ErrorMessage = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ErrorMessage, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ErrorMessage.displayName = 'proto.liwords.ErrorMessage';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ServerMessage = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ServerMessage, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ServerMessage.displayName = 'proto.liwords.ServerMessage';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ChatMessageDeleted = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ChatMessageDeleted, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ChatMessageDeleted.displayName = 'proto.liwords.ChatMessageDeleted';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ClientGameplayEvent = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ClientGameplayEvent, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ClientGameplayEvent.displayName = 'proto.liwords.ClientGameplayEvent';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.ReadyForTournamentGame = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.ReadyForTournamentGame, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.ReadyForTournamentGame.displayName = 'proto.liwords.ReadyForTournamentGame';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TimedOut = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TimedOut, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TimedOut.displayName = 'proto.liwords.TimedOut';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DeclineMatchRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.DeclineMatchRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DeclineMatchRequest.displayName = 'proto.liwords.DeclineMatchRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentPerson = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentPerson, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentPerson.displayName = 'proto.liwords.TournamentPerson';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentPersons = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.TournamentPersons.repeatedFields_, null);
};
goog.inherits(proto.liwords.TournamentPersons, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentPersons.displayName = 'proto.liwords.TournamentPersons';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.RoundControl = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.RoundControl, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.RoundControl.displayName = 'proto.liwords.RoundControl';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DivisionControls = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.DivisionControls, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DivisionControls.displayName = 'proto.liwords.DivisionControls';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentGame = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.TournamentGame.repeatedFields_, null);
};
goog.inherits(proto.liwords.TournamentGame, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentGame.displayName = 'proto.liwords.TournamentGame';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.Pairing = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.Pairing.repeatedFields_, null);
};
goog.inherits(proto.liwords.Pairing, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.Pairing.displayName = 'proto.liwords.Pairing';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.PlayerStanding = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.PlayerStanding, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.PlayerStanding.displayName = 'proto.liwords.PlayerStanding';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.RoundStandings = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.RoundStandings.repeatedFields_, null);
};
goog.inherits(proto.liwords.RoundStandings, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.RoundStandings.displayName = 'proto.liwords.RoundStandings';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DivisionPairingsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.DivisionPairingsResponse.repeatedFields_, null);
};
goog.inherits(proto.liwords.DivisionPairingsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DivisionPairingsResponse.displayName = 'proto.liwords.DivisionPairingsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DivisionPairingsDeletedResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.DivisionPairingsDeletedResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DivisionPairingsDeletedResponse.displayName = 'proto.liwords.DivisionPairingsDeletedResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.PlayersAddedOrRemovedResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.PlayersAddedOrRemovedResponse.repeatedFields_, null);
};
goog.inherits(proto.liwords.PlayersAddedOrRemovedResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.PlayersAddedOrRemovedResponse.displayName = 'proto.liwords.PlayersAddedOrRemovedResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DivisionRoundControls = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.DivisionRoundControls.repeatedFields_, null);
};
goog.inherits(proto.liwords.DivisionRoundControls, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DivisionRoundControls.displayName = 'proto.liwords.DivisionRoundControls';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.DivisionControlsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.DivisionControlsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.DivisionControlsResponse.displayName = 'proto.liwords.DivisionControlsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentDivisionDataResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.liwords.TournamentDivisionDataResponse.repeatedFields_, null);
};
goog.inherits(proto.liwords.TournamentDivisionDataResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentDivisionDataResponse.displayName = 'proto.liwords.TournamentDivisionDataResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.FullTournamentDivisions = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.FullTournamentDivisions, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.FullTournamentDivisions.displayName = 'proto.liwords.FullTournamentDivisions';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentFinishedResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentFinishedResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentFinishedResponse.displayName = 'proto.liwords.TournamentFinishedResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentDataResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentDataResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentDataResponse.displayName = 'proto.liwords.TournamentDataResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.TournamentDivisionDeletedResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.TournamentDivisionDeletedResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.TournamentDivisionDeletedResponse.displayName = 'proto.liwords.TournamentDivisionDeletedResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.JoinPath = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.JoinPath, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.JoinPath.displayName = 'proto.liwords.JoinPath';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.liwords.UnjoinRealm = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.liwords.UnjoinRealm, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.liwords.UnjoinRealm.displayName = 'proto.liwords.UnjoinRealm';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameRules.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameRules.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameRules} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameRules.toObject = function(includeInstance, msg) {
  var f, obj = {
    boardLayoutName: jspb.Message.getFieldWithDefault(msg, 1, ""),
    letterDistributionName: jspb.Message.getFieldWithDefault(msg, 2, ""),
    variantName: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameRules}
 */
proto.liwords.GameRules.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameRules;
  return proto.liwords.GameRules.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameRules} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameRules}
 */
proto.liwords.GameRules.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setBoardLayoutName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setLetterDistributionName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setVariantName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameRules.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameRules.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameRules} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameRules.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getBoardLayoutName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getLetterDistributionName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getVariantName();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional string board_layout_name = 1;
 * @return {string}
 */
proto.liwords.GameRules.prototype.getBoardLayoutName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRules} returns this
 */
proto.liwords.GameRules.prototype.setBoardLayoutName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string letter_distribution_name = 2;
 * @return {string}
 */
proto.liwords.GameRules.prototype.getLetterDistributionName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRules} returns this
 */
proto.liwords.GameRules.prototype.setLetterDistributionName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string variant_name = 3;
 * @return {string}
 */
proto.liwords.GameRules.prototype.getVariantName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRules} returns this
 */
proto.liwords.GameRules.prototype.setVariantName = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    lexicon: jspb.Message.getFieldWithDefault(msg, 1, ""),
    rules: (f = msg.getRules()) && proto.liwords.GameRules.toObject(includeInstance, f),
    initialTimeSeconds: jspb.Message.getFieldWithDefault(msg, 3, 0),
    incrementSeconds: jspb.Message.getFieldWithDefault(msg, 4, 0),
    challengeRule: jspb.Message.getFieldWithDefault(msg, 5, 0),
    gameMode: jspb.Message.getFieldWithDefault(msg, 6, 0),
    ratingMode: jspb.Message.getFieldWithDefault(msg, 7, 0),
    requestId: jspb.Message.getFieldWithDefault(msg, 8, ""),
    maxOvertimeMinutes: jspb.Message.getFieldWithDefault(msg, 9, 0),
    playerVsBot: jspb.Message.getBooleanFieldWithDefault(msg, 10, false),
    originalRequestId: jspb.Message.getFieldWithDefault(msg, 11, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameRequest}
 */
proto.liwords.GameRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameRequest;
  return proto.liwords.GameRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameRequest}
 */
proto.liwords.GameRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setLexicon(value);
      break;
    case 2:
      var value = new proto.liwords.GameRules;
      reader.readMessage(value,proto.liwords.GameRules.deserializeBinaryFromReader);
      msg.setRules(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setInitialTimeSeconds(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setIncrementSeconds(value);
      break;
    case 5:
      var value = /** @type {!proto.macondo.ChallengeRule} */ (reader.readEnum());
      msg.setChallengeRule(value);
      break;
    case 6:
      var value = /** @type {!proto.liwords.GameMode} */ (reader.readEnum());
      msg.setGameMode(value);
      break;
    case 7:
      var value = /** @type {!proto.liwords.RatingMode} */ (reader.readEnum());
      msg.setRatingMode(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setRequestId(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxOvertimeMinutes(value);
      break;
    case 10:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setPlayerVsBot(value);
      break;
    case 11:
      var value = /** @type {string} */ (reader.readString());
      msg.setOriginalRequestId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getLexicon();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRules();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.liwords.GameRules.serializeBinaryToWriter
    );
  }
  f = message.getInitialTimeSeconds();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getIncrementSeconds();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getChallengeRule();
  if (f !== 0.0) {
    writer.writeEnum(
      5,
      f
    );
  }
  f = message.getGameMode();
  if (f !== 0.0) {
    writer.writeEnum(
      6,
      f
    );
  }
  f = message.getRatingMode();
  if (f !== 0.0) {
    writer.writeEnum(
      7,
      f
    );
  }
  f = message.getRequestId();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
  f = message.getMaxOvertimeMinutes();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getPlayerVsBot();
  if (f) {
    writer.writeBool(
      10,
      f
    );
  }
  f = message.getOriginalRequestId();
  if (f.length > 0) {
    writer.writeString(
      11,
      f
    );
  }
};


/**
 * optional string lexicon = 1;
 * @return {string}
 */
proto.liwords.GameRequest.prototype.getLexicon = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setLexicon = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional GameRules rules = 2;
 * @return {?proto.liwords.GameRules}
 */
proto.liwords.GameRequest.prototype.getRules = function() {
  return /** @type{?proto.liwords.GameRules} */ (
    jspb.Message.getWrapperField(this, proto.liwords.GameRules, 2));
};


/**
 * @param {?proto.liwords.GameRules|undefined} value
 * @return {!proto.liwords.GameRequest} returns this
*/
proto.liwords.GameRequest.prototype.setRules = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.clearRules = function() {
  return this.setRules(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.GameRequest.prototype.hasRules = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional int32 initial_time_seconds = 3;
 * @return {number}
 */
proto.liwords.GameRequest.prototype.getInitialTimeSeconds = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setInitialTimeSeconds = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 increment_seconds = 4;
 * @return {number}
 */
proto.liwords.GameRequest.prototype.getIncrementSeconds = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setIncrementSeconds = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional macondo.ChallengeRule challenge_rule = 5;
 * @return {!proto.macondo.ChallengeRule}
 */
proto.liwords.GameRequest.prototype.getChallengeRule = function() {
  return /** @type {!proto.macondo.ChallengeRule} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {!proto.macondo.ChallengeRule} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setChallengeRule = function(value) {
  return jspb.Message.setProto3EnumField(this, 5, value);
};


/**
 * optional GameMode game_mode = 6;
 * @return {!proto.liwords.GameMode}
 */
proto.liwords.GameRequest.prototype.getGameMode = function() {
  return /** @type {!proto.liwords.GameMode} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {!proto.liwords.GameMode} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setGameMode = function(value) {
  return jspb.Message.setProto3EnumField(this, 6, value);
};


/**
 * optional RatingMode rating_mode = 7;
 * @return {!proto.liwords.RatingMode}
 */
proto.liwords.GameRequest.prototype.getRatingMode = function() {
  return /** @type {!proto.liwords.RatingMode} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {!proto.liwords.RatingMode} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setRatingMode = function(value) {
  return jspb.Message.setProto3EnumField(this, 7, value);
};


/**
 * optional string request_id = 8;
 * @return {string}
 */
proto.liwords.GameRequest.prototype.getRequestId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setRequestId = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};


/**
 * optional int32 max_overtime_minutes = 9;
 * @return {number}
 */
proto.liwords.GameRequest.prototype.getMaxOvertimeMinutes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setMaxOvertimeMinutes = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional bool player_vs_bot = 10;
 * @return {boolean}
 */
proto.liwords.GameRequest.prototype.getPlayerVsBot = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 10, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setPlayerVsBot = function(value) {
  return jspb.Message.setProto3BooleanField(this, 10, value);
};


/**
 * optional string original_request_id = 11;
 * @return {string}
 */
proto.liwords.GameRequest.prototype.getOriginalRequestId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 11, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameRequest} returns this
 */
proto.liwords.GameRequest.prototype.setOriginalRequestId = function(value) {
  return jspb.Message.setProto3StringField(this, 11, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.MatchUser.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.MatchUser.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.MatchUser} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchUser.toObject = function(includeInstance, msg) {
  var f, obj = {
    userId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    relevantRating: jspb.Message.getFieldWithDefault(msg, 2, ""),
    isAnonymous: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    displayName: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.MatchUser}
 */
proto.liwords.MatchUser.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.MatchUser;
  return proto.liwords.MatchUser.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.MatchUser} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.MatchUser}
 */
proto.liwords.MatchUser.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRelevantRating(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAnonymous(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setDisplayName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.MatchUser.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.MatchUser.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.MatchUser} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchUser.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRelevantRating();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getIsAnonymous();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getDisplayName();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional string user_id = 1;
 * @return {string}
 */
proto.liwords.MatchUser.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchUser} returns this
 */
proto.liwords.MatchUser.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string relevant_rating = 2;
 * @return {string}
 */
proto.liwords.MatchUser.prototype.getRelevantRating = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchUser} returns this
 */
proto.liwords.MatchUser.prototype.setRelevantRating = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional bool is_anonymous = 3;
 * @return {boolean}
 */
proto.liwords.MatchUser.prototype.getIsAnonymous = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.MatchUser} returns this
 */
proto.liwords.MatchUser.prototype.setIsAnonymous = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional string display_name = 4;
 * @return {string}
 */
proto.liwords.MatchUser.prototype.getDisplayName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchUser} returns this
 */
proto.liwords.MatchUser.prototype.setDisplayName = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameDeletion.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameDeletion.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameDeletion} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameDeletion.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameDeletion}
 */
proto.liwords.GameDeletion.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameDeletion;
  return proto.liwords.GameDeletion.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameDeletion} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameDeletion}
 */
proto.liwords.GameDeletion.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameDeletion.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameDeletion.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameDeletion} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameDeletion.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.GameDeletion.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameDeletion} returns this
 */
proto.liwords.GameDeletion.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ActiveGamePlayer.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ActiveGamePlayer.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ActiveGamePlayer} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ActiveGamePlayer.toObject = function(includeInstance, msg) {
  var f, obj = {
    username: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ActiveGamePlayer}
 */
proto.liwords.ActiveGamePlayer.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ActiveGamePlayer;
  return proto.liwords.ActiveGamePlayer.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ActiveGamePlayer} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ActiveGamePlayer}
 */
proto.liwords.ActiveGamePlayer.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUsername(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ActiveGamePlayer.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ActiveGamePlayer.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ActiveGamePlayer} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ActiveGamePlayer.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsername();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string username = 1;
 * @return {string}
 */
proto.liwords.ActiveGamePlayer.prototype.getUsername = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ActiveGamePlayer} returns this
 */
proto.liwords.ActiveGamePlayer.prototype.setUsername = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.liwords.ActiveGamePlayer.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ActiveGamePlayer} returns this
 */
proto.liwords.ActiveGamePlayer.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.ActiveGameEntry.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ActiveGameEntry.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ActiveGameEntry.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ActiveGameEntry} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ActiveGameEntry.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    playerList: jspb.Message.toObjectList(msg.getPlayerList(),
    proto.liwords.ActiveGamePlayer.toObject, includeInstance),
    ttl: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ActiveGameEntry}
 */
proto.liwords.ActiveGameEntry.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ActiveGameEntry;
  return proto.liwords.ActiveGameEntry.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ActiveGameEntry} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ActiveGameEntry}
 */
proto.liwords.ActiveGameEntry.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = new proto.liwords.ActiveGamePlayer;
      reader.readMessage(value,proto.liwords.ActiveGamePlayer.deserializeBinaryFromReader);
      msg.addPlayer(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTtl(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ActiveGameEntry.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ActiveGameEntry.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ActiveGameEntry} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ActiveGameEntry.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPlayerList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.liwords.ActiveGamePlayer.serializeBinaryToWriter
    );
  }
  f = message.getTtl();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.ActiveGameEntry.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ActiveGameEntry} returns this
 */
proto.liwords.ActiveGameEntry.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated ActiveGamePlayer player = 2;
 * @return {!Array<!proto.liwords.ActiveGamePlayer>}
 */
proto.liwords.ActiveGameEntry.prototype.getPlayerList = function() {
  return /** @type{!Array<!proto.liwords.ActiveGamePlayer>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.ActiveGamePlayer, 2));
};


/**
 * @param {!Array<!proto.liwords.ActiveGamePlayer>} value
 * @return {!proto.liwords.ActiveGameEntry} returns this
*/
proto.liwords.ActiveGameEntry.prototype.setPlayerList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.liwords.ActiveGamePlayer=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.ActiveGamePlayer}
 */
proto.liwords.ActiveGameEntry.prototype.addPlayer = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.liwords.ActiveGamePlayer, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.ActiveGameEntry} returns this
 */
proto.liwords.ActiveGameEntry.prototype.clearPlayerList = function() {
  return this.setPlayerList([]);
};


/**
 * optional int64 ttl = 3;
 * @return {number}
 */
proto.liwords.ActiveGameEntry.prototype.getTtl = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.ActiveGameEntry} returns this
 */
proto.liwords.ActiveGameEntry.prototype.setTtl = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.LagMeasurement.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.LagMeasurement.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.LagMeasurement} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.LagMeasurement.toObject = function(includeInstance, msg) {
  var f, obj = {
    lagMs: jspb.Message.getFieldWithDefault(msg, 1, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.LagMeasurement}
 */
proto.liwords.LagMeasurement.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.LagMeasurement;
  return proto.liwords.LagMeasurement.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.LagMeasurement} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.LagMeasurement}
 */
proto.liwords.LagMeasurement.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setLagMs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.LagMeasurement.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.LagMeasurement.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.LagMeasurement} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.LagMeasurement.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getLagMs();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
};


/**
 * optional int32 lag_ms = 1;
 * @return {number}
 */
proto.liwords.LagMeasurement.prototype.getLagMs = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.LagMeasurement} returns this
 */
proto.liwords.LagMeasurement.prototype.setLagMs = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ChatMessage.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ChatMessage.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ChatMessage} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessage.toObject = function(includeInstance, msg) {
  var f, obj = {
    username: jspb.Message.getFieldWithDefault(msg, 1, ""),
    channel: jspb.Message.getFieldWithDefault(msg, 2, ""),
    message: jspb.Message.getFieldWithDefault(msg, 3, ""),
    timestamp: jspb.Message.getFieldWithDefault(msg, 4, 0),
    userId: jspb.Message.getFieldWithDefault(msg, 5, ""),
    id: jspb.Message.getFieldWithDefault(msg, 6, ""),
    countryCode: jspb.Message.getFieldWithDefault(msg, 7, ""),
    avatarUrl: jspb.Message.getFieldWithDefault(msg, 8, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ChatMessage}
 */
proto.liwords.ChatMessage.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ChatMessage;
  return proto.liwords.ChatMessage.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ChatMessage} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ChatMessage}
 */
proto.liwords.ChatMessage.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUsername(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setChannel(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTimestamp(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setCountryCode(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setAvatarUrl(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ChatMessage.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ChatMessage.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ChatMessage} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessage.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsername();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getChannel();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getTimestamp();
  if (f !== 0) {
    writer.writeInt64(
      4,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getCountryCode();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getAvatarUrl();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
};


/**
 * optional string username = 1;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getUsername = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setUsername = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string channel = 2;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getChannel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setChannel = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string message = 3;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional int64 timestamp = 4;
 * @return {number}
 */
proto.liwords.ChatMessage.prototype.getTimestamp = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setTimestamp = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional string user_id = 5;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string id = 6;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string country_code = 7;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getCountryCode = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setCountryCode = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional string avatar_url = 8;
 * @return {string}
 */
proto.liwords.ChatMessage.prototype.getAvatarUrl = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessage} returns this
 */
proto.liwords.ChatMessage.prototype.setAvatarUrl = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.ChatMessages.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ChatMessages.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ChatMessages.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ChatMessages} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessages.toObject = function(includeInstance, msg) {
  var f, obj = {
    messagesList: jspb.Message.toObjectList(msg.getMessagesList(),
    proto.liwords.ChatMessage.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ChatMessages}
 */
proto.liwords.ChatMessages.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ChatMessages;
  return proto.liwords.ChatMessages.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ChatMessages} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ChatMessages}
 */
proto.liwords.ChatMessages.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.ChatMessage;
      reader.readMessage(value,proto.liwords.ChatMessage.deserializeBinaryFromReader);
      msg.addMessages(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ChatMessages.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ChatMessages.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ChatMessages} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessages.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMessagesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.liwords.ChatMessage.serializeBinaryToWriter
    );
  }
};


/**
 * repeated ChatMessage messages = 1;
 * @return {!Array<!proto.liwords.ChatMessage>}
 */
proto.liwords.ChatMessages.prototype.getMessagesList = function() {
  return /** @type{!Array<!proto.liwords.ChatMessage>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.ChatMessage, 1));
};


/**
 * @param {!Array<!proto.liwords.ChatMessage>} value
 * @return {!proto.liwords.ChatMessages} returns this
*/
proto.liwords.ChatMessages.prototype.setMessagesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.liwords.ChatMessage=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.ChatMessage}
 */
proto.liwords.ChatMessages.prototype.addMessages = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.liwords.ChatMessage, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.ChatMessages} returns this
 */
proto.liwords.ChatMessages.prototype.clearMessagesList = function() {
  return this.setMessagesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.UserPresence.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.UserPresence.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.UserPresence} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UserPresence.toObject = function(includeInstance, msg) {
  var f, obj = {
    username: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    channel: jspb.Message.getFieldWithDefault(msg, 3, ""),
    isAnonymous: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    deleting: jspb.Message.getBooleanFieldWithDefault(msg, 5, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.UserPresence}
 */
proto.liwords.UserPresence.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.UserPresence;
  return proto.liwords.UserPresence.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.UserPresence} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.UserPresence}
 */
proto.liwords.UserPresence.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUsername(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setChannel(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsAnonymous(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setDeleting(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.UserPresence.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.UserPresence.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.UserPresence} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UserPresence.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsername();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getChannel();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getIsAnonymous();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getDeleting();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
};


/**
 * optional string username = 1;
 * @return {string}
 */
proto.liwords.UserPresence.prototype.getUsername = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.UserPresence} returns this
 */
proto.liwords.UserPresence.prototype.setUsername = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.liwords.UserPresence.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.UserPresence} returns this
 */
proto.liwords.UserPresence.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string channel = 3;
 * @return {string}
 */
proto.liwords.UserPresence.prototype.getChannel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.UserPresence} returns this
 */
proto.liwords.UserPresence.prototype.setChannel = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional bool is_anonymous = 4;
 * @return {boolean}
 */
proto.liwords.UserPresence.prototype.getIsAnonymous = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.UserPresence} returns this
 */
proto.liwords.UserPresence.prototype.setIsAnonymous = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool deleting = 5;
 * @return {boolean}
 */
proto.liwords.UserPresence.prototype.getDeleting = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.UserPresence} returns this
 */
proto.liwords.UserPresence.prototype.setDeleting = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.UserPresences.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.UserPresences.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.UserPresences.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.UserPresences} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UserPresences.toObject = function(includeInstance, msg) {
  var f, obj = {
    presencesList: jspb.Message.toObjectList(msg.getPresencesList(),
    proto.liwords.UserPresence.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.UserPresences}
 */
proto.liwords.UserPresences.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.UserPresences;
  return proto.liwords.UserPresences.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.UserPresences} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.UserPresences}
 */
proto.liwords.UserPresences.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.UserPresence;
      reader.readMessage(value,proto.liwords.UserPresence.deserializeBinaryFromReader);
      msg.addPresences(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.UserPresences.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.UserPresences.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.UserPresences} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UserPresences.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPresencesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.liwords.UserPresence.serializeBinaryToWriter
    );
  }
};


/**
 * repeated UserPresence presences = 1;
 * @return {!Array<!proto.liwords.UserPresence>}
 */
proto.liwords.UserPresences.prototype.getPresencesList = function() {
  return /** @type{!Array<!proto.liwords.UserPresence>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.UserPresence, 1));
};


/**
 * @param {!Array<!proto.liwords.UserPresence>} value
 * @return {!proto.liwords.UserPresences} returns this
*/
proto.liwords.UserPresences.prototype.setPresencesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.liwords.UserPresence=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.UserPresence}
 */
proto.liwords.UserPresences.prototype.addPresences = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.liwords.UserPresence, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.UserPresences} returns this
 */
proto.liwords.UserPresences.prototype.clearPresencesList = function() {
  return this.setPresencesList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.PresenceEntry.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.PresenceEntry.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.PresenceEntry.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.PresenceEntry} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PresenceEntry.toObject = function(includeInstance, msg) {
  var f, obj = {
    username: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    channelList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.PresenceEntry}
 */
proto.liwords.PresenceEntry.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.PresenceEntry;
  return proto.liwords.PresenceEntry.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.PresenceEntry} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.PresenceEntry}
 */
proto.liwords.PresenceEntry.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUsername(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addChannel(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.PresenceEntry.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.PresenceEntry.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.PresenceEntry} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PresenceEntry.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsername();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getChannelList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
};


/**
 * optional string username = 1;
 * @return {string}
 */
proto.liwords.PresenceEntry.prototype.getUsername = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.PresenceEntry} returns this
 */
proto.liwords.PresenceEntry.prototype.setUsername = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.liwords.PresenceEntry.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.PresenceEntry} returns this
 */
proto.liwords.PresenceEntry.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated string channel = 3;
 * @return {!Array<string>}
 */
proto.liwords.PresenceEntry.prototype.getChannelList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.liwords.PresenceEntry} returns this
 */
proto.liwords.PresenceEntry.prototype.setChannelList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.liwords.PresenceEntry} returns this
 */
proto.liwords.PresenceEntry.prototype.addChannel = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.PresenceEntry} returns this
 */
proto.liwords.PresenceEntry.prototype.clearChannelList = function() {
  return this.setChannelList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.SeekRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.SeekRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.SeekRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SeekRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameRequest: (f = msg.getGameRequest()) && proto.liwords.GameRequest.toObject(includeInstance, f),
    user: (f = msg.getUser()) && proto.liwords.MatchUser.toObject(includeInstance, f),
    minimumRating: jspb.Message.getFieldWithDefault(msg, 3, 0),
    maximumRating: jspb.Message.getFieldWithDefault(msg, 4, 0),
    connectionId: jspb.Message.getFieldWithDefault(msg, 5, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.SeekRequest}
 */
proto.liwords.SeekRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.SeekRequest;
  return proto.liwords.SeekRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.SeekRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.SeekRequest}
 */
proto.liwords.SeekRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.GameRequest;
      reader.readMessage(value,proto.liwords.GameRequest.deserializeBinaryFromReader);
      msg.setGameRequest(value);
      break;
    case 2:
      var value = new proto.liwords.MatchUser;
      reader.readMessage(value,proto.liwords.MatchUser.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMinimumRating(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaximumRating(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setConnectionId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.SeekRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.SeekRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.SeekRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SeekRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameRequest();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.liwords.GameRequest.serializeBinaryToWriter
    );
  }
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.liwords.MatchUser.serializeBinaryToWriter
    );
  }
  f = message.getMinimumRating();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getMaximumRating();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getConnectionId();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
};


/**
 * optional GameRequest game_request = 1;
 * @return {?proto.liwords.GameRequest}
 */
proto.liwords.SeekRequest.prototype.getGameRequest = function() {
  return /** @type{?proto.liwords.GameRequest} */ (
    jspb.Message.getWrapperField(this, proto.liwords.GameRequest, 1));
};


/**
 * @param {?proto.liwords.GameRequest|undefined} value
 * @return {!proto.liwords.SeekRequest} returns this
*/
proto.liwords.SeekRequest.prototype.setGameRequest = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.SeekRequest} returns this
 */
proto.liwords.SeekRequest.prototype.clearGameRequest = function() {
  return this.setGameRequest(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.SeekRequest.prototype.hasGameRequest = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional MatchUser user = 2;
 * @return {?proto.liwords.MatchUser}
 */
proto.liwords.SeekRequest.prototype.getUser = function() {
  return /** @type{?proto.liwords.MatchUser} */ (
    jspb.Message.getWrapperField(this, proto.liwords.MatchUser, 2));
};


/**
 * @param {?proto.liwords.MatchUser|undefined} value
 * @return {!proto.liwords.SeekRequest} returns this
*/
proto.liwords.SeekRequest.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.SeekRequest} returns this
 */
proto.liwords.SeekRequest.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.SeekRequest.prototype.hasUser = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional int32 minimum_rating = 3;
 * @return {number}
 */
proto.liwords.SeekRequest.prototype.getMinimumRating = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.SeekRequest} returns this
 */
proto.liwords.SeekRequest.prototype.setMinimumRating = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 maximum_rating = 4;
 * @return {number}
 */
proto.liwords.SeekRequest.prototype.getMaximumRating = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.SeekRequest} returns this
 */
proto.liwords.SeekRequest.prototype.setMaximumRating = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional string connection_id = 5;
 * @return {string}
 */
proto.liwords.SeekRequest.prototype.getConnectionId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.SeekRequest} returns this
 */
proto.liwords.SeekRequest.prototype.setConnectionId = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.MatchRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.MatchRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.MatchRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameRequest: (f = msg.getGameRequest()) && proto.liwords.GameRequest.toObject(includeInstance, f),
    user: (f = msg.getUser()) && proto.liwords.MatchUser.toObject(includeInstance, f),
    receivingUser: (f = msg.getReceivingUser()) && proto.liwords.MatchUser.toObject(includeInstance, f),
    rematchFor: jspb.Message.getFieldWithDefault(msg, 4, ""),
    connectionId: jspb.Message.getFieldWithDefault(msg, 5, ""),
    tournamentId: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.MatchRequest}
 */
proto.liwords.MatchRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.MatchRequest;
  return proto.liwords.MatchRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.MatchRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.MatchRequest}
 */
proto.liwords.MatchRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.GameRequest;
      reader.readMessage(value,proto.liwords.GameRequest.deserializeBinaryFromReader);
      msg.setGameRequest(value);
      break;
    case 2:
      var value = new proto.liwords.MatchUser;
      reader.readMessage(value,proto.liwords.MatchUser.deserializeBinaryFromReader);
      msg.setUser(value);
      break;
    case 3:
      var value = new proto.liwords.MatchUser;
      reader.readMessage(value,proto.liwords.MatchUser.deserializeBinaryFromReader);
      msg.setReceivingUser(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setRematchFor(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setConnectionId(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setTournamentId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.MatchRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.MatchRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.MatchRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameRequest();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.liwords.GameRequest.serializeBinaryToWriter
    );
  }
  f = message.getUser();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.liwords.MatchUser.serializeBinaryToWriter
    );
  }
  f = message.getReceivingUser();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.liwords.MatchUser.serializeBinaryToWriter
    );
  }
  f = message.getRematchFor();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getConnectionId();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getTournamentId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * optional GameRequest game_request = 1;
 * @return {?proto.liwords.GameRequest}
 */
proto.liwords.MatchRequest.prototype.getGameRequest = function() {
  return /** @type{?proto.liwords.GameRequest} */ (
    jspb.Message.getWrapperField(this, proto.liwords.GameRequest, 1));
};


/**
 * @param {?proto.liwords.GameRequest|undefined} value
 * @return {!proto.liwords.MatchRequest} returns this
*/
proto.liwords.MatchRequest.prototype.setGameRequest = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.clearGameRequest = function() {
  return this.setGameRequest(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.MatchRequest.prototype.hasGameRequest = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional MatchUser user = 2;
 * @return {?proto.liwords.MatchUser}
 */
proto.liwords.MatchRequest.prototype.getUser = function() {
  return /** @type{?proto.liwords.MatchUser} */ (
    jspb.Message.getWrapperField(this, proto.liwords.MatchUser, 2));
};


/**
 * @param {?proto.liwords.MatchUser|undefined} value
 * @return {!proto.liwords.MatchRequest} returns this
*/
proto.liwords.MatchRequest.prototype.setUser = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.clearUser = function() {
  return this.setUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.MatchRequest.prototype.hasUser = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional MatchUser receiving_user = 3;
 * @return {?proto.liwords.MatchUser}
 */
proto.liwords.MatchRequest.prototype.getReceivingUser = function() {
  return /** @type{?proto.liwords.MatchUser} */ (
    jspb.Message.getWrapperField(this, proto.liwords.MatchUser, 3));
};


/**
 * @param {?proto.liwords.MatchUser|undefined} value
 * @return {!proto.liwords.MatchRequest} returns this
*/
proto.liwords.MatchRequest.prototype.setReceivingUser = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.clearReceivingUser = function() {
  return this.setReceivingUser(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.MatchRequest.prototype.hasReceivingUser = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional string rematch_for = 4;
 * @return {string}
 */
proto.liwords.MatchRequest.prototype.getRematchFor = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.setRematchFor = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string connection_id = 5;
 * @return {string}
 */
proto.liwords.MatchRequest.prototype.getConnectionId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.setConnectionId = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string tournament_id = 6;
 * @return {string}
 */
proto.liwords.MatchRequest.prototype.getTournamentId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchRequest} returns this
 */
proto.liwords.MatchRequest.prototype.setTournamentId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ReadyForGame.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ReadyForGame.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ReadyForGame} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ReadyForGame.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ReadyForGame}
 */
proto.liwords.ReadyForGame.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ReadyForGame;
  return proto.liwords.ReadyForGame.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ReadyForGame} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ReadyForGame}
 */
proto.liwords.ReadyForGame.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ReadyForGame.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ReadyForGame.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ReadyForGame} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ReadyForGame.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string game_id = 1;
 * @return {string}
 */
proto.liwords.ReadyForGame.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ReadyForGame} returns this
 */
proto.liwords.ReadyForGame.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.SoughtGameProcessEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.SoughtGameProcessEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.SoughtGameProcessEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SoughtGameProcessEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    requestId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.SoughtGameProcessEvent}
 */
proto.liwords.SoughtGameProcessEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.SoughtGameProcessEvent;
  return proto.liwords.SoughtGameProcessEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.SoughtGameProcessEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.SoughtGameProcessEvent}
 */
proto.liwords.SoughtGameProcessEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRequestId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.SoughtGameProcessEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.SoughtGameProcessEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.SoughtGameProcessEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SoughtGameProcessEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRequestId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string request_id = 1;
 * @return {string}
 */
proto.liwords.SoughtGameProcessEvent.prototype.getRequestId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.SoughtGameProcessEvent} returns this
 */
proto.liwords.SoughtGameProcessEvent.prototype.setRequestId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.MatchRequestCancellation.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.MatchRequestCancellation.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.MatchRequestCancellation} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequestCancellation.toObject = function(includeInstance, msg) {
  var f, obj = {
    requestId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.MatchRequestCancellation}
 */
proto.liwords.MatchRequestCancellation.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.MatchRequestCancellation;
  return proto.liwords.MatchRequestCancellation.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.MatchRequestCancellation} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.MatchRequestCancellation}
 */
proto.liwords.MatchRequestCancellation.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRequestId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.MatchRequestCancellation.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.MatchRequestCancellation.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.MatchRequestCancellation} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequestCancellation.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRequestId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string request_id = 1;
 * @return {string}
 */
proto.liwords.MatchRequestCancellation.prototype.getRequestId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.MatchRequestCancellation} returns this
 */
proto.liwords.MatchRequestCancellation.prototype.setRequestId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.SeekRequests.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.SeekRequests.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.SeekRequests.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.SeekRequests} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SeekRequests.toObject = function(includeInstance, msg) {
  var f, obj = {
    requestsList: jspb.Message.toObjectList(msg.getRequestsList(),
    proto.liwords.SeekRequest.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.SeekRequests}
 */
proto.liwords.SeekRequests.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.SeekRequests;
  return proto.liwords.SeekRequests.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.SeekRequests} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.SeekRequests}
 */
proto.liwords.SeekRequests.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.SeekRequest;
      reader.readMessage(value,proto.liwords.SeekRequest.deserializeBinaryFromReader);
      msg.addRequests(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.SeekRequests.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.SeekRequests.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.SeekRequests} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.SeekRequests.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRequestsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.liwords.SeekRequest.serializeBinaryToWriter
    );
  }
};


/**
 * repeated SeekRequest requests = 1;
 * @return {!Array<!proto.liwords.SeekRequest>}
 */
proto.liwords.SeekRequests.prototype.getRequestsList = function() {
  return /** @type{!Array<!proto.liwords.SeekRequest>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.SeekRequest, 1));
};


/**
 * @param {!Array<!proto.liwords.SeekRequest>} value
 * @return {!proto.liwords.SeekRequests} returns this
*/
proto.liwords.SeekRequests.prototype.setRequestsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.liwords.SeekRequest=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.SeekRequest}
 */
proto.liwords.SeekRequests.prototype.addRequests = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.liwords.SeekRequest, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.SeekRequests} returns this
 */
proto.liwords.SeekRequests.prototype.clearRequestsList = function() {
  return this.setRequestsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.MatchRequests.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.MatchRequests.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.MatchRequests.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.MatchRequests} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequests.toObject = function(includeInstance, msg) {
  var f, obj = {
    requestsList: jspb.Message.toObjectList(msg.getRequestsList(),
    proto.liwords.MatchRequest.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.MatchRequests}
 */
proto.liwords.MatchRequests.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.MatchRequests;
  return proto.liwords.MatchRequests.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.MatchRequests} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.MatchRequests}
 */
proto.liwords.MatchRequests.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.MatchRequest;
      reader.readMessage(value,proto.liwords.MatchRequest.deserializeBinaryFromReader);
      msg.addRequests(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.MatchRequests.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.MatchRequests.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.MatchRequests} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.MatchRequests.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRequestsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.liwords.MatchRequest.serializeBinaryToWriter
    );
  }
};


/**
 * repeated MatchRequest requests = 1;
 * @return {!Array<!proto.liwords.MatchRequest>}
 */
proto.liwords.MatchRequests.prototype.getRequestsList = function() {
  return /** @type{!Array<!proto.liwords.MatchRequest>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.MatchRequest, 1));
};


/**
 * @param {!Array<!proto.liwords.MatchRequest>} value
 * @return {!proto.liwords.MatchRequests} returns this
*/
proto.liwords.MatchRequests.prototype.setRequestsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.liwords.MatchRequest=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.MatchRequest}
 */
proto.liwords.MatchRequests.prototype.addRequests = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.liwords.MatchRequest, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.MatchRequests} returns this
 */
proto.liwords.MatchRequests.prototype.clearRequestsList = function() {
  return this.setRequestsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ServerGameplayEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ServerGameplayEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ServerGameplayEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerGameplayEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    event: (f = msg.getEvent()) && macondo_api_proto_macondo_macondo_pb.GameEvent.toObject(includeInstance, f),
    gameId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    newRack: jspb.Message.getFieldWithDefault(msg, 3, ""),
    timeRemaining: jspb.Message.getFieldWithDefault(msg, 4, 0),
    playing: jspb.Message.getFieldWithDefault(msg, 5, 0),
    userId: jspb.Message.getFieldWithDefault(msg, 6, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ServerGameplayEvent}
 */
proto.liwords.ServerGameplayEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ServerGameplayEvent;
  return proto.liwords.ServerGameplayEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ServerGameplayEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ServerGameplayEvent}
 */
proto.liwords.ServerGameplayEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new macondo_api_proto_macondo_macondo_pb.GameEvent;
      reader.readMessage(value,macondo_api_proto_macondo_macondo_pb.GameEvent.deserializeBinaryFromReader);
      msg.setEvent(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setNewRack(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTimeRemaining(value);
      break;
    case 5:
      var value = /** @type {!proto.macondo.PlayState} */ (reader.readEnum());
      msg.setPlaying(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ServerGameplayEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ServerGameplayEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ServerGameplayEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerGameplayEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEvent();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      macondo_api_proto_macondo_macondo_pb.GameEvent.serializeBinaryToWriter
    );
  }
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getNewRack();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getTimeRemaining();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getPlaying();
  if (f !== 0.0) {
    writer.writeEnum(
      5,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
};


/**
 * optional macondo.GameEvent event = 1;
 * @return {?proto.macondo.GameEvent}
 */
proto.liwords.ServerGameplayEvent.prototype.getEvent = function() {
  return /** @type{?proto.macondo.GameEvent} */ (
    jspb.Message.getWrapperField(this, macondo_api_proto_macondo_macondo_pb.GameEvent, 1));
};


/**
 * @param {?proto.macondo.GameEvent|undefined} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
*/
proto.liwords.ServerGameplayEvent.prototype.setEvent = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.clearEvent = function() {
  return this.setEvent(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.ServerGameplayEvent.prototype.hasEvent = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional string game_id = 2;
 * @return {string}
 */
proto.liwords.ServerGameplayEvent.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string new_rack = 3;
 * @return {string}
 */
proto.liwords.ServerGameplayEvent.prototype.getNewRack = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.setNewRack = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional int32 time_remaining = 4;
 * @return {number}
 */
proto.liwords.ServerGameplayEvent.prototype.getTimeRemaining = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.setTimeRemaining = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional macondo.PlayState playing = 5;
 * @return {!proto.macondo.PlayState}
 */
proto.liwords.ServerGameplayEvent.prototype.getPlaying = function() {
  return /** @type {!proto.macondo.PlayState} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {!proto.macondo.PlayState} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.setPlaying = function(value) {
  return jspb.Message.setProto3EnumField(this, 5, value);
};


/**
 * optional string user_id = 6;
 * @return {string}
 */
proto.liwords.ServerGameplayEvent.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerGameplayEvent} returns this
 */
proto.liwords.ServerGameplayEvent.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ServerChallengeResultEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ServerChallengeResultEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ServerChallengeResultEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerChallengeResultEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    valid: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    challenger: jspb.Message.getFieldWithDefault(msg, 2, ""),
    challengeRule: jspb.Message.getFieldWithDefault(msg, 3, 0),
    returnedTiles: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ServerChallengeResultEvent}
 */
proto.liwords.ServerChallengeResultEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ServerChallengeResultEvent;
  return proto.liwords.ServerChallengeResultEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ServerChallengeResultEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ServerChallengeResultEvent}
 */
proto.liwords.ServerChallengeResultEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setValid(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setChallenger(value);
      break;
    case 3:
      var value = /** @type {!proto.macondo.ChallengeRule} */ (reader.readEnum());
      msg.setChallengeRule(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setReturnedTiles(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ServerChallengeResultEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ServerChallengeResultEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ServerChallengeResultEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerChallengeResultEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getValid();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getChallenger();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getChallengeRule();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getReturnedTiles();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * optional bool valid = 1;
 * @return {boolean}
 */
proto.liwords.ServerChallengeResultEvent.prototype.getValid = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.ServerChallengeResultEvent} returns this
 */
proto.liwords.ServerChallengeResultEvent.prototype.setValid = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string challenger = 2;
 * @return {string}
 */
proto.liwords.ServerChallengeResultEvent.prototype.getChallenger = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerChallengeResultEvent} returns this
 */
proto.liwords.ServerChallengeResultEvent.prototype.setChallenger = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional macondo.ChallengeRule challenge_rule = 3;
 * @return {!proto.macondo.ChallengeRule}
 */
proto.liwords.ServerChallengeResultEvent.prototype.getChallengeRule = function() {
  return /** @type {!proto.macondo.ChallengeRule} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.macondo.ChallengeRule} value
 * @return {!proto.liwords.ServerChallengeResultEvent} returns this
 */
proto.liwords.ServerChallengeResultEvent.prototype.setChallengeRule = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional string returned_tiles = 4;
 * @return {string}
 */
proto.liwords.ServerChallengeResultEvent.prototype.getReturnedTiles = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerChallengeResultEvent} returns this
 */
proto.liwords.ServerChallengeResultEvent.prototype.setReturnedTiles = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameEndedEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameEndedEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameEndedEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameEndedEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    scoresMap: (f = msg.getScoresMap()) ? f.toObject(includeInstance, undefined) : [],
    newRatingsMap: (f = msg.getNewRatingsMap()) ? f.toObject(includeInstance, undefined) : [],
    endReason: jspb.Message.getFieldWithDefault(msg, 3, 0),
    winner: jspb.Message.getFieldWithDefault(msg, 4, ""),
    loser: jspb.Message.getFieldWithDefault(msg, 5, ""),
    tie: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    time: jspb.Message.getFieldWithDefault(msg, 7, 0),
    ratingDeltasMap: (f = msg.getRatingDeltasMap()) ? f.toObject(includeInstance, undefined) : [],
    history: (f = msg.getHistory()) && macondo_api_proto_macondo_macondo_pb.GameHistory.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameEndedEvent}
 */
proto.liwords.GameEndedEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameEndedEvent;
  return proto.liwords.GameEndedEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameEndedEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameEndedEvent}
 */
proto.liwords.GameEndedEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getScoresMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readInt32, null, "", 0);
         });
      break;
    case 2:
      var value = msg.getNewRatingsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readInt32, null, "", 0);
         });
      break;
    case 3:
      var value = /** @type {!proto.liwords.GameEndReason} */ (reader.readEnum());
      msg.setEndReason(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setWinner(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setLoser(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setTie(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTime(value);
      break;
    case 8:
      var value = msg.getRatingDeltasMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readInt32, null, "", 0);
         });
      break;
    case 9:
      var value = new macondo_api_proto_macondo_macondo_pb.GameHistory;
      reader.readMessage(value,macondo_api_proto_macondo_macondo_pb.GameHistory.deserializeBinaryFromReader);
      msg.setHistory(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameEndedEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameEndedEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameEndedEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameEndedEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getScoresMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeInt32);
  }
  f = message.getNewRatingsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(2, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeInt32);
  }
  f = message.getEndReason();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getWinner();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getLoser();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getTie();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getTime();
  if (f !== 0) {
    writer.writeInt64(
      7,
      f
    );
  }
  f = message.getRatingDeltasMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(8, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeInt32);
  }
  f = message.getHistory();
  if (f != null) {
    writer.writeMessage(
      9,
      f,
      macondo_api_proto_macondo_macondo_pb.GameHistory.serializeBinaryToWriter
    );
  }
};


/**
 * map<string, int32> scores = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,number>}
 */
proto.liwords.GameEndedEvent.prototype.getScoresMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,number>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.clearScoresMap = function() {
  this.getScoresMap().clear();
  return this;};


/**
 * map<string, int32> new_ratings = 2;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,number>}
 */
proto.liwords.GameEndedEvent.prototype.getNewRatingsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,number>} */ (
      jspb.Message.getMapField(this, 2, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.clearNewRatingsMap = function() {
  this.getNewRatingsMap().clear();
  return this;};


/**
 * optional GameEndReason end_reason = 3;
 * @return {!proto.liwords.GameEndReason}
 */
proto.liwords.GameEndedEvent.prototype.getEndReason = function() {
  return /** @type {!proto.liwords.GameEndReason} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.liwords.GameEndReason} value
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.setEndReason = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional string winner = 4;
 * @return {string}
 */
proto.liwords.GameEndedEvent.prototype.getWinner = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.setWinner = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string loser = 5;
 * @return {string}
 */
proto.liwords.GameEndedEvent.prototype.getLoser = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.setLoser = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional bool tie = 6;
 * @return {boolean}
 */
proto.liwords.GameEndedEvent.prototype.getTie = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.setTie = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional int64 time = 7;
 * @return {number}
 */
proto.liwords.GameEndedEvent.prototype.getTime = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.setTime = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};


/**
 * map<string, int32> rating_deltas = 8;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,number>}
 */
proto.liwords.GameEndedEvent.prototype.getRatingDeltasMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,number>} */ (
      jspb.Message.getMapField(this, 8, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.clearRatingDeltasMap = function() {
  this.getRatingDeltasMap().clear();
  return this;};


/**
 * optional macondo.GameHistory history = 9;
 * @return {?proto.macondo.GameHistory}
 */
proto.liwords.GameEndedEvent.prototype.getHistory = function() {
  return /** @type{?proto.macondo.GameHistory} */ (
    jspb.Message.getWrapperField(this, macondo_api_proto_macondo_macondo_pb.GameHistory, 9));
};


/**
 * @param {?proto.macondo.GameHistory|undefined} value
 * @return {!proto.liwords.GameEndedEvent} returns this
*/
proto.liwords.GameEndedEvent.prototype.setHistory = function(value) {
  return jspb.Message.setWrapperField(this, 9, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.GameEndedEvent} returns this
 */
proto.liwords.GameEndedEvent.prototype.clearHistory = function() {
  return this.setHistory(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.GameEndedEvent.prototype.hasHistory = function() {
  return jspb.Message.getField(this, 9) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameMetaEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameMetaEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameMetaEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameMetaEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    origEventId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    timestamp: (f = msg.getTimestamp()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    type: jspb.Message.getFieldWithDefault(msg, 3, 0),
    playerId: jspb.Message.getFieldWithDefault(msg, 4, ""),
    gameId: jspb.Message.getFieldWithDefault(msg, 5, ""),
    expiry: jspb.Message.getFieldWithDefault(msg, 6, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameMetaEvent}
 */
proto.liwords.GameMetaEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameMetaEvent;
  return proto.liwords.GameMetaEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameMetaEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameMetaEvent}
 */
proto.liwords.GameMetaEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setOrigEventId(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setTimestamp(value);
      break;
    case 3:
      var value = /** @type {!proto.liwords.GameMetaEvent.EventType} */ (reader.readEnum());
      msg.setType(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setPlayerId(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setExpiry(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameMetaEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameMetaEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameMetaEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameMetaEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getOrigEventId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getTimestamp();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getPlayerId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getExpiry();
  if (f !== 0) {
    writer.writeInt32(
      6,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.liwords.GameMetaEvent.EventType = {
  REQUEST_ABORT: 0,
  REQUEST_ADJUDICATION: 1,
  REQUEST_UNDO: 2,
  REQUEST_ADJOURN: 3,
  ABORT_ACCEPTED: 4,
  ABORT_DENIED: 5,
  ADJUDICATION_ACCEPTED: 6,
  ADJUDICATION_DENIED: 7,
  UNDO_ACCEPTED: 8,
  UNDO_DENIED: 9,
  ADD_TIME: 10,
  TIMER_EXPIRED: 11
};

/**
 * optional string orig_event_id = 1;
 * @return {string}
 */
proto.liwords.GameMetaEvent.prototype.getOrigEventId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.setOrigEventId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional google.protobuf.Timestamp timestamp = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.liwords.GameMetaEvent.prototype.getTimestamp = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.liwords.GameMetaEvent} returns this
*/
proto.liwords.GameMetaEvent.prototype.setTimestamp = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.clearTimestamp = function() {
  return this.setTimestamp(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.GameMetaEvent.prototype.hasTimestamp = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional EventType type = 3;
 * @return {!proto.liwords.GameMetaEvent.EventType}
 */
proto.liwords.GameMetaEvent.prototype.getType = function() {
  return /** @type {!proto.liwords.GameMetaEvent.EventType} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.liwords.GameMetaEvent.EventType} value
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional string player_id = 4;
 * @return {string}
 */
proto.liwords.GameMetaEvent.prototype.getPlayerId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.setPlayerId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string game_id = 5;
 * @return {string}
 */
proto.liwords.GameMetaEvent.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional int32 expiry = 6;
 * @return {number}
 */
proto.liwords.GameMetaEvent.prototype.getExpiry = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameMetaEvent} returns this
 */
proto.liwords.GameMetaEvent.prototype.setExpiry = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.TournamentGameEndedEvent.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentGameEndedEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentGameEndedEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentGameEndedEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGameEndedEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    playersList: jspb.Message.toObjectList(msg.getPlayersList(),
    proto.liwords.TournamentGameEndedEvent.Player.toObject, includeInstance),
    endReason: jspb.Message.getFieldWithDefault(msg, 3, 0),
    time: jspb.Message.getFieldWithDefault(msg, 4, 0),
    round: jspb.Message.getFieldWithDefault(msg, 5, 0),
    division: jspb.Message.getFieldWithDefault(msg, 6, ""),
    gameIndex: jspb.Message.getFieldWithDefault(msg, 7, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentGameEndedEvent}
 */
proto.liwords.TournamentGameEndedEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentGameEndedEvent;
  return proto.liwords.TournamentGameEndedEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentGameEndedEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentGameEndedEvent}
 */
proto.liwords.TournamentGameEndedEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 2:
      var value = new proto.liwords.TournamentGameEndedEvent.Player;
      reader.readMessage(value,proto.liwords.TournamentGameEndedEvent.Player.deserializeBinaryFromReader);
      msg.addPlayers(value);
      break;
    case 3:
      var value = /** @type {!proto.liwords.GameEndReason} */ (reader.readEnum());
      msg.setEndReason(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setTime(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setGameIndex(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentGameEndedEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentGameEndedEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentGameEndedEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGameEndedEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPlayersList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      proto.liwords.TournamentGameEndedEvent.Player.serializeBinaryToWriter
    );
  }
  f = message.getEndReason();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getTime();
  if (f !== 0) {
    writer.writeInt64(
      4,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getGameIndex();
  if (f !== 0) {
    writer.writeInt32(
      7,
      f
    );
  }
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentGameEndedEvent.Player.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentGameEndedEvent.Player} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGameEndedEvent.Player.toObject = function(includeInstance, msg) {
  var f, obj = {
    username: jspb.Message.getFieldWithDefault(msg, 1, ""),
    score: jspb.Message.getFieldWithDefault(msg, 2, 0),
    result: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentGameEndedEvent.Player}
 */
proto.liwords.TournamentGameEndedEvent.Player.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentGameEndedEvent.Player;
  return proto.liwords.TournamentGameEndedEvent.Player.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentGameEndedEvent.Player} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentGameEndedEvent.Player}
 */
proto.liwords.TournamentGameEndedEvent.Player.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setUsername(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setScore(value);
      break;
    case 3:
      var value = /** @type {!proto.liwords.TournamentGameResult} */ (reader.readEnum());
      msg.setResult(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentGameEndedEvent.Player.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentGameEndedEvent.Player} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGameEndedEvent.Player.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getUsername();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getScore();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getResult();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
};


/**
 * optional string username = 1;
 * @return {string}
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.getUsername = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentGameEndedEvent.Player} returns this
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.setUsername = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int32 score = 2;
 * @return {number}
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.getScore = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentGameEndedEvent.Player} returns this
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.setScore = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional TournamentGameResult result = 3;
 * @return {!proto.liwords.TournamentGameResult}
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.getResult = function() {
  return /** @type {!proto.liwords.TournamentGameResult} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.liwords.TournamentGameResult} value
 * @return {!proto.liwords.TournamentGameEndedEvent.Player} returns this
 */
proto.liwords.TournamentGameEndedEvent.Player.prototype.setResult = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional string game_id = 1;
 * @return {string}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated Player players = 2;
 * @return {!Array<!proto.liwords.TournamentGameEndedEvent.Player>}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getPlayersList = function() {
  return /** @type{!Array<!proto.liwords.TournamentGameEndedEvent.Player>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.TournamentGameEndedEvent.Player, 2));
};


/**
 * @param {!Array<!proto.liwords.TournamentGameEndedEvent.Player>} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
*/
proto.liwords.TournamentGameEndedEvent.prototype.setPlayersList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.liwords.TournamentGameEndedEvent.Player=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.TournamentGameEndedEvent.Player}
 */
proto.liwords.TournamentGameEndedEvent.prototype.addPlayers = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.liwords.TournamentGameEndedEvent.Player, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.clearPlayersList = function() {
  return this.setPlayersList([]);
};


/**
 * optional GameEndReason end_reason = 3;
 * @return {!proto.liwords.GameEndReason}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getEndReason = function() {
  return /** @type {!proto.liwords.GameEndReason} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.liwords.GameEndReason} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setEndReason = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional int64 time = 4;
 * @return {number}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getTime = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setTime = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional int32 round = 5;
 * @return {number}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional string division = 6;
 * @return {string}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional int32 game_index = 7;
 * @return {number}
 */
proto.liwords.TournamentGameEndedEvent.prototype.getGameIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentGameEndedEvent} returns this
 */
proto.liwords.TournamentGameEndedEvent.prototype.setGameIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentRoundStarted.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentRoundStarted.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentRoundStarted} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentRoundStarted.toObject = function(includeInstance, msg) {
  var f, obj = {
    tournamentId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    round: jspb.Message.getFieldWithDefault(msg, 3, 0),
    gameIndex: jspb.Message.getFieldWithDefault(msg, 4, 0),
    deadline: (f = msg.getDeadline()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentRoundStarted}
 */
proto.liwords.TournamentRoundStarted.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentRoundStarted;
  return proto.liwords.TournamentRoundStarted.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentRoundStarted} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentRoundStarted}
 */
proto.liwords.TournamentRoundStarted.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTournamentId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setGameIndex(value);
      break;
    case 5:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setDeadline(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentRoundStarted.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentRoundStarted.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentRoundStarted} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentRoundStarted.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTournamentId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getGameIndex();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getDeadline();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional string tournament_id = 1;
 * @return {string}
 */
proto.liwords.TournamentRoundStarted.prototype.getTournamentId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentRoundStarted} returns this
 */
proto.liwords.TournamentRoundStarted.prototype.setTournamentId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.TournamentRoundStarted.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentRoundStarted} returns this
 */
proto.liwords.TournamentRoundStarted.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 round = 3;
 * @return {number}
 */
proto.liwords.TournamentRoundStarted.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentRoundStarted} returns this
 */
proto.liwords.TournamentRoundStarted.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 game_index = 4;
 * @return {number}
 */
proto.liwords.TournamentRoundStarted.prototype.getGameIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentRoundStarted} returns this
 */
proto.liwords.TournamentRoundStarted.prototype.setGameIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional google.protobuf.Timestamp deadline = 5;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.liwords.TournamentRoundStarted.prototype.getDeadline = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 5));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.liwords.TournamentRoundStarted} returns this
*/
proto.liwords.TournamentRoundStarted.prototype.setDeadline = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.TournamentRoundStarted} returns this
 */
proto.liwords.TournamentRoundStarted.prototype.clearDeadline = function() {
  return this.setDeadline(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.TournamentRoundStarted.prototype.hasDeadline = function() {
  return jspb.Message.getField(this, 5) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.RematchStartedEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.RematchStartedEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.RematchStartedEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RematchStartedEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    rematchGameId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.RematchStartedEvent}
 */
proto.liwords.RematchStartedEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.RematchStartedEvent;
  return proto.liwords.RematchStartedEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.RematchStartedEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.RematchStartedEvent}
 */
proto.liwords.RematchStartedEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRematchGameId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.RematchStartedEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.RematchStartedEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.RematchStartedEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RematchStartedEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRematchGameId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string rematch_game_id = 1;
 * @return {string}
 */
proto.liwords.RematchStartedEvent.prototype.getRematchGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.RematchStartedEvent} returns this
 */
proto.liwords.RematchStartedEvent.prototype.setRematchGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.GameHistoryRefresher.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.GameHistoryRefresher.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.GameHistoryRefresher} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameHistoryRefresher.toObject = function(includeInstance, msg) {
  var f, obj = {
    history: (f = msg.getHistory()) && macondo_api_proto_macondo_macondo_pb.GameHistory.toObject(includeInstance, f),
    timePlayer1: jspb.Message.getFieldWithDefault(msg, 2, 0),
    timePlayer2: jspb.Message.getFieldWithDefault(msg, 3, 0),
    maxOvertimeMinutes: jspb.Message.getFieldWithDefault(msg, 4, 0),
    outstandingEvent: (f = msg.getOutstandingEvent()) && proto.liwords.GameMetaEvent.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.GameHistoryRefresher}
 */
proto.liwords.GameHistoryRefresher.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.GameHistoryRefresher;
  return proto.liwords.GameHistoryRefresher.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.GameHistoryRefresher} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.GameHistoryRefresher}
 */
proto.liwords.GameHistoryRefresher.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new macondo_api_proto_macondo_macondo_pb.GameHistory;
      reader.readMessage(value,macondo_api_proto_macondo_macondo_pb.GameHistory.deserializeBinaryFromReader);
      msg.setHistory(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTimePlayer1(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setTimePlayer2(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxOvertimeMinutes(value);
      break;
    case 5:
      var value = new proto.liwords.GameMetaEvent;
      reader.readMessage(value,proto.liwords.GameMetaEvent.deserializeBinaryFromReader);
      msg.setOutstandingEvent(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.GameHistoryRefresher.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.GameHistoryRefresher.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.GameHistoryRefresher} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.GameHistoryRefresher.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getHistory();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      macondo_api_proto_macondo_macondo_pb.GameHistory.serializeBinaryToWriter
    );
  }
  f = message.getTimePlayer1();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getTimePlayer2();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getMaxOvertimeMinutes();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getOutstandingEvent();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.liwords.GameMetaEvent.serializeBinaryToWriter
    );
  }
};


/**
 * optional macondo.GameHistory history = 1;
 * @return {?proto.macondo.GameHistory}
 */
proto.liwords.GameHistoryRefresher.prototype.getHistory = function() {
  return /** @type{?proto.macondo.GameHistory} */ (
    jspb.Message.getWrapperField(this, macondo_api_proto_macondo_macondo_pb.GameHistory, 1));
};


/**
 * @param {?proto.macondo.GameHistory|undefined} value
 * @return {!proto.liwords.GameHistoryRefresher} returns this
*/
proto.liwords.GameHistoryRefresher.prototype.setHistory = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.GameHistoryRefresher} returns this
 */
proto.liwords.GameHistoryRefresher.prototype.clearHistory = function() {
  return this.setHistory(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.GameHistoryRefresher.prototype.hasHistory = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional int32 time_player1 = 2;
 * @return {number}
 */
proto.liwords.GameHistoryRefresher.prototype.getTimePlayer1 = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameHistoryRefresher} returns this
 */
proto.liwords.GameHistoryRefresher.prototype.setTimePlayer1 = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional int32 time_player2 = 3;
 * @return {number}
 */
proto.liwords.GameHistoryRefresher.prototype.getTimePlayer2 = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameHistoryRefresher} returns this
 */
proto.liwords.GameHistoryRefresher.prototype.setTimePlayer2 = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 max_overtime_minutes = 4;
 * @return {number}
 */
proto.liwords.GameHistoryRefresher.prototype.getMaxOvertimeMinutes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.GameHistoryRefresher} returns this
 */
proto.liwords.GameHistoryRefresher.prototype.setMaxOvertimeMinutes = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional GameMetaEvent outstanding_event = 5;
 * @return {?proto.liwords.GameMetaEvent}
 */
proto.liwords.GameHistoryRefresher.prototype.getOutstandingEvent = function() {
  return /** @type{?proto.liwords.GameMetaEvent} */ (
    jspb.Message.getWrapperField(this, proto.liwords.GameMetaEvent, 5));
};


/**
 * @param {?proto.liwords.GameMetaEvent|undefined} value
 * @return {!proto.liwords.GameHistoryRefresher} returns this
*/
proto.liwords.GameHistoryRefresher.prototype.setOutstandingEvent = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.GameHistoryRefresher} returns this
 */
proto.liwords.GameHistoryRefresher.prototype.clearOutstandingEvent = function() {
  return this.setOutstandingEvent(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.GameHistoryRefresher.prototype.hasOutstandingEvent = function() {
  return jspb.Message.getField(this, 5) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.NewGameEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.NewGameEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.NewGameEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.NewGameEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    requesterCid: jspb.Message.getFieldWithDefault(msg, 2, ""),
    accepterCid: jspb.Message.getFieldWithDefault(msg, 3, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.NewGameEvent}
 */
proto.liwords.NewGameEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.NewGameEvent;
  return proto.liwords.NewGameEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.NewGameEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.NewGameEvent}
 */
proto.liwords.NewGameEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setRequesterCid(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setAccepterCid(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.NewGameEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.NewGameEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.NewGameEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.NewGameEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRequesterCid();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getAccepterCid();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
};


/**
 * optional string game_id = 1;
 * @return {string}
 */
proto.liwords.NewGameEvent.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.NewGameEvent} returns this
 */
proto.liwords.NewGameEvent.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string requester_cid = 2;
 * @return {string}
 */
proto.liwords.NewGameEvent.prototype.getRequesterCid = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.NewGameEvent} returns this
 */
proto.liwords.NewGameEvent.prototype.setRequesterCid = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string accepter_cid = 3;
 * @return {string}
 */
proto.liwords.NewGameEvent.prototype.getAccepterCid = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.NewGameEvent} returns this
 */
proto.liwords.NewGameEvent.prototype.setAccepterCid = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ErrorMessage.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ErrorMessage.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ErrorMessage} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ErrorMessage.toObject = function(includeInstance, msg) {
  var f, obj = {
    message: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ErrorMessage}
 */
proto.liwords.ErrorMessage.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ErrorMessage;
  return proto.liwords.ErrorMessage.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ErrorMessage} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ErrorMessage}
 */
proto.liwords.ErrorMessage.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ErrorMessage.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ErrorMessage.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ErrorMessage} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ErrorMessage.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string message = 1;
 * @return {string}
 */
proto.liwords.ErrorMessage.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ErrorMessage} returns this
 */
proto.liwords.ErrorMessage.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ServerMessage.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ServerMessage.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ServerMessage} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerMessage.toObject = function(includeInstance, msg) {
  var f, obj = {
    message: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ServerMessage}
 */
proto.liwords.ServerMessage.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ServerMessage;
  return proto.liwords.ServerMessage.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ServerMessage} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ServerMessage}
 */
proto.liwords.ServerMessage.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ServerMessage.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ServerMessage.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ServerMessage} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ServerMessage.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string message = 1;
 * @return {string}
 */
proto.liwords.ServerMessage.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ServerMessage} returns this
 */
proto.liwords.ServerMessage.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ChatMessageDeleted.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ChatMessageDeleted.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ChatMessageDeleted} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessageDeleted.toObject = function(includeInstance, msg) {
  var f, obj = {
    channel: jspb.Message.getFieldWithDefault(msg, 1, ""),
    id: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ChatMessageDeleted}
 */
proto.liwords.ChatMessageDeleted.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ChatMessageDeleted;
  return proto.liwords.ChatMessageDeleted.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ChatMessageDeleted} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ChatMessageDeleted}
 */
proto.liwords.ChatMessageDeleted.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setChannel(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ChatMessageDeleted.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ChatMessageDeleted.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ChatMessageDeleted} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ChatMessageDeleted.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getChannel();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string channel = 1;
 * @return {string}
 */
proto.liwords.ChatMessageDeleted.prototype.getChannel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessageDeleted} returns this
 */
proto.liwords.ChatMessageDeleted.prototype.setChannel = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string id = 2;
 * @return {string}
 */
proto.liwords.ChatMessageDeleted.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ChatMessageDeleted} returns this
 */
proto.liwords.ChatMessageDeleted.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ClientGameplayEvent.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ClientGameplayEvent.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ClientGameplayEvent} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ClientGameplayEvent.toObject = function(includeInstance, msg) {
  var f, obj = {
    type: jspb.Message.getFieldWithDefault(msg, 1, 0),
    gameId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    positionCoords: jspb.Message.getFieldWithDefault(msg, 3, ""),
    tiles: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ClientGameplayEvent}
 */
proto.liwords.ClientGameplayEvent.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ClientGameplayEvent;
  return proto.liwords.ClientGameplayEvent.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ClientGameplayEvent} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ClientGameplayEvent}
 */
proto.liwords.ClientGameplayEvent.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.liwords.ClientGameplayEvent.EventType} */ (reader.readEnum());
      msg.setType(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setPositionCoords(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setTiles(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ClientGameplayEvent.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ClientGameplayEvent.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ClientGameplayEvent} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ClientGameplayEvent.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getType();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPositionCoords();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getTiles();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * @enum {number}
 */
proto.liwords.ClientGameplayEvent.EventType = {
  TILE_PLACEMENT: 0,
  PASS: 1,
  EXCHANGE: 2,
  CHALLENGE_PLAY: 3,
  RESIGN: 4
};

/**
 * optional EventType type = 1;
 * @return {!proto.liwords.ClientGameplayEvent.EventType}
 */
proto.liwords.ClientGameplayEvent.prototype.getType = function() {
  return /** @type {!proto.liwords.ClientGameplayEvent.EventType} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.liwords.ClientGameplayEvent.EventType} value
 * @return {!proto.liwords.ClientGameplayEvent} returns this
 */
proto.liwords.ClientGameplayEvent.prototype.setType = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional string game_id = 2;
 * @return {string}
 */
proto.liwords.ClientGameplayEvent.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ClientGameplayEvent} returns this
 */
proto.liwords.ClientGameplayEvent.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string position_coords = 3;
 * @return {string}
 */
proto.liwords.ClientGameplayEvent.prototype.getPositionCoords = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ClientGameplayEvent} returns this
 */
proto.liwords.ClientGameplayEvent.prototype.setPositionCoords = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string tiles = 4;
 * @return {string}
 */
proto.liwords.ClientGameplayEvent.prototype.getTiles = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ClientGameplayEvent} returns this
 */
proto.liwords.ClientGameplayEvent.prototype.setTiles = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.ReadyForTournamentGame.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.ReadyForTournamentGame.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.ReadyForTournamentGame} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ReadyForTournamentGame.toObject = function(includeInstance, msg) {
  var f, obj = {
    tournamentId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    round: jspb.Message.getFieldWithDefault(msg, 3, 0),
    playerId: jspb.Message.getFieldWithDefault(msg, 4, ""),
    gameIndex: jspb.Message.getFieldWithDefault(msg, 5, 0),
    unready: jspb.Message.getBooleanFieldWithDefault(msg, 6, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.ReadyForTournamentGame}
 */
proto.liwords.ReadyForTournamentGame.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.ReadyForTournamentGame;
  return proto.liwords.ReadyForTournamentGame.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.ReadyForTournamentGame} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.ReadyForTournamentGame}
 */
proto.liwords.ReadyForTournamentGame.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setTournamentId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setPlayerId(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setGameIndex(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setUnready(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.ReadyForTournamentGame.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.ReadyForTournamentGame.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.ReadyForTournamentGame} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.ReadyForTournamentGame.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getTournamentId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getPlayerId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getGameIndex();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getUnready();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
};


/**
 * optional string tournament_id = 1;
 * @return {string}
 */
proto.liwords.ReadyForTournamentGame.prototype.getTournamentId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setTournamentId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.ReadyForTournamentGame.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 round = 3;
 * @return {number}
 */
proto.liwords.ReadyForTournamentGame.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional string player_id = 4;
 * @return {string}
 */
proto.liwords.ReadyForTournamentGame.prototype.getPlayerId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setPlayerId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional int32 game_index = 5;
 * @return {number}
 */
proto.liwords.ReadyForTournamentGame.prototype.getGameIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setGameIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional bool unready = 6;
 * @return {boolean}
 */
proto.liwords.ReadyForTournamentGame.prototype.getUnready = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.ReadyForTournamentGame} returns this
 */
proto.liwords.ReadyForTournamentGame.prototype.setUnready = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TimedOut.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TimedOut.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TimedOut} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TimedOut.toObject = function(includeInstance, msg) {
  var f, obj = {
    gameId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    userId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TimedOut}
 */
proto.liwords.TimedOut.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TimedOut;
  return proto.liwords.TimedOut.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TimedOut} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TimedOut}
 */
proto.liwords.TimedOut.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGameId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setUserId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TimedOut.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TimedOut.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TimedOut} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TimedOut.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGameId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getUserId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string game_id = 1;
 * @return {string}
 */
proto.liwords.TimedOut.prototype.getGameId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TimedOut} returns this
 */
proto.liwords.TimedOut.prototype.setGameId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string user_id = 2;
 * @return {string}
 */
proto.liwords.TimedOut.prototype.getUserId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TimedOut} returns this
 */
proto.liwords.TimedOut.prototype.setUserId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DeclineMatchRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DeclineMatchRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DeclineMatchRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DeclineMatchRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    requestId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DeclineMatchRequest}
 */
proto.liwords.DeclineMatchRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DeclineMatchRequest;
  return proto.liwords.DeclineMatchRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DeclineMatchRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DeclineMatchRequest}
 */
proto.liwords.DeclineMatchRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setRequestId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DeclineMatchRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DeclineMatchRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DeclineMatchRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DeclineMatchRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getRequestId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string request_id = 1;
 * @return {string}
 */
proto.liwords.DeclineMatchRequest.prototype.getRequestId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DeclineMatchRequest} returns this
 */
proto.liwords.DeclineMatchRequest.prototype.setRequestId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentPerson.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentPerson.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentPerson} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentPerson.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    rating: jspb.Message.getFieldWithDefault(msg, 2, 0),
    suspended: jspb.Message.getBooleanFieldWithDefault(msg, 3, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentPerson}
 */
proto.liwords.TournamentPerson.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentPerson;
  return proto.liwords.TournamentPerson.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentPerson} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentPerson}
 */
proto.liwords.TournamentPerson.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRating(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuspended(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentPerson.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentPerson.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentPerson} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentPerson.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRating();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getSuspended();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentPerson.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentPerson} returns this
 */
proto.liwords.TournamentPerson.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int32 rating = 2;
 * @return {number}
 */
proto.liwords.TournamentPerson.prototype.getRating = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentPerson} returns this
 */
proto.liwords.TournamentPerson.prototype.setRating = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional bool suspended = 3;
 * @return {boolean}
 */
proto.liwords.TournamentPerson.prototype.getSuspended = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.TournamentPerson} returns this
 */
proto.liwords.TournamentPerson.prototype.setSuspended = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.TournamentPersons.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentPersons.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentPersons.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentPersons} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentPersons.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    personsList: jspb.Message.toObjectList(msg.getPersonsList(),
    proto.liwords.TournamentPerson.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentPersons}
 */
proto.liwords.TournamentPersons.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentPersons;
  return proto.liwords.TournamentPersons.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentPersons} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentPersons}
 */
proto.liwords.TournamentPersons.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.TournamentPerson;
      reader.readMessage(value,proto.liwords.TournamentPerson.deserializeBinaryFromReader);
      msg.addPersons(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentPersons.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentPersons.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentPersons} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentPersons.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPersonsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.liwords.TournamentPerson.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentPersons.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentPersons} returns this
 */
proto.liwords.TournamentPersons.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.TournamentPersons.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentPersons} returns this
 */
proto.liwords.TournamentPersons.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated TournamentPerson persons = 3;
 * @return {!Array<!proto.liwords.TournamentPerson>}
 */
proto.liwords.TournamentPersons.prototype.getPersonsList = function() {
  return /** @type{!Array<!proto.liwords.TournamentPerson>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.TournamentPerson, 3));
};


/**
 * @param {!Array<!proto.liwords.TournamentPerson>} value
 * @return {!proto.liwords.TournamentPersons} returns this
*/
proto.liwords.TournamentPersons.prototype.setPersonsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.liwords.TournamentPerson=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.TournamentPerson}
 */
proto.liwords.TournamentPersons.prototype.addPersons = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.liwords.TournamentPerson, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.TournamentPersons} returns this
 */
proto.liwords.TournamentPersons.prototype.clearPersonsList = function() {
  return this.setPersonsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.RoundControl.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.RoundControl.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.RoundControl} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RoundControl.toObject = function(includeInstance, msg) {
  var f, obj = {
    pairingMethod: jspb.Message.getFieldWithDefault(msg, 1, 0),
    firstMethod: jspb.Message.getFieldWithDefault(msg, 2, 0),
    gamesPerRound: jspb.Message.getFieldWithDefault(msg, 3, 0),
    round: jspb.Message.getFieldWithDefault(msg, 4, 0),
    factor: jspb.Message.getFieldWithDefault(msg, 5, 0),
    initialFontes: jspb.Message.getFieldWithDefault(msg, 6, 0),
    maxRepeats: jspb.Message.getFieldWithDefault(msg, 7, 0),
    allowOverMaxRepeats: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    repeatRelativeWeight: jspb.Message.getFieldWithDefault(msg, 9, 0),
    winDifferenceRelativeWeight: jspb.Message.getFieldWithDefault(msg, 10, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.RoundControl}
 */
proto.liwords.RoundControl.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.RoundControl;
  return proto.liwords.RoundControl.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.RoundControl} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.RoundControl}
 */
proto.liwords.RoundControl.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.liwords.PairingMethod} */ (reader.readEnum());
      msg.setPairingMethod(value);
      break;
    case 2:
      var value = /** @type {!proto.liwords.FirstMethod} */ (reader.readEnum());
      msg.setFirstMethod(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setGamesPerRound(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFactor(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setInitialFontes(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMaxRepeats(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAllowOverMaxRepeats(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRepeatRelativeWeight(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setWinDifferenceRelativeWeight(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.RoundControl.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.RoundControl.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.RoundControl} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RoundControl.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPairingMethod();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getFirstMethod();
  if (f !== 0.0) {
    writer.writeEnum(
      2,
      f
    );
  }
  f = message.getGamesPerRound();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getFactor();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getInitialFontes();
  if (f !== 0) {
    writer.writeInt32(
      6,
      f
    );
  }
  f = message.getMaxRepeats();
  if (f !== 0) {
    writer.writeInt32(
      7,
      f
    );
  }
  f = message.getAllowOverMaxRepeats();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getRepeatRelativeWeight();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getWinDifferenceRelativeWeight();
  if (f !== 0) {
    writer.writeInt32(
      10,
      f
    );
  }
};


/**
 * optional PairingMethod pairing_method = 1;
 * @return {!proto.liwords.PairingMethod}
 */
proto.liwords.RoundControl.prototype.getPairingMethod = function() {
  return /** @type {!proto.liwords.PairingMethod} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.liwords.PairingMethod} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setPairingMethod = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional FirstMethod first_method = 2;
 * @return {!proto.liwords.FirstMethod}
 */
proto.liwords.RoundControl.prototype.getFirstMethod = function() {
  return /** @type {!proto.liwords.FirstMethod} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {!proto.liwords.FirstMethod} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setFirstMethod = function(value) {
  return jspb.Message.setProto3EnumField(this, 2, value);
};


/**
 * optional int32 games_per_round = 3;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getGamesPerRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setGamesPerRound = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 round = 4;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional int32 factor = 5;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getFactor = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setFactor = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional int32 initial_fontes = 6;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getInitialFontes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setInitialFontes = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};


/**
 * optional int32 max_repeats = 7;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getMaxRepeats = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setMaxRepeats = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};


/**
 * optional bool allow_over_max_repeats = 8;
 * @return {boolean}
 */
proto.liwords.RoundControl.prototype.getAllowOverMaxRepeats = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setAllowOverMaxRepeats = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional int32 repeat_relative_weight = 9;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getRepeatRelativeWeight = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setRepeatRelativeWeight = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional int32 win_difference_relative_weight = 10;
 * @return {number}
 */
proto.liwords.RoundControl.prototype.getWinDifferenceRelativeWeight = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.RoundControl} returns this
 */
proto.liwords.RoundControl.prototype.setWinDifferenceRelativeWeight = function(value) {
  return jspb.Message.setProto3IntField(this, 10, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DivisionControls.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DivisionControls.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DivisionControls} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionControls.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    gameRequest: (f = msg.getGameRequest()) && proto.liwords.GameRequest.toObject(includeInstance, f),
    suspendedResult: jspb.Message.getFieldWithDefault(msg, 4, 0),
    suspendedSpread: jspb.Message.getFieldWithDefault(msg, 5, 0),
    autoStart: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    spreadCap: jspb.Message.getFieldWithDefault(msg, 7, 0),
    gibsonize: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    gibsonSpread: jspb.Message.getFieldWithDefault(msg, 9, 0),
    minimumPlacement: jspb.Message.getFieldWithDefault(msg, 10, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DivisionControls}
 */
proto.liwords.DivisionControls.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DivisionControls;
  return proto.liwords.DivisionControls.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DivisionControls} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DivisionControls}
 */
proto.liwords.DivisionControls.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.GameRequest;
      reader.readMessage(value,proto.liwords.GameRequest.deserializeBinaryFromReader);
      msg.setGameRequest(value);
      break;
    case 4:
      var value = /** @type {!proto.liwords.TournamentGameResult} */ (reader.readEnum());
      msg.setSuspendedResult(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setSuspendedSpread(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setAutoStart(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setSpreadCap(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setGibsonize(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setGibsonSpread(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setMinimumPlacement(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DivisionControls.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DivisionControls.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DivisionControls} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionControls.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getGameRequest();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.liwords.GameRequest.serializeBinaryToWriter
    );
  }
  f = message.getSuspendedResult();
  if (f !== 0.0) {
    writer.writeEnum(
      4,
      f
    );
  }
  f = message.getSuspendedSpread();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getAutoStart();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getSpreadCap();
  if (f !== 0) {
    writer.writeInt32(
      7,
      f
    );
  }
  f = message.getGibsonize();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getGibsonSpread();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getMinimumPlacement();
  if (f !== 0) {
    writer.writeInt32(
      10,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.DivisionControls.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.DivisionControls.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional GameRequest game_request = 3;
 * @return {?proto.liwords.GameRequest}
 */
proto.liwords.DivisionControls.prototype.getGameRequest = function() {
  return /** @type{?proto.liwords.GameRequest} */ (
    jspb.Message.getWrapperField(this, proto.liwords.GameRequest, 3));
};


/**
 * @param {?proto.liwords.GameRequest|undefined} value
 * @return {!proto.liwords.DivisionControls} returns this
*/
proto.liwords.DivisionControls.prototype.setGameRequest = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.clearGameRequest = function() {
  return this.setGameRequest(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.DivisionControls.prototype.hasGameRequest = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional TournamentGameResult suspended_result = 4;
 * @return {!proto.liwords.TournamentGameResult}
 */
proto.liwords.DivisionControls.prototype.getSuspendedResult = function() {
  return /** @type {!proto.liwords.TournamentGameResult} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.liwords.TournamentGameResult} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setSuspendedResult = function(value) {
  return jspb.Message.setProto3EnumField(this, 4, value);
};


/**
 * optional int32 suspended_spread = 5;
 * @return {number}
 */
proto.liwords.DivisionControls.prototype.getSuspendedSpread = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setSuspendedSpread = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional bool auto_start = 6;
 * @return {boolean}
 */
proto.liwords.DivisionControls.prototype.getAutoStart = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setAutoStart = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional int32 spread_cap = 7;
 * @return {number}
 */
proto.liwords.DivisionControls.prototype.getSpreadCap = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setSpreadCap = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};


/**
 * optional bool gibsonize = 8;
 * @return {boolean}
 */
proto.liwords.DivisionControls.prototype.getGibsonize = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setGibsonize = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional int32 gibson_spread = 9;
 * @return {number}
 */
proto.liwords.DivisionControls.prototype.getGibsonSpread = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setGibsonSpread = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional int32 minimum_placement = 10;
 * @return {number}
 */
proto.liwords.DivisionControls.prototype.getMinimumPlacement = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.DivisionControls} returns this
 */
proto.liwords.DivisionControls.prototype.setMinimumPlacement = function(value) {
  return jspb.Message.setProto3IntField(this, 10, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.TournamentGame.repeatedFields_ = [1,2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentGame.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentGame.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentGame} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGame.toObject = function(includeInstance, msg) {
  var f, obj = {
    scoresList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    resultsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f,
    gameEndReason: jspb.Message.getFieldWithDefault(msg, 3, 0),
    id: jspb.Message.getFieldWithDefault(msg, 4, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentGame}
 */
proto.liwords.TournamentGame.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentGame;
  return proto.liwords.TournamentGame.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentGame} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentGame}
 */
proto.liwords.TournamentGame.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt32() : [reader.readInt32()]);
      for (var i = 0; i < values.length; i++) {
        msg.addScores(values[i]);
      }
      break;
    case 2:
      var values = /** @type {!Array<!proto.liwords.TournamentGameResult>} */ (reader.isDelimited() ? reader.readPackedEnum() : [reader.readEnum()]);
      for (var i = 0; i < values.length; i++) {
        msg.addResults(values[i]);
      }
      break;
    case 3:
      var value = /** @type {!proto.liwords.GameEndReason} */ (reader.readEnum());
      msg.setGameEndReason(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentGame.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentGame.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentGame} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentGame.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getScoresList();
  if (f.length > 0) {
    writer.writePackedInt32(
      1,
      f
    );
  }
  f = message.getResultsList();
  if (f.length > 0) {
    writer.writePackedEnum(
      2,
      f
    );
  }
  f = message.getGameEndReason();
  if (f !== 0.0) {
    writer.writeEnum(
      3,
      f
    );
  }
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
};


/**
 * repeated int32 scores = 1;
 * @return {!Array<number>}
 */
proto.liwords.TournamentGame.prototype.getScoresList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.setScoresList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.addScores = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.clearScoresList = function() {
  return this.setScoresList([]);
};


/**
 * repeated TournamentGameResult results = 2;
 * @return {!Array<!proto.liwords.TournamentGameResult>}
 */
proto.liwords.TournamentGame.prototype.getResultsList = function() {
  return /** @type {!Array<!proto.liwords.TournamentGameResult>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<!proto.liwords.TournamentGameResult>} value
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.setResultsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {!proto.liwords.TournamentGameResult} value
 * @param {number=} opt_index
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.addResults = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.clearResultsList = function() {
  return this.setResultsList([]);
};


/**
 * optional GameEndReason game_end_reason = 3;
 * @return {!proto.liwords.GameEndReason}
 */
proto.liwords.TournamentGame.prototype.getGameEndReason = function() {
  return /** @type {!proto.liwords.GameEndReason} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {!proto.liwords.GameEndReason} value
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.setGameEndReason = function(value) {
  return jspb.Message.setProto3EnumField(this, 3, value);
};


/**
 * optional string id = 4;
 * @return {string}
 */
proto.liwords.TournamentGame.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentGame} returns this
 */
proto.liwords.TournamentGame.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.Pairing.repeatedFields_ = [1,3,4,5];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.Pairing.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.Pairing.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.Pairing} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.Pairing.toObject = function(includeInstance, msg) {
  var f, obj = {
    playersList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    round: jspb.Message.getFieldWithDefault(msg, 2, 0),
    gamesList: jspb.Message.toObjectList(msg.getGamesList(),
    proto.liwords.TournamentGame.toObject, includeInstance),
    outcomesList: (f = jspb.Message.getRepeatedField(msg, 4)) == null ? undefined : f,
    readyStatesList: (f = jspb.Message.getRepeatedField(msg, 5)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.Pairing}
 */
proto.liwords.Pairing.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.Pairing;
  return proto.liwords.Pairing.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.Pairing} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.Pairing}
 */
proto.liwords.Pairing.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var values = /** @type {!Array<number>} */ (reader.isDelimited() ? reader.readPackedInt32() : [reader.readInt32()]);
      for (var i = 0; i < values.length; i++) {
        msg.addPlayers(values[i]);
      }
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    case 3:
      var value = new proto.liwords.TournamentGame;
      reader.readMessage(value,proto.liwords.TournamentGame.deserializeBinaryFromReader);
      msg.addGames(value);
      break;
    case 4:
      var values = /** @type {!Array<!proto.liwords.TournamentGameResult>} */ (reader.isDelimited() ? reader.readPackedEnum() : [reader.readEnum()]);
      for (var i = 0; i < values.length; i++) {
        msg.addOutcomes(values[i]);
      }
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.addReadyStates(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.Pairing.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.Pairing.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.Pairing} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.Pairing.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPlayersList();
  if (f.length > 0) {
    writer.writePackedInt32(
      1,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getGamesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.liwords.TournamentGame.serializeBinaryToWriter
    );
  }
  f = message.getOutcomesList();
  if (f.length > 0) {
    writer.writePackedEnum(
      4,
      f
    );
  }
  f = message.getReadyStatesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      5,
      f
    );
  }
};


/**
 * repeated int32 players = 1;
 * @return {!Array<number>}
 */
proto.liwords.Pairing.prototype.getPlayersList = function() {
  return /** @type {!Array<number>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<number>} value
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.setPlayersList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {number} value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.addPlayers = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.clearPlayersList = function() {
  return this.setPlayersList([]);
};


/**
 * optional int32 round = 2;
 * @return {number}
 */
proto.liwords.Pairing.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * repeated TournamentGame games = 3;
 * @return {!Array<!proto.liwords.TournamentGame>}
 */
proto.liwords.Pairing.prototype.getGamesList = function() {
  return /** @type{!Array<!proto.liwords.TournamentGame>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.TournamentGame, 3));
};


/**
 * @param {!Array<!proto.liwords.TournamentGame>} value
 * @return {!proto.liwords.Pairing} returns this
*/
proto.liwords.Pairing.prototype.setGamesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.liwords.TournamentGame=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.TournamentGame}
 */
proto.liwords.Pairing.prototype.addGames = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.liwords.TournamentGame, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.clearGamesList = function() {
  return this.setGamesList([]);
};


/**
 * repeated TournamentGameResult outcomes = 4;
 * @return {!Array<!proto.liwords.TournamentGameResult>}
 */
proto.liwords.Pairing.prototype.getOutcomesList = function() {
  return /** @type {!Array<!proto.liwords.TournamentGameResult>} */ (jspb.Message.getRepeatedField(this, 4));
};


/**
 * @param {!Array<!proto.liwords.TournamentGameResult>} value
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.setOutcomesList = function(value) {
  return jspb.Message.setField(this, 4, value || []);
};


/**
 * @param {!proto.liwords.TournamentGameResult} value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.addOutcomes = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 4, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.clearOutcomesList = function() {
  return this.setOutcomesList([]);
};


/**
 * repeated string ready_states = 5;
 * @return {!Array<string>}
 */
proto.liwords.Pairing.prototype.getReadyStatesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 5));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.setReadyStatesList = function(value) {
  return jspb.Message.setField(this, 5, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.addReadyStates = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 5, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.Pairing} returns this
 */
proto.liwords.Pairing.prototype.clearReadyStatesList = function() {
  return this.setReadyStatesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.PlayerStanding.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.PlayerStanding.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.PlayerStanding} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PlayerStanding.toObject = function(includeInstance, msg) {
  var f, obj = {
    playerId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    wins: jspb.Message.getFieldWithDefault(msg, 2, 0),
    losses: jspb.Message.getFieldWithDefault(msg, 3, 0),
    draws: jspb.Message.getFieldWithDefault(msg, 4, 0),
    spread: jspb.Message.getFieldWithDefault(msg, 5, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.PlayerStanding}
 */
proto.liwords.PlayerStanding.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.PlayerStanding;
  return proto.liwords.PlayerStanding.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.PlayerStanding} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.PlayerStanding}
 */
proto.liwords.PlayerStanding.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setPlayerId(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setWins(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setLosses(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setDraws(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setSpread(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.PlayerStanding.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.PlayerStanding.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.PlayerStanding} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PlayerStanding.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPlayerId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getWins();
  if (f !== 0) {
    writer.writeInt32(
      2,
      f
    );
  }
  f = message.getLosses();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getDraws();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getSpread();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
};


/**
 * optional string player_id = 1;
 * @return {string}
 */
proto.liwords.PlayerStanding.prototype.getPlayerId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.PlayerStanding} returns this
 */
proto.liwords.PlayerStanding.prototype.setPlayerId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional int32 wins = 2;
 * @return {number}
 */
proto.liwords.PlayerStanding.prototype.getWins = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.PlayerStanding} returns this
 */
proto.liwords.PlayerStanding.prototype.setWins = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional int32 losses = 3;
 * @return {number}
 */
proto.liwords.PlayerStanding.prototype.getLosses = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.PlayerStanding} returns this
 */
proto.liwords.PlayerStanding.prototype.setLosses = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 draws = 4;
 * @return {number}
 */
proto.liwords.PlayerStanding.prototype.getDraws = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.PlayerStanding} returns this
 */
proto.liwords.PlayerStanding.prototype.setDraws = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional int32 spread = 5;
 * @return {number}
 */
proto.liwords.PlayerStanding.prototype.getSpread = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.PlayerStanding} returns this
 */
proto.liwords.PlayerStanding.prototype.setSpread = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.RoundStandings.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.RoundStandings.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.RoundStandings.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.RoundStandings} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RoundStandings.toObject = function(includeInstance, msg) {
  var f, obj = {
    standingsList: jspb.Message.toObjectList(msg.getStandingsList(),
    proto.liwords.PlayerStanding.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.RoundStandings}
 */
proto.liwords.RoundStandings.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.RoundStandings;
  return proto.liwords.RoundStandings.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.RoundStandings} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.RoundStandings}
 */
proto.liwords.RoundStandings.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.liwords.PlayerStanding;
      reader.readMessage(value,proto.liwords.PlayerStanding.deserializeBinaryFromReader);
      msg.addStandings(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.RoundStandings.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.RoundStandings.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.RoundStandings} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.RoundStandings.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getStandingsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.liwords.PlayerStanding.serializeBinaryToWriter
    );
  }
};


/**
 * repeated PlayerStanding standings = 1;
 * @return {!Array<!proto.liwords.PlayerStanding>}
 */
proto.liwords.RoundStandings.prototype.getStandingsList = function() {
  return /** @type{!Array<!proto.liwords.PlayerStanding>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.PlayerStanding, 1));
};


/**
 * @param {!Array<!proto.liwords.PlayerStanding>} value
 * @return {!proto.liwords.RoundStandings} returns this
*/
proto.liwords.RoundStandings.prototype.setStandingsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.liwords.PlayerStanding=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.PlayerStanding}
 */
proto.liwords.RoundStandings.prototype.addStandings = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.liwords.PlayerStanding, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.RoundStandings} returns this
 */
proto.liwords.RoundStandings.prototype.clearStandingsList = function() {
  return this.setStandingsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.DivisionPairingsResponse.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DivisionPairingsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DivisionPairingsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DivisionPairingsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionPairingsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    divisionPairingsList: jspb.Message.toObjectList(msg.getDivisionPairingsList(),
    proto.liwords.Pairing.toObject, includeInstance),
    divisionStandingsMap: (f = msg.getDivisionStandingsMap()) ? f.toObject(includeInstance, proto.liwords.RoundStandings.toObject) : [],
    gibsonizedPlayersMap: (f = msg.getGibsonizedPlayersMap()) ? f.toObject(includeInstance, undefined) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DivisionPairingsResponse}
 */
proto.liwords.DivisionPairingsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DivisionPairingsResponse;
  return proto.liwords.DivisionPairingsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DivisionPairingsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DivisionPairingsResponse}
 */
proto.liwords.DivisionPairingsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.Pairing;
      reader.readMessage(value,proto.liwords.Pairing.deserializeBinaryFromReader);
      msg.addDivisionPairings(value);
      break;
    case 4:
      var value = msg.getDivisionStandingsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readInt32, jspb.BinaryReader.prototype.readMessage, proto.liwords.RoundStandings.deserializeBinaryFromReader, 0, new proto.liwords.RoundStandings());
         });
      break;
    case 5:
      var value = msg.getGibsonizedPlayersMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readInt32, null, "", 0);
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DivisionPairingsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DivisionPairingsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DivisionPairingsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionPairingsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDivisionPairingsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.liwords.Pairing.serializeBinaryToWriter
    );
  }
  f = message.getDivisionStandingsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeInt32, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.RoundStandings.serializeBinaryToWriter);
  }
  f = message.getGibsonizedPlayersMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeInt32);
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.DivisionPairingsResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
 */
proto.liwords.DivisionPairingsResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.DivisionPairingsResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
 */
proto.liwords.DivisionPairingsResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated Pairing division_pairings = 3;
 * @return {!Array<!proto.liwords.Pairing>}
 */
proto.liwords.DivisionPairingsResponse.prototype.getDivisionPairingsList = function() {
  return /** @type{!Array<!proto.liwords.Pairing>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.Pairing, 3));
};


/**
 * @param {!Array<!proto.liwords.Pairing>} value
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
*/
proto.liwords.DivisionPairingsResponse.prototype.setDivisionPairingsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.liwords.Pairing=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing}
 */
proto.liwords.DivisionPairingsResponse.prototype.addDivisionPairings = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.liwords.Pairing, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
 */
proto.liwords.DivisionPairingsResponse.prototype.clearDivisionPairingsList = function() {
  return this.setDivisionPairingsList([]);
};


/**
 * map<int32, RoundStandings> division_standings = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<number,!proto.liwords.RoundStandings>}
 */
proto.liwords.DivisionPairingsResponse.prototype.getDivisionStandingsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<number,!proto.liwords.RoundStandings>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.liwords.RoundStandings));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
 */
proto.liwords.DivisionPairingsResponse.prototype.clearDivisionStandingsMap = function() {
  this.getDivisionStandingsMap().clear();
  return this;};


/**
 * map<string, int32> gibsonized_players = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,number>}
 */
proto.liwords.DivisionPairingsResponse.prototype.getGibsonizedPlayersMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,number>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      null));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.DivisionPairingsResponse} returns this
 */
proto.liwords.DivisionPairingsResponse.prototype.clearGibsonizedPlayersMap = function() {
  this.getGibsonizedPlayersMap().clear();
  return this;};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DivisionPairingsDeletedResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DivisionPairingsDeletedResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionPairingsDeletedResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    round: jspb.Message.getFieldWithDefault(msg, 3, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DivisionPairingsDeletedResponse}
 */
proto.liwords.DivisionPairingsDeletedResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DivisionPairingsDeletedResponse;
  return proto.liwords.DivisionPairingsDeletedResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DivisionPairingsDeletedResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DivisionPairingsDeletedResponse}
 */
proto.liwords.DivisionPairingsDeletedResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRound(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DivisionPairingsDeletedResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DivisionPairingsDeletedResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionPairingsDeletedResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRound();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionPairingsDeletedResponse} returns this
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionPairingsDeletedResponse} returns this
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int32 round = 3;
 * @return {number}
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.getRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.DivisionPairingsDeletedResponse} returns this
 */
proto.liwords.DivisionPairingsDeletedResponse.prototype.setRound = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.PlayersAddedOrRemovedResponse.repeatedFields_ = [4];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.PlayersAddedOrRemovedResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.PlayersAddedOrRemovedResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PlayersAddedOrRemovedResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    players: (f = msg.getPlayers()) && proto.liwords.TournamentPersons.toObject(includeInstance, f),
    divisionPairingsList: jspb.Message.toObjectList(msg.getDivisionPairingsList(),
    proto.liwords.Pairing.toObject, includeInstance),
    divisionStandingsMap: (f = msg.getDivisionStandingsMap()) ? f.toObject(includeInstance, proto.liwords.RoundStandings.toObject) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse}
 */
proto.liwords.PlayersAddedOrRemovedResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.PlayersAddedOrRemovedResponse;
  return proto.liwords.PlayersAddedOrRemovedResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.PlayersAddedOrRemovedResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse}
 */
proto.liwords.PlayersAddedOrRemovedResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.TournamentPersons;
      reader.readMessage(value,proto.liwords.TournamentPersons.deserializeBinaryFromReader);
      msg.setPlayers(value);
      break;
    case 4:
      var value = new proto.liwords.Pairing;
      reader.readMessage(value,proto.liwords.Pairing.deserializeBinaryFromReader);
      msg.addDivisionPairings(value);
      break;
    case 5:
      var value = msg.getDivisionStandingsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readInt32, jspb.BinaryReader.prototype.readMessage, proto.liwords.RoundStandings.deserializeBinaryFromReader, 0, new proto.liwords.RoundStandings());
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.PlayersAddedOrRemovedResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.PlayersAddedOrRemovedResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.PlayersAddedOrRemovedResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPlayers();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.liwords.TournamentPersons.serializeBinaryToWriter
    );
  }
  f = message.getDivisionPairingsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      4,
      f,
      proto.liwords.Pairing.serializeBinaryToWriter
    );
  }
  f = message.getDivisionStandingsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeInt32, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.RoundStandings.serializeBinaryToWriter);
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional TournamentPersons players = 3;
 * @return {?proto.liwords.TournamentPersons}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.getPlayers = function() {
  return /** @type{?proto.liwords.TournamentPersons} */ (
    jspb.Message.getWrapperField(this, proto.liwords.TournamentPersons, 3));
};


/**
 * @param {?proto.liwords.TournamentPersons|undefined} value
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
*/
proto.liwords.PlayersAddedOrRemovedResponse.prototype.setPlayers = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.clearPlayers = function() {
  return this.setPlayers(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.hasPlayers = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * repeated Pairing division_pairings = 4;
 * @return {!Array<!proto.liwords.Pairing>}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.getDivisionPairingsList = function() {
  return /** @type{!Array<!proto.liwords.Pairing>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.Pairing, 4));
};


/**
 * @param {!Array<!proto.liwords.Pairing>} value
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
*/
proto.liwords.PlayersAddedOrRemovedResponse.prototype.setDivisionPairingsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 4, value);
};


/**
 * @param {!proto.liwords.Pairing=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.addDivisionPairings = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 4, opt_value, proto.liwords.Pairing, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.clearDivisionPairingsList = function() {
  return this.setDivisionPairingsList([]);
};


/**
 * map<int32, RoundStandings> division_standings = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<number,!proto.liwords.RoundStandings>}
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.getDivisionStandingsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<number,!proto.liwords.RoundStandings>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.liwords.RoundStandings));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.PlayersAddedOrRemovedResponse} returns this
 */
proto.liwords.PlayersAddedOrRemovedResponse.prototype.clearDivisionStandingsMap = function() {
  this.getDivisionStandingsMap().clear();
  return this;};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.DivisionRoundControls.repeatedFields_ = [3,4];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DivisionRoundControls.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DivisionRoundControls.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DivisionRoundControls} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionRoundControls.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    roundControlsList: jspb.Message.toObjectList(msg.getRoundControlsList(),
    proto.liwords.RoundControl.toObject, includeInstance),
    divisionPairingsList: jspb.Message.toObjectList(msg.getDivisionPairingsList(),
    proto.liwords.Pairing.toObject, includeInstance),
    divisionStandingsMap: (f = msg.getDivisionStandingsMap()) ? f.toObject(includeInstance, proto.liwords.RoundStandings.toObject) : []
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DivisionRoundControls}
 */
proto.liwords.DivisionRoundControls.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DivisionRoundControls;
  return proto.liwords.DivisionRoundControls.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DivisionRoundControls} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DivisionRoundControls}
 */
proto.liwords.DivisionRoundControls.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.RoundControl;
      reader.readMessage(value,proto.liwords.RoundControl.deserializeBinaryFromReader);
      msg.addRoundControls(value);
      break;
    case 4:
      var value = new proto.liwords.Pairing;
      reader.readMessage(value,proto.liwords.Pairing.deserializeBinaryFromReader);
      msg.addDivisionPairings(value);
      break;
    case 5:
      var value = msg.getDivisionStandingsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readInt32, jspb.BinaryReader.prototype.readMessage, proto.liwords.RoundStandings.deserializeBinaryFromReader, 0, new proto.liwords.RoundStandings());
         });
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DivisionRoundControls.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DivisionRoundControls.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DivisionRoundControls} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionRoundControls.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getRoundControlsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.liwords.RoundControl.serializeBinaryToWriter
    );
  }
  f = message.getDivisionPairingsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      4,
      f,
      proto.liwords.Pairing.serializeBinaryToWriter
    );
  }
  f = message.getDivisionStandingsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeInt32, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.RoundStandings.serializeBinaryToWriter);
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.DivisionRoundControls.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionRoundControls} returns this
 */
proto.liwords.DivisionRoundControls.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.DivisionRoundControls.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionRoundControls} returns this
 */
proto.liwords.DivisionRoundControls.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated RoundControl round_controls = 3;
 * @return {!Array<!proto.liwords.RoundControl>}
 */
proto.liwords.DivisionRoundControls.prototype.getRoundControlsList = function() {
  return /** @type{!Array<!proto.liwords.RoundControl>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.RoundControl, 3));
};


/**
 * @param {!Array<!proto.liwords.RoundControl>} value
 * @return {!proto.liwords.DivisionRoundControls} returns this
*/
proto.liwords.DivisionRoundControls.prototype.setRoundControlsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.liwords.RoundControl=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.RoundControl}
 */
proto.liwords.DivisionRoundControls.prototype.addRoundControls = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.liwords.RoundControl, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.DivisionRoundControls} returns this
 */
proto.liwords.DivisionRoundControls.prototype.clearRoundControlsList = function() {
  return this.setRoundControlsList([]);
};


/**
 * repeated Pairing division_pairings = 4;
 * @return {!Array<!proto.liwords.Pairing>}
 */
proto.liwords.DivisionRoundControls.prototype.getDivisionPairingsList = function() {
  return /** @type{!Array<!proto.liwords.Pairing>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.Pairing, 4));
};


/**
 * @param {!Array<!proto.liwords.Pairing>} value
 * @return {!proto.liwords.DivisionRoundControls} returns this
*/
proto.liwords.DivisionRoundControls.prototype.setDivisionPairingsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 4, value);
};


/**
 * @param {!proto.liwords.Pairing=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.Pairing}
 */
proto.liwords.DivisionRoundControls.prototype.addDivisionPairings = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 4, opt_value, proto.liwords.Pairing, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.DivisionRoundControls} returns this
 */
proto.liwords.DivisionRoundControls.prototype.clearDivisionPairingsList = function() {
  return this.setDivisionPairingsList([]);
};


/**
 * map<int32, RoundStandings> division_standings = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<number,!proto.liwords.RoundStandings>}
 */
proto.liwords.DivisionRoundControls.prototype.getDivisionStandingsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<number,!proto.liwords.RoundStandings>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.liwords.RoundStandings));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.DivisionRoundControls} returns this
 */
proto.liwords.DivisionRoundControls.prototype.clearDivisionStandingsMap = function() {
  this.getDivisionStandingsMap().clear();
  return this;};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.DivisionControlsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.DivisionControlsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.DivisionControlsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionControlsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    divisionControls: (f = msg.getDivisionControls()) && proto.liwords.DivisionControls.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.DivisionControlsResponse}
 */
proto.liwords.DivisionControlsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.DivisionControlsResponse;
  return proto.liwords.DivisionControlsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.DivisionControlsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.DivisionControlsResponse}
 */
proto.liwords.DivisionControlsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.DivisionControls;
      reader.readMessage(value,proto.liwords.DivisionControls.deserializeBinaryFromReader);
      msg.setDivisionControls(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.DivisionControlsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.DivisionControlsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.DivisionControlsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.DivisionControlsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDivisionControls();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.liwords.DivisionControls.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.DivisionControlsResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionControlsResponse} returns this
 */
proto.liwords.DivisionControlsResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.DivisionControlsResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.DivisionControlsResponse} returns this
 */
proto.liwords.DivisionControlsResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional DivisionControls division_controls = 3;
 * @return {?proto.liwords.DivisionControls}
 */
proto.liwords.DivisionControlsResponse.prototype.getDivisionControls = function() {
  return /** @type{?proto.liwords.DivisionControls} */ (
    jspb.Message.getWrapperField(this, proto.liwords.DivisionControls, 3));
};


/**
 * @param {?proto.liwords.DivisionControls|undefined} value
 * @return {!proto.liwords.DivisionControlsResponse} returns this
*/
proto.liwords.DivisionControlsResponse.prototype.setDivisionControls = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.DivisionControlsResponse} returns this
 */
proto.liwords.DivisionControlsResponse.prototype.clearDivisionControls = function() {
  return this.setDivisionControls(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.DivisionControlsResponse.prototype.hasDivisionControls = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.liwords.TournamentDivisionDataResponse.repeatedFields_ = [7];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentDivisionDataResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentDivisionDataResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDivisionDataResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, ""),
    players: (f = msg.getPlayers()) && proto.liwords.TournamentPersons.toObject(includeInstance, f),
    standingsMap: (f = msg.getStandingsMap()) ? f.toObject(includeInstance, proto.liwords.RoundStandings.toObject) : [],
    pairingMapMap: (f = msg.getPairingMapMap()) ? f.toObject(includeInstance, proto.liwords.Pairing.toObject) : [],
    controls: (f = msg.getControls()) && proto.liwords.DivisionControls.toObject(includeInstance, f),
    roundControlsList: jspb.Message.toObjectList(msg.getRoundControlsList(),
    proto.liwords.RoundControl.toObject, includeInstance),
    currentRound: jspb.Message.getFieldWithDefault(msg, 8, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentDivisionDataResponse}
 */
proto.liwords.TournamentDivisionDataResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentDivisionDataResponse;
  return proto.liwords.TournamentDivisionDataResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentDivisionDataResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentDivisionDataResponse}
 */
proto.liwords.TournamentDivisionDataResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    case 3:
      var value = new proto.liwords.TournamentPersons;
      reader.readMessage(value,proto.liwords.TournamentPersons.deserializeBinaryFromReader);
      msg.setPlayers(value);
      break;
    case 4:
      var value = msg.getStandingsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readInt32, jspb.BinaryReader.prototype.readMessage, proto.liwords.RoundStandings.deserializeBinaryFromReader, 0, new proto.liwords.RoundStandings());
         });
      break;
    case 5:
      var value = msg.getPairingMapMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.liwords.Pairing.deserializeBinaryFromReader, "", new proto.liwords.Pairing());
         });
      break;
    case 6:
      var value = new proto.liwords.DivisionControls;
      reader.readMessage(value,proto.liwords.DivisionControls.deserializeBinaryFromReader);
      msg.setControls(value);
      break;
    case 7:
      var value = new proto.liwords.RoundControl;
      reader.readMessage(value,proto.liwords.RoundControl.deserializeBinaryFromReader);
      msg.addRoundControls(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setCurrentRound(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentDivisionDataResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentDivisionDataResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDivisionDataResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getPlayers();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.liwords.TournamentPersons.serializeBinaryToWriter
    );
  }
  f = message.getStandingsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(4, writer, jspb.BinaryWriter.prototype.writeInt32, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.RoundStandings.serializeBinaryToWriter);
  }
  f = message.getPairingMapMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(5, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.Pairing.serializeBinaryToWriter);
  }
  f = message.getControls();
  if (f != null) {
    writer.writeMessage(
      6,
      f,
      proto.liwords.DivisionControls.serializeBinaryToWriter
    );
  }
  f = message.getRoundControlsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      7,
      f,
      proto.liwords.RoundControl.serializeBinaryToWriter
    );
  }
  f = message.getCurrentRound();
  if (f !== 0) {
    writer.writeInt32(
      8,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional TournamentPersons players = 3;
 * @return {?proto.liwords.TournamentPersons}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getPlayers = function() {
  return /** @type{?proto.liwords.TournamentPersons} */ (
    jspb.Message.getWrapperField(this, proto.liwords.TournamentPersons, 3));
};


/**
 * @param {?proto.liwords.TournamentPersons|undefined} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
*/
proto.liwords.TournamentDivisionDataResponse.prototype.setPlayers = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.clearPlayers = function() {
  return this.setPlayers(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.hasPlayers = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * map<int32, RoundStandings> standings = 4;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<number,!proto.liwords.RoundStandings>}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getStandingsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<number,!proto.liwords.RoundStandings>} */ (
      jspb.Message.getMapField(this, 4, opt_noLazyCreate,
      proto.liwords.RoundStandings));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.clearStandingsMap = function() {
  this.getStandingsMap().clear();
  return this;};


/**
 * map<string, Pairing> pairing_map = 5;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.liwords.Pairing>}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getPairingMapMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.liwords.Pairing>} */ (
      jspb.Message.getMapField(this, 5, opt_noLazyCreate,
      proto.liwords.Pairing));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.clearPairingMapMap = function() {
  this.getPairingMapMap().clear();
  return this;};


/**
 * optional DivisionControls controls = 6;
 * @return {?proto.liwords.DivisionControls}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getControls = function() {
  return /** @type{?proto.liwords.DivisionControls} */ (
    jspb.Message.getWrapperField(this, proto.liwords.DivisionControls, 6));
};


/**
 * @param {?proto.liwords.DivisionControls|undefined} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
*/
proto.liwords.TournamentDivisionDataResponse.prototype.setControls = function(value) {
  return jspb.Message.setWrapperField(this, 6, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.clearControls = function() {
  return this.setControls(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.hasControls = function() {
  return jspb.Message.getField(this, 6) != null;
};


/**
 * repeated RoundControl round_controls = 7;
 * @return {!Array<!proto.liwords.RoundControl>}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getRoundControlsList = function() {
  return /** @type{!Array<!proto.liwords.RoundControl>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.liwords.RoundControl, 7));
};


/**
 * @param {!Array<!proto.liwords.RoundControl>} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
*/
proto.liwords.TournamentDivisionDataResponse.prototype.setRoundControlsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 7, value);
};


/**
 * @param {!proto.liwords.RoundControl=} opt_value
 * @param {number=} opt_index
 * @return {!proto.liwords.RoundControl}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.addRoundControls = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 7, opt_value, proto.liwords.RoundControl, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.clearRoundControlsList = function() {
  return this.setRoundControlsList([]);
};


/**
 * optional int32 current_round = 8;
 * @return {number}
 */
proto.liwords.TournamentDivisionDataResponse.prototype.getCurrentRound = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.liwords.TournamentDivisionDataResponse} returns this
 */
proto.liwords.TournamentDivisionDataResponse.prototype.setCurrentRound = function(value) {
  return jspb.Message.setProto3IntField(this, 8, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.FullTournamentDivisions.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.FullTournamentDivisions.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.FullTournamentDivisions} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.FullTournamentDivisions.toObject = function(includeInstance, msg) {
  var f, obj = {
    divisionsMap: (f = msg.getDivisionsMap()) ? f.toObject(includeInstance, proto.liwords.TournamentDivisionDataResponse.toObject) : [],
    started: jspb.Message.getBooleanFieldWithDefault(msg, 2, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.FullTournamentDivisions}
 */
proto.liwords.FullTournamentDivisions.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.FullTournamentDivisions;
  return proto.liwords.FullTournamentDivisions.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.FullTournamentDivisions} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.FullTournamentDivisions}
 */
proto.liwords.FullTournamentDivisions.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = msg.getDivisionsMap();
      reader.readMessage(value, function(message, reader) {
        jspb.Map.deserializeBinary(message, reader, jspb.BinaryReader.prototype.readString, jspb.BinaryReader.prototype.readMessage, proto.liwords.TournamentDivisionDataResponse.deserializeBinaryFromReader, "", new proto.liwords.TournamentDivisionDataResponse());
         });
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setStarted(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.FullTournamentDivisions.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.FullTournamentDivisions.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.FullTournamentDivisions} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.FullTournamentDivisions.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getDivisionsMap(true);
  if (f && f.getLength() > 0) {
    f.serializeBinary(1, writer, jspb.BinaryWriter.prototype.writeString, jspb.BinaryWriter.prototype.writeMessage, proto.liwords.TournamentDivisionDataResponse.serializeBinaryToWriter);
  }
  f = message.getStarted();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * map<string, TournamentDivisionDataResponse> divisions = 1;
 * @param {boolean=} opt_noLazyCreate Do not create the map if
 * empty, instead returning `undefined`
 * @return {!jspb.Map<string,!proto.liwords.TournamentDivisionDataResponse>}
 */
proto.liwords.FullTournamentDivisions.prototype.getDivisionsMap = function(opt_noLazyCreate) {
  return /** @type {!jspb.Map<string,!proto.liwords.TournamentDivisionDataResponse>} */ (
      jspb.Message.getMapField(this, 1, opt_noLazyCreate,
      proto.liwords.TournamentDivisionDataResponse));
};


/**
 * Clears values from the map. The map will be non-null.
 * @return {!proto.liwords.FullTournamentDivisions} returns this
 */
proto.liwords.FullTournamentDivisions.prototype.clearDivisionsMap = function() {
  this.getDivisionsMap().clear();
  return this;};


/**
 * optional bool started = 2;
 * @return {boolean}
 */
proto.liwords.FullTournamentDivisions.prototype.getStarted = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.FullTournamentDivisions} returns this
 */
proto.liwords.FullTournamentDivisions.prototype.setStarted = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentFinishedResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentFinishedResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentFinishedResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentFinishedResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentFinishedResponse}
 */
proto.liwords.TournamentFinishedResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentFinishedResponse;
  return proto.liwords.TournamentFinishedResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentFinishedResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentFinishedResponse}
 */
proto.liwords.TournamentFinishedResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentFinishedResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentFinishedResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentFinishedResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentFinishedResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentFinishedResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentFinishedResponse} returns this
 */
proto.liwords.TournamentFinishedResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentDataResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentDataResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentDataResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDataResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    description: jspb.Message.getFieldWithDefault(msg, 3, ""),
    executiveDirector: jspb.Message.getFieldWithDefault(msg, 4, ""),
    directors: (f = msg.getDirectors()) && proto.liwords.TournamentPersons.toObject(includeInstance, f),
    isStarted: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    startTime: (f = msg.getStartTime()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentDataResponse}
 */
proto.liwords.TournamentDataResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentDataResponse;
  return proto.liwords.TournamentDataResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentDataResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentDataResponse}
 */
proto.liwords.TournamentDataResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setDescription(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setExecutiveDirector(value);
      break;
    case 5:
      var value = new proto.liwords.TournamentPersons;
      reader.readMessage(value,proto.liwords.TournamentPersons.deserializeBinaryFromReader);
      msg.setDirectors(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsStarted(value);
      break;
    case 7:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setStartTime(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentDataResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentDataResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentDataResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDataResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getDescription();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getExecutiveDirector();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getDirectors();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      proto.liwords.TournamentPersons.serializeBinaryToWriter
    );
  }
  f = message.getIsStarted();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getStartTime();
  if (f != null) {
    writer.writeMessage(
      7,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentDataResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.liwords.TournamentDataResponse.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string description = 3;
 * @return {string}
 */
proto.liwords.TournamentDataResponse.prototype.getDescription = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.setDescription = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string executive_director = 4;
 * @return {string}
 */
proto.liwords.TournamentDataResponse.prototype.getExecutiveDirector = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.setExecutiveDirector = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional TournamentPersons directors = 5;
 * @return {?proto.liwords.TournamentPersons}
 */
proto.liwords.TournamentDataResponse.prototype.getDirectors = function() {
  return /** @type{?proto.liwords.TournamentPersons} */ (
    jspb.Message.getWrapperField(this, proto.liwords.TournamentPersons, 5));
};


/**
 * @param {?proto.liwords.TournamentPersons|undefined} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
*/
proto.liwords.TournamentDataResponse.prototype.setDirectors = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.clearDirectors = function() {
  return this.setDirectors(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.TournamentDataResponse.prototype.hasDirectors = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool is_started = 6;
 * @return {boolean}
 */
proto.liwords.TournamentDataResponse.prototype.getIsStarted = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.setIsStarted = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional google.protobuf.Timestamp start_time = 7;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.liwords.TournamentDataResponse.prototype.getStartTime = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 7));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.liwords.TournamentDataResponse} returns this
*/
proto.liwords.TournamentDataResponse.prototype.setStartTime = function(value) {
  return jspb.Message.setWrapperField(this, 7, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.liwords.TournamentDataResponse} returns this
 */
proto.liwords.TournamentDataResponse.prototype.clearStartTime = function() {
  return this.setStartTime(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.liwords.TournamentDataResponse.prototype.hasStartTime = function() {
  return jspb.Message.getField(this, 7) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.TournamentDivisionDeletedResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.TournamentDivisionDeletedResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDivisionDeletedResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    division: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.TournamentDivisionDeletedResponse}
 */
proto.liwords.TournamentDivisionDeletedResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.TournamentDivisionDeletedResponse;
  return proto.liwords.TournamentDivisionDeletedResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.TournamentDivisionDeletedResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.TournamentDivisionDeletedResponse}
 */
proto.liwords.TournamentDivisionDeletedResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setDivision(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.TournamentDivisionDeletedResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.TournamentDivisionDeletedResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.TournamentDivisionDeletedResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getDivision();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDivisionDeletedResponse} returns this
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string division = 2;
 * @return {string}
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.getDivision = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.TournamentDivisionDeletedResponse} returns this
 */
proto.liwords.TournamentDivisionDeletedResponse.prototype.setDivision = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.JoinPath.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.JoinPath.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.JoinPath} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.JoinPath.toObject = function(includeInstance, msg) {
  var f, obj = {
    path: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.JoinPath}
 */
proto.liwords.JoinPath.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.JoinPath;
  return proto.liwords.JoinPath.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.JoinPath} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.JoinPath}
 */
proto.liwords.JoinPath.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setPath(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.JoinPath.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.JoinPath.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.JoinPath} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.JoinPath.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPath();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string path = 1;
 * @return {string}
 */
proto.liwords.JoinPath.prototype.getPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.liwords.JoinPath} returns this
 */
proto.liwords.JoinPath.prototype.setPath = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.liwords.UnjoinRealm.prototype.toObject = function(opt_includeInstance) {
  return proto.liwords.UnjoinRealm.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.liwords.UnjoinRealm} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UnjoinRealm.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.liwords.UnjoinRealm}
 */
proto.liwords.UnjoinRealm.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.liwords.UnjoinRealm;
  return proto.liwords.UnjoinRealm.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.liwords.UnjoinRealm} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.liwords.UnjoinRealm}
 */
proto.liwords.UnjoinRealm.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.liwords.UnjoinRealm.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.liwords.UnjoinRealm.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.liwords.UnjoinRealm} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.liwords.UnjoinRealm.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};


/**
 * @enum {number}
 */
proto.liwords.GameMode = {
  REAL_TIME: 0,
  CORRESPONDENCE: 1
};

/**
 * @enum {number}
 */
proto.liwords.RatingMode = {
  RATED: 0,
  CASUAL: 1
};

/**
 * @enum {number}
 */
proto.liwords.ChildStatus = {
  CHILD: 0,
  NOT_CHILD: 1,
  UNKNOWN: 2
};

/**
 * @enum {number}
 */
proto.liwords.MessageType = {
  SEEK_REQUEST: 0,
  MATCH_REQUEST: 1,
  SOUGHT_GAME_PROCESS_EVENT: 2,
  CLIENT_GAMEPLAY_EVENT: 3,
  SERVER_GAMEPLAY_EVENT: 4,
  GAME_ENDED_EVENT: 5,
  GAME_HISTORY_REFRESHER: 6,
  ERROR_MESSAGE: 7,
  NEW_GAME_EVENT: 8,
  SERVER_CHALLENGE_RESULT_EVENT: 9,
  SEEK_REQUESTS: 10,
  MATCH_REQUEST_CANCELLATION: 11,
  ONGOING_GAME_EVENT: 12,
  TIMED_OUT: 13,
  ONGOING_GAMES: 14,
  READY_FOR_TOURNAMENT_GAME: 15,
  TOURNAMENT_ROUND_STARTED: 16,
  GAME_DELETION: 17,
  MATCH_REQUESTS: 18,
  DECLINE_MATCH_REQUEST: 19,
  CHAT_MESSAGE: 20,
  CHAT_MESSAGE_DELETED: 21,
  USER_PRESENCE: 22,
  USER_PRESENCES: 23,
  SERVER_MESSAGE: 24,
  READY_FOR_GAME: 25,
  LAG_MEASUREMENT: 26,
  TOURNAMENT_GAME_ENDED_EVENT: 27,
  TOURNAMENT_MESSAGE: 28,
  REMATCH_STARTED: 29,
  TOURNAMENT_DIVISION_MESSAGE: 30,
  TOURNAMENT_DIVISION_DELETED_MESSAGE: 31,
  TOURNAMENT_FULL_DIVISIONS_MESSAGE: 32,
  TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE: 34,
  TOURNAMENT_DIVISION_PAIRINGS_MESSAGE: 35,
  TOURNAMENT_DIVISION_CONTROLS_MESSAGE: 36,
  TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE: 37,
  TOURNAMENT_FINISHED_MESSAGE: 38,
  TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE: 39,
  PRESENCE_ENTRY: 40,
  ACTIVE_GAME_ENTRY: 41,
  GAME_META_EVENT: 42
};

/**
 * @enum {number}
 */
proto.liwords.GameEndReason = {
  NONE: 0,
  TIME: 1,
  STANDARD: 2,
  CONSECUTIVE_ZEROES: 3,
  RESIGNED: 4,
  ABORTED: 5,
  TRIPLE_CHALLENGE: 6,
  CANCELLED: 7,
  FORCE_FORFEIT: 8
};

/**
 * @enum {number}
 */
proto.liwords.TournamentGameResult = {
  NO_RESULT: 0,
  WIN: 1,
  LOSS: 2,
  DRAW: 3,
  BYE: 4,
  FORFEIT_WIN: 5,
  FORFEIT_LOSS: 6,
  ELIMINATED: 7
};

/**
 * @enum {number}
 */
proto.liwords.PairingMethod = {
  RANDOM: 0,
  ROUND_ROBIN: 1,
  KING_OF_THE_HILL: 2,
  ELIMINATION: 3,
  FACTOR: 4,
  INITIAL_FONTES: 5,
  SWISS: 6,
  QUICKPAIR: 7,
  MANUAL: 8,
  TEAM_ROUND_ROBIN: 9
};

/**
 * @enum {number}
 */
proto.liwords.FirstMethod = {
  MANUAL_FIRST: 0,
  RANDOM_FIRST: 1,
  AUTOMATIC_FIRST: 2
};

goog.object.extend(exports, proto.liwords);
