// a serialization library

// Protocol version: 1 = legacy (2-byte length), 2 = new (3-byte length)
let protocolVersion = 1;

export const setProtocolVersion = (v: number) => {
  protocolVersion = v;
};

export const getProtocolVersion = () => protocolVersion;

// SocketFmt is just the protobuf, with an extra byte prepended,
// indicating the message type.
// V1: 2-byte length prefix (max ~64KB)
// V2: 3-byte length prefix (max ~16MB)
export const encodeToSocketFmt = (
  msgTypeCode: number,
  serializedPBPacket: Uint8Array,
): Uint8Array => {
  // 1 byte for the msg type.
  const packetLength = serializedPBPacket.length + 1;

  if (protocolVersion === 2) {
    // V2: 3-byte length prefix (big-endian)
    const overallLength = packetLength + 3;
    const newArr = new Uint8Array(overallLength);
    newArr[0] = (packetLength >> 16) & 255;
    newArr[1] = (packetLength >> 8) & 255;
    newArr[2] = packetLength & 255;
    newArr[3] = msgTypeCode;
    newArr.set(serializedPBPacket, 4);
    return newArr;
  }

  // V1: 2-byte length prefix (big-endian)
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
  onload: (reader: FileReader) => void,
) => {
  const reader = new FileReader();
  reader.onload = () => onload(reader);
  reader.readAsArrayBuffer(data);
};

type EnumOption = { label: string; value: number | string };

export function enumToOptions<T extends Record<string, string | number>>(
  enumObj: T,
): EnumOption[] {
  return Object.keys(enumObj)
    .filter((key) => isNaN(Number(key))) // Filter out numeric keys (reverse mapping)
    .map((key) => ({
      label: key, // The string key of the enum
      value: enumObj[key as keyof T], // The associated value of the enum
    }));
}

export function getEnumLabel<T extends Record<string, string | number>>(
  enumObj: T,
  value: number,
): string | undefined {
  // Find the key where the value matches the input number
  const label = Object.keys(enumObj).find(
    (key) => enumObj[key as keyof T] === value && isNaN(Number(key)), // Exclude reverse mapping numeric keys
  );
  return label;
}

export function getEnumValue<T extends Record<string, string | number>>(
  enumObj: T,
  label: string,
): number | undefined {
  // Ensure the label exists in the enum
  if (label in enumObj) {
    const value = enumObj[label as keyof T];
    // Ensure the value is a number (to exclude reverse-mapping strings in numeric enums)
    if (typeof value === "number") {
      return value;
    }
  }
  return undefined;
}
