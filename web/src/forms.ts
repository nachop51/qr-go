// Content-type definitions: the tab list, each type's form fields, and how a
// form's values become the string payload (via the Go content helpers, so the
// escaping rules live in exactly one place).

import { coordinatesAreValid } from "./validation";

export interface Field {
  key: string;
  label: string;
  kind: "text" | "textarea" | "url" | "tel" | "email" | "number" | "password" | "select" | "checkbox" | "datetime";
  placeholder?: string;
  options?: [value: string, label: string][];
  step?: string;
  min?: string;
  max?: string;
}

export type Values = Record<string, string>;

export interface ContentType {
  id: string;
  label: string;
  /** Returns the payload string, "" when required fields are still empty, or throws never. */
  payload(v: Values): string | { error: string };
}

export const FIELDS: Record<string, Field[]> = {
  text: [{ key: "text", label: "Text", kind: "textarea", placeholder: "Anything. Plain text ends up in the code verbatim" }],
  url: [{ key: "url", label: "URL", kind: "url", placeholder: "https://example.com" }],
  wifi: [
    { key: "ssid", label: "Network name (SSID)", kind: "text", placeholder: "my-network" },
    { key: "pass", label: "Password", kind: "password", placeholder: "" },
    {
      key: "auth",
      label: "Security",
      kind: "select",
      options: [
        ["", "Auto"],
        ["WPA", "WPA / WPA2 / WPA3"],
        ["WEP", "WEP"],
        ["nopass", "Open network"],
      ],
    },
    { key: "hidden", label: "Hidden network", kind: "checkbox" },
  ],
  vcard: [
    { key: "first", label: "First name", kind: "text" },
    { key: "last", label: "Last name", kind: "text" },
    { key: "org", label: "Company", kind: "text" },
    { key: "title", label: "Job title", kind: "text" },
    { key: "phone", label: "Phone", kind: "tel" },
    { key: "email", label: "Email", kind: "email" },
    { key: "url", label: "Website", kind: "url" },
    { key: "address", label: "Address", kind: "text" },
  ],
  event: [
    { key: "summary", label: "Title", kind: "text", placeholder: "Team offsite" },
    { key: "location", label: "Location", kind: "text" },
    { key: "description", label: "Description", kind: "textarea" },
    { key: "start", label: "Starts", kind: "datetime" },
    { key: "end", label: "Ends", kind: "datetime" },
  ],
  tel: [{ key: "number", label: "Phone number", kind: "tel", placeholder: "+1 555 123 4567" }],
  sms: [
    { key: "number", label: "Phone number", kind: "tel", placeholder: "+1 555 123 4567" },
    { key: "message", label: "Message", kind: "textarea", placeholder: "Pre-filled text" },
  ],
  geo: [
    { key: "lat", label: "Latitude", kind: "number", step: "any", min: "-90", max: "90", placeholder: "-34.9011" },
    { key: "lng", label: "Longitude", kind: "number", step: "any", min: "-180", max: "180", placeholder: "-56.1645" },
  ],
  email: [
    { key: "to", label: "To", kind: "email", placeholder: "someone@example.com" },
    { key: "subject", label: "Subject", kind: "text" },
    { key: "body", label: "Body", kind: "textarea" },
  ],
};

export const CONTENT_TYPES: ContentType[] = [
  { id: "text", label: "Text", payload: (v) => v.text?.trim() ?? "" },
  { id: "url", label: "URL", payload: (v) => (v.url?.trim() ? qrgo.content.url({ url: v.url.trim() }) : "") },
  {
    id: "wifi",
    label: "Wi-Fi",
    payload: (v) =>
      v.ssid?.trim()
        ? qrgo.content.wifi({ ssid: v.ssid.trim(), pass: v.pass ?? "", auth: v.auth ?? "", hidden: v.hidden === "true" })
        : "",
  },
  {
    id: "vcard",
    label: "Contact",
    payload: (v) => {
      const fields = { first: v.first, last: v.last, org: v.org, title: v.title, phone: v.phone, email: v.email, url: v.url, address: v.address };
      const any = Object.values(fields).some((x) => x?.trim());
      return any ? qrgo.content.vcard(clean(fields)) : "";
    },
  },
  {
    id: "event",
    label: "Event",
    payload: (v) =>
      v.summary?.trim()
        ? qrgo.content.event(clean({ summary: v.summary, location: v.location, description: v.description, start: v.start, end: v.end }))
        : "",
  },
  { id: "tel", label: "Call", payload: (v) => (v.number?.trim() ? qrgo.content.tel({ number: v.number.trim() }) : "") },
  {
    id: "sms",
    label: "SMS",
    payload: (v) => (v.number?.trim() ? qrgo.content.sms({ number: v.number.trim(), message: v.message ?? "" }) : ""),
  },
  {
    id: "geo",
    label: "Location",
    payload: (v) => {
      const lat = parseFloat(v.lat ?? "");
      const lng = parseFloat(v.lng ?? "");
      if (Number.isNaN(lat) || Number.isNaN(lng)) return "";
      if (!coordinatesAreValid(lat, lng)) {
        return { error: "Latitude must be within -90..90 and longitude within -180..180" };
      }
      return qrgo.content.geo({ lat, lng });
    },
  },
  {
    id: "email",
    label: "Email",
    payload: (v) => (v.to?.trim() ? qrgo.content.email({ to: v.to.trim(), subject: v.subject ?? "", body: v.body ?? "" }) : ""),
  },
];

function clean(o: Record<string, string | undefined>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(o)) if (v?.trim()) out[k] = v.trim();
  return out;
}
