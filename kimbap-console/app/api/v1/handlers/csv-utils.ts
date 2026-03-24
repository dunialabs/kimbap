export function sanitizeCsvField(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }

  let field = String(value);
  if (/^\s*[=+\-@|]/.test(field)) {
    field = `'${field}`;
  }
  field = field.replace(/"/g, '""');
  if (field.includes(',') || field.includes('"') || field.includes('\n') || field.includes('\r')) {
    return `"${field}"`;
  }
  return field;
}
