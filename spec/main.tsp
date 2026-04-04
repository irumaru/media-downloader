import "@typespec/http";

using TypeSpec.Http;

@service(#{
  title: "Media Downloader API",
})
@route("/api/{secret}")
namespace MediaDownloader;

/** ダウンロードステータス */
enum DownloadStatus {
  pending,
  downloading,
  converting,
  completed,
  error,
}

/** ダウンロード情報 */
model Download {
  id: string;
  url: string;
  title?: string;
  status: DownloadStatus;
  progress: int32;
  filename?: string;
  error?: string;
  createdAt: utcDateTime;
  completedAt?: utcDateTime;
}

/** ダウンロード開始リクエスト */
model CreateDownloadRequest {
  url: string;
}

/** ダウンロード一覧レスポンス */
model DownloadListResponse {
  downloads: Download[];
}

/** チャンネル情報レスポンス */
model ChannelInfoResponse {
  name: string;
}

/** エラーレスポンス */
@error
model ErrorResponse {
  message: string;
}

/** チャンネル情報を取得 */
@get
op getChannelInfo(@path secret: string): ChannelInfoResponse | ErrorResponse;

/** ダウンロード一覧を取得 */
@route("/downloads")
@get
op listDownloads(@path secret: string): DownloadListResponse | ErrorResponse;

/** ダウンロードを開始 */
@route("/downloads")
@post
op createDownload(
  @path secret: string,
  @body body: CreateDownloadRequest,
): {
  @statusCode statusCode: 202;
  @body download: Download;
} | ErrorResponse;

/** ダウンロード情報を取得 */
@route("/downloads/{id}")
@get
op getDownload(@path secret: string, @path id: string): Download | ErrorResponse;
