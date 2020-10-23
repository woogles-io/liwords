export const singularCount = (n: number, singular: string, plural: string) =>
  `${n} ${n === 1 ? singular : plural}`;
