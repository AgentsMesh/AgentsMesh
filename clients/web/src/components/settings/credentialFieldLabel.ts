export function getCredentialFieldLabel(
  fieldName: string,
  t: (key: string) => string
): string {
  const i18nKey = `settings.agentCredentials.fields.${fieldName}`;
  const translated = t(i18nKey);
  return translated !== i18nKey ? translated : fieldName;
}
