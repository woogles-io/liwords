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
export function generateCameraShareUrl(key: string, room: string): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&label=${encodeURIComponent("Camera")}${postapi}`;
}

/**
 * Generates a vdo.ninja share URL for screenshot/screen share stream
 */
export function generateScreenshotShareUrl(key: string, room: string): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&screenshare&label=${encodeURIComponent("Screen")}${postapi}`;
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
 * Generates a QR code-friendly URL that directly opens vdo.ninja
 * for phone camera setup (bypasses authentication)
 */
export function generatePhoneQRUrl(key: string, room: string): string {
  const webhookUrl = import.meta.env.PUBLIC_VDO_WEBHOOK_URL;
  const postapi = webhookUrl
    ? `&postapi=${encodeURIComponent(webhookUrl)}`
    : "";
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&label=${encodeURIComponent("Phone Camera")}${postapi}`;
}
