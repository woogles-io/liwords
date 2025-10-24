// VDO.ninja integration utilities for tournament monitoring

/**
 * Generates a unique monitoring key for a user in a tournament
 * Format: {tournamentId}_{randomAlphanumeric}
 * Only uses letters, numbers, and underscores (VDO.Ninja compatible)
 * The tournamentId prefix allows O(1) webhook lookup
 */
export function generateMonitoringKey(
  tournamentId: string,
  userId: string,
): string {
  // Generate 12 character random alphanumeric string
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const randomBytes = crypto.getRandomValues(new Uint8Array(12));
  const randomSuffix = Array.from(randomBytes)
    .map((byte) => chars[byte % chars.length])
    .join("");

  return `${tournamentId}_${randomSuffix}`;
}

/**
 * Generates the screenshot key from a base camera key
 */
export function getScreenshotKey(cameraKey: string): string {
  return `${cameraKey}_ss`;
}

/**
 * Generates a vdo.ninja share URL for camera stream
 */
export function generateCameraShareUrl(
  key: string,
  room: string,
  username?: string,
): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  const apiKey = `&api=${encodeURIComponent(key)}_api`;
  const label = username ? `${username} - Camera` : "Camera";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&retry=30&retrytimeout=10000&label=${encodeURIComponent(label)}${apiKey}${postapi}`;
}

/**
 * Generates a vdo.ninja share URL for screenshot/screen share stream
 */
export function generateScreenshotShareUrl(
  key: string,
  room: string,
  username?: string,
): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  const apiKey = `&api=${encodeURIComponent(key)}_api`;
  const label = username ? `${username} - Screen` : "Screen";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&screenshare&retry=30&retrytimeout=10000&label=${encodeURIComponent(label)}${apiKey}${postapi}`;
}

/**
 * Generates a vdo.ninja view URL for directors to watch a camera stream
 */
export function generateCameraViewUrl(
  key: string,
  room: string,
  lowQuality: boolean = true,
): string {
  const qualityParam = lowQuality ? "&scale=30" : "";
  return `https://vdo.ninja/?view=${encodeURIComponent(key)}${qualityParam}`;
}

/**
 * Generates a vdo.ninja view URL for directors to watch a screenshot stream
 */
export function generateScreenshotViewUrl(
  key: string,
  room: string,
  lowQuality: boolean = true,
): string {
  const qualityParam = lowQuality ? "&scale=30" : "";
  return `https://vdo.ninja/?view=${encodeURIComponent(key)}${qualityParam}`;
}

/**
 * Generates a vdo.ninja view URL for directors to watch multiple streams simultaneously
 * VDO.Ninja supports comma-separated view IDs to show multiple streams in a grid
 */
export function generateMultiStreamViewUrl(
  keys: string[],
  lowQuality: boolean = true,
): string {
  if (keys.length === 0) {
    return "";
  }
  // VDO.Ninja supports comma-separated view IDs: ?view=key1,key2,key3
  const viewParam = keys.map((k) => encodeURIComponent(k)).join(",");
  const qualityParam = lowQuality ? "&scale=30" : "";
  // showlabels displays the labels set during push (username - Camera/Screen)
  return `https://vdo.ninja/?view=${viewParam}${qualityParam}&showlabels`;
}

/**
 * Generates a QR code-friendly URL that directly opens vdo.ninja
 * for phone camera setup (bypasses authentication)
 */
export function generatePhoneQRUrl(
  key: string,
  room: string,
  username?: string,
): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  const apiKey = `&api=${encodeURIComponent(key)}_api`;
  const label = username ? `${username} - Camera` : "Phone Camera";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&retry=30&retrytimeout=10000&label=${encodeURIComponent(label)}${apiKey}${postapi}`;
}
