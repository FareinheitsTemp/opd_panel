use tokio::sync::broadcast;

/// Subscribe to a server's log broadcast channel.
/// Returns a broadcast::Receiver that yields log lines.
pub fn subscribe(tx: &broadcast::Sender<String>) -> broadcast::Receiver<String> {
    tx.subscribe()
}
