export const MAX_LOGO_BYTES = 16 * 1024 * 1024;
export const MAX_BROWSER_PNG_EDGE = 4096;

export function coordinatesAreValid(lat: number, lng: number): boolean {
  return Number.isFinite(lat) && lat >= -90 && lat <= 90 && Number.isFinite(lng) && lng >= -180 && lng <= 180;
}
