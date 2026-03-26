import ChannelPage from "./pages/ChannelPage";

export default function App() {
  // Extract the first path segment as the channel secret.
  // e.g. "/abc123def456" → "abc123def456"
  const secret = window.location.pathname.split("/").filter(Boolean)[0] ?? "";

  return <ChannelPage secret={secret} />;
}
