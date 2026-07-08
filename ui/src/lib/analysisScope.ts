export function analysisScopeSummary(include: string, exclude: string): string {
  const includeCount = splitAnalysisScopeLines(include).length;
  const excludeCount = splitAnalysisScopeLines(exclude).length;
  if (includeCount === 0 && excludeCount === 0) {
    return "all files";
  }
  return `include ${includeCount} / exclude ${excludeCount}`;
}

export function splitAnalysisScopeLines(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}
