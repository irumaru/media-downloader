import { useCallback, useEffect, useRef, useState } from "react";
import {
  createDownload,
  getChannelInfo,
  listDownloads,
  type Download,
} from "../api/client";

const STATUS_COLOR: Record<string, string> = {
  pending: "#9ca3af",
  downloading: "#3b82f6",
  converting: "#f97316",
  completed: "#22c55e",
  error: "#ef4444",
};

const STATUS_LABEL: Record<string, string> = {
  pending: "待機中",
  downloading: "ダウンロード中",
  converting: "変換中",
  completed: "完了",
  error: "エラー",
};

const ACTIVE_STATUSES = new Set(["pending", "downloading", "converting"]);

interface Props {
  secret: string;
}

export default function ChannelPage({ secret }: Props) {
  const [channelName, setChannelName] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [downloads, setDownloads] = useState<Download[]>([]);
  const [url, setUrl] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchDownloads = useCallback(async () => {
    try {
      const res = await listDownloads(secret);
      setDownloads(res.downloads);
      // Stop polling if no active downloads remain.
      if (!res.downloads.some((d) => ACTIVE_STATUSES.has(d.status))) {
        if (pollingRef.current) {
          clearInterval(pollingRef.current);
          pollingRef.current = null;
        }
      }
    } catch {
      // ignore polling errors
    }
  }, [secret]);

  const startPolling = useCallback(() => {
    if (pollingRef.current) return;
    pollingRef.current = setInterval(fetchDownloads, 2000);
  }, [fetchDownloads]);

  useEffect(() => {
    if (!secret) return;
    getChannelInfo(secret).then((info) => {
      if (info === null) {
        setNotFound(true);
      } else {
        setChannelName(info.name);
        fetchDownloads();
      }
    });
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, [secret, fetchDownloads]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitError(null);
    if (!url.trim()) return;
    setSubmitting(true);
    try {
      const d = await createDownload(secret, url.trim());
      setDownloads((prev) => [d, ...prev]);
      setUrl("");
      startPolling();
    } catch (err: unknown) {
      setSubmitError((err as Error).message ?? "エラーが発生しました");
    } finally {
      setSubmitting(false);
    }
  };

  if (notFound) {
    return (
      <div style={styles.container}>
        <p style={{ color: "#ef4444" }}>チャンネルが見つかりません。</p>
      </div>
    );
  }

  if (channelName === null) {
    return <div style={styles.container}>読み込み中...</div>;
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.heading}>{channelName}</h1>

      <form onSubmit={handleSubmit} style={styles.form}>
        <input
          type="text"
          placeholder="YouTube URL を入力"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          style={styles.input}
          disabled={submitting}
        />
        <button type="submit" disabled={submitting || !url.trim()} style={styles.button}>
          {submitting ? "追加中..." : "ダウンロード"}
        </button>
      </form>
      {submitError && <p style={styles.errorText}>{submitError}</p>}

      {downloads.length > 0 && (
        <div style={styles.list}>
          <h2 style={styles.subheading}>ダウンロード一覧</h2>
          {downloads.map((d) => (
            <DownloadItem key={d.id} download={d} />
          ))}
        </div>
      )}
    </div>
  );
}

function DownloadItem({ download: d }: { download: Download }) {
  const color = STATUS_COLOR[d.status] ?? "#9ca3af";
  const label = STATUS_LABEL[d.status] ?? d.status;
  const showProgress = ACTIVE_STATUSES.has(d.status);

  return (
    <div style={styles.item}>
      <div style={styles.itemHeader}>
        <span style={styles.title}>{d.title ?? d.url}</span>
        <span style={{ ...styles.badge, backgroundColor: color }}>{label}</span>
      </div>

      {showProgress && (
        <div style={styles.progressTrack}>
          <div
            style={{ ...styles.progressFill, width: `${d.progress}%`, backgroundColor: color }}
          />
        </div>
      )}

      {d.status === "completed" && d.filename && (
        <span style={styles.filename}>{d.filename}</span>
      )}
      {d.status === "error" && d.error && (
        <span style={styles.errorText}>{d.error}</span>
      )}
    </div>
  );
}

const styles = {
  container: {
    maxWidth: 720,
    margin: "0 auto",
    padding: "2rem 1rem",
    fontFamily: "system-ui, sans-serif",
    color: "#111827",
  },
  heading: {
    fontSize: "1.5rem",
    fontWeight: 700,
    marginBottom: "1.5rem",
  },
  subheading: {
    fontSize: "1.1rem",
    fontWeight: 600,
    marginBottom: "1rem",
    color: "#374151",
  },
  form: {
    display: "flex",
    gap: "0.5rem",
    marginBottom: "0.5rem",
  },
  input: {
    flex: 1,
    padding: "0.5rem 0.75rem",
    border: "1px solid #d1d5db",
    borderRadius: 6,
    fontSize: "0.95rem",
    outline: "none",
  },
  button: {
    padding: "0.5rem 1.25rem",
    backgroundColor: "#3b82f6",
    color: "#fff",
    border: "none",
    borderRadius: 6,
    fontSize: "0.95rem",
    cursor: "pointer",
    whiteSpace: "nowrap" as const,
  },
  errorText: {
    color: "#ef4444",
    fontSize: "0.875rem",
    marginTop: "0.25rem",
  },
  list: {
    marginTop: "2rem",
  },
  item: {
    padding: "0.875rem 1rem",
    border: "1px solid #e5e7eb",
    borderRadius: 8,
    marginBottom: "0.75rem",
    backgroundColor: "#f9fafb",
  },
  itemHeader: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    gap: "0.5rem",
    marginBottom: "0.4rem",
  },
  title: {
    fontSize: "0.9rem",
    fontWeight: 500,
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap" as const,
    flex: 1,
  },
  badge: {
    fontSize: "0.75rem",
    color: "#fff",
    padding: "0.2rem 0.6rem",
    borderRadius: 9999,
    flexShrink: 0,
  },
  progressTrack: {
    height: 6,
    backgroundColor: "#e5e7eb",
    borderRadius: 9999,
    overflow: "hidden",
    marginTop: "0.4rem",
  },
  progressFill: {
    height: "100%",
    borderRadius: 9999,
    transition: "width 0.3s ease",
  },
  filename: {
    fontSize: "0.8rem",
    color: "#6b7280",
    marginTop: "0.25rem",
    display: "block",
  },
} as const;
