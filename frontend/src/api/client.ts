// schema.d.ts is auto-generated from the OpenAPI spec via `mise run gen:ts`.
// Run `mise run gen:ts` to generate it before building.
import createClient from "openapi-fetch";
import type { paths } from "./schema";

export type { components } from "./schema";

export type DownloadStatus =
  components["schemas"]["DownloadStatus"];

export type Download =
  components["schemas"]["Download"];

const client = createClient<paths>({ baseUrl: "/" });

export async function getChannelInfo(
  secret: string
): Promise<{ name: string } | null> {
  const { data, error, response } = await client.GET(
    "/api/{secret}" as never,
    { params: { path: { secret } } }
  );
  if (response.status === 404) return null;
  if (error) throw new Error((error as { message?: string }).message ?? "request failed");
  return data as { name: string };
}

export async function listDownloads(
  secret: string
): Promise<{ downloads: Download[] }> {
  const { data, error } = await client.GET(
    "/api/{secret}/downloads" as never,
    { params: { path: { secret } } }
  );
  if (error) throw new Error((error as { message?: string }).message ?? "request failed");
  return data as { downloads: Download[] };
}

export async function createDownload(
  secret: string,
  url: string
): Promise<Download> {
  const { data, error } = await client.POST(
    "/api/{secret}/downloads" as never,
    {
      params: { path: { secret } },
      body: { url },
    }
  );
  if (error) throw new Error((error as { message?: string }).message ?? "request failed");
  return data as Download;
}
