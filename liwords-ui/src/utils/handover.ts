// Stores the settings a handover link carries: each key is set to its
// value, and a key given as null is removed.
export const applyHandoverStorage = (settings: Record<string, unknown>) => {
  for (const k in settings) {
    if (settings[k] === null) {
      localStorage.removeItem(k);
    } else {
      localStorage.setItem(k, String(settings[k]));
    }
  }
};
