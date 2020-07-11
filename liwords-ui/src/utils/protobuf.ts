/* eslint-disable no-bitwise */
// a serialization library

// SocketFmt is just the protobuf, with an extra byte prepended,
// indicating the message type.
export const encodeToSocketFmt = (
  msgTypeCode: number,
  serializedPBPacket: Uint8Array
): Uint8Array => {
  // 1 byte for the msg type.
  const packetLength = serializedPBPacket.length + 1;
  // 2 bytes for the packetLength
  const overallLength = packetLength + 2;

  const newArr = new Uint8Array(overallLength);
  newArr[0] = Math.floor(packetLength / 256);
  newArr[1] = packetLength & 255;
  newArr[2] = msgTypeCode;

  newArr.set(serializedPBPacket, 3);
  return newArr;
};

export const decodeToMsg = (
  data: Blob,
  onload: (reader: FileReader) => void
) => {
  const reader = new FileReader();
  reader.onload = () => onload(reader);
  reader.readAsArrayBuffer(data);
};
