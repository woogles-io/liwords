export const canMod = (perms: Array<string>): boolean => {
  return perms.includes("adm") || perms.includes("mod");
};
