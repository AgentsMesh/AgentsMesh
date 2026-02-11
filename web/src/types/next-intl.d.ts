import common from "@/messages/en/common.json";
import auth from "@/messages/en/auth.json";
import landing from "@/messages/en/landing.json";
import app from "@/messages/en/app.json";
import settings from "@/messages/en/settings.json";
import ide from "@/messages/en/ide.json";
import repositories from "@/messages/en/repositories.json";
import runners from "@/messages/en/runners.json";
import docs from "@/messages/en/docs.json";
import content from "@/messages/en/content.json";

type Messages = typeof common &
  typeof auth &
  typeof landing &
  typeof app &
  typeof settings &
  typeof ide &
  typeof repositories &
  typeof runners &
  typeof docs &
  typeof content;

declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface IntlMessages extends Messages {}
}
