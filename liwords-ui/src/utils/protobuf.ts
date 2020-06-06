/* eslint-disable no-bitwise */
// a serialization library

import {
  MessageType,
  MessageTypeMap,
  SeekRequest,
} from '../gen/api/proto/game_service_pb';

// SocketFmt is just the protobuf, with an extra byte prepended,
// indicating the message type.
export const encodeToSocketFmt = (
  msgTypeCode: number,
  serializedPBPacket: Uint8Array
): Uint8Array => {
  const packetLength = serializedPBPacket.length;
  const overallLength = packetLength + 1;

  const newArr = new Uint8Array(overallLength);
  newArr[0] = msgTypeCode;
  newArr.set(serializedPBPacket, 1);
  return newArr;
};

export const decodeToMsg = (
  data: Blob,
  onload: (reader: FileReader) => void
) => {
  console.log('in decodeToMsg');
  const reader = new FileReader();
  reader.onload = () => onload(reader);
  reader.readAsArrayBuffer(data);
};
