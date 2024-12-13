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

type EnumOption = { label: string; value: number | string };

export function enumToOptions<T extends Record<string, string | number>>(
  enumObj: T
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
  value: number
): string | undefined {
  // Find the key where the value matches the input number
  const label = Object.keys(enumObj).find(
    (key) => enumObj[key as keyof T] === value && isNaN(Number(key)) // Exclude reverse mapping numeric keys
  );
  return label;
}

export function getEnumValue<T extends Record<string, string | number>>(
  enumObj: T,
  label: string
): number | undefined {
  // Ensure the label exists in the enum
  if (label in enumObj) {
    const value = enumObj[label as keyof T];
    // Ensure the value is a number (to exclude reverse-mapping strings in numeric enums)
    if (typeof value === 'number') {
      return value;
    }
  }
  return undefined;
}
