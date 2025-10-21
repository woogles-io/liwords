// VDO.ninja integration utilities for tournament monitoring

/**
 * Simple hash function to generate a short code from a string
 */
function simpleHash(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash).toString(36);
}

/**
 * Generates a unique monitoring key for a user in a tournament
 * Format: {shortTournamentHash}_{shortUserHash}_{timestamp}
 */
export function generateMonitoringKey(
  tournamentId: string,
  userId: string,
): string {
  const timestamp = Date.now();
  const tourneyHash = simpleHash(tournamentId);
  const userHash = simpleHash(userId);
  const timeHash = timestamp.toString(36); // Base36 is shorter
  return `${tourneyHash}_${userHash}_${timeHash}`;
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
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&label=${encodeURIComponent("Camera")}`;
}

/**
 * Generates a vdo.ninja share URL for screenshot/screen share stream
 */
export function generateScreenshotShareUrl(key: string, room: string): string {
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&screenshare&label=${encodeURIComponent("Screen")}`;
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
  return `https://vdo.ninja/?push=${encodeURIComponent(key)}&webcam&label=${encodeURIComponent("Phone Camera")}`;
}
