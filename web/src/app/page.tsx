'use client'

import { useEffect, useRef, useState, useCallback } from 'react'

const API = 'http://127.0.0.1:51201'

interface Server {
  id: string
  name: string
  status: string
  port: number
  pid: number
  ram_used: number
  ram_max: number
  cpu: number
  uptime: string
  jar: string
}

function formatBytes(mb: number) {
  if (mb < 1024) return `${mb} MB`
  return `${(mb / 1024).toFixed(1)} GB`
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    running: 'text-green-400',
    stopped: 'text-gray-500',
    crashed: 'text-red-400',
    starting: 'text-yellow-400',
  }
  return (
    <span className={`flex items-center gap-1.5 text-xs font-mono uppercase tracking-wider ${colors[status] ?? 'text-gray-400'}`}>
      <span className={`status-dot ${status}`} />
      {status}
    </span>
  )
}

function LogViewer({ serverId, serverName }: { serverId: string; serverName: string }) {
  const [logs, setLogs] = useState<string[]>([])
  const [cmd, setCmd] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    setLogs([])
    if (esRef.current) esRef.current.close()

    const es = new EventSource(`${API}/ws/logs/${serverId}`)
    esRef.current = es
    es.onmessage = (e) => {
      setLogs(prev => [...prev.slice(-500), e.data])
    }
    es.onerror = () => {
      setLogs(prev => [...prev, '[connection lost — server may be stopped]'])
    }
    return () => es.close()
  }, [serverId])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const sendCmd = async () => {
    if (!cmd.trim()) return
    await fetch(`${API}/api/servers/${serverId}/command`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cmd }),
    })
    setCmd('')
  }

  const classifyLine = (line: string) => {
    const l = line.toLowerCase()
    if (l.includes('warn')) return 'warn'
    if (l.includes('error') || l.includes('exception') || l.includes('fatal')) return 'error'
    if (l.includes('joined the game') || l.includes('logged in')) return 'join'
    if (l.includes('info')) return 'info'
    return ''
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{
        flex: 1,
        overflowY: 'auto',
        background: '#0d0d0d',
        border: '1px solid rgba(193,20,20,0.2)',
        borderRadius: '8px 8px 0 0',
        padding: '12px',
        minHeight: 0,
      }}>
        {logs.length === 0 ? (
          <p style={{ color: '#555', fontFamily: 'JetBrains Mono, monospace', fontSize: 12 }}>
            Waiting for logs from {serverName}...
          </p>
        ) : (
          logs.map((line, i) => (
            <div key={i} className={`log-line ${classifyLine(line)}`}>{line}</div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
      <div style={{
        display: 'flex',
        gap: 8,
        padding: '8px',
        background: '#111',
        border: '1px solid rgba(193,20,20,0.2)',
        borderTop: 'none',
        borderRadius: '0 0 8px 8px',
      }}>
        <span style={{ color: '#c11414', fontFamily: 'monospace', lineHeight: '34px', fontSize: 14 }}>{'>'}</span>
        <input
          value={cmd}
          onChange={e => setCmd(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && sendCmd()}
          placeholder="Enter server command..."
          style={{
            flex: 1,
            background: 'transparent',
            border: 'none',
            outline: 'none',
            color: '#f0f0f0',
            fontFamily: 'JetBrains Mono, monospace',
            fontSize: 13,
          }}
        />
        <button
          onClick={sendCmd}
          style={{
            background: '#c11414',
            color: 'white',
            border: 'none',
            borderRadius: 6,
            padding: '6px 16px',
            cursor: 'pointer',
            fontSize: 13,
            fontWeight: 500,
            transition: 'background 150ms',
          }}
          onMouseOver={e => (e.currentTarget.style.background = '#e51d1d')}
          onMouseOut={e => (e.currentTarget.style.background = '#c11414')}
        >
          Send
        </button>
      </div>
    </div>
  )
}

function ServerCard({
  server,
  selected,
  onSelect,
  onAction,
}: {
  server: Server
  selected: boolean
  onSelect: () => void
  onAction: (action: string) => void
}) {
  const ramPct = server.ram_max > 0
    ? Math.round((server.ram_used / 1024 / 1024 / server.ram_max) * 100)
    : 0

  return (
    <div
      onClick={onSelect}
      style={{
        background: selected ? 'rgba(193,20,20,0.1)' : '#111',
        border: selected ? '1px solid rgba(193,20,20,0.5)' : '1px solid rgba(255,255,255,0.06)',
        borderRadius: 10,
        padding: '14px 16px',
        cursor: 'pointer',
        transition: 'all 200ms',
        animation: 'fadeIn 0.3s ease-in-out',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
        <div>
          <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 4 }}>{server.name}</div>
          <div style={{ color: '#555', fontSize: 11, fontFamily: 'monospace' }}>{server.id}</div>
        </div>
        <StatusBadge status={server.status} />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 12 }}>
        <div style={{ background: '#0d0d0d', borderRadius: 6, padding: '8px 10px' }}>
          <div style={{ color: '#555', fontSize: 11, marginBottom: 2 }}>PORT</div>
          <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{server.port}</div>
        </div>
        <div style={{ background: '#0d0d0d', borderRadius: 6, padding: '8px 10px' }}>
          <div style={{ color: '#555', fontSize: 11, marginBottom: 2 }}>CPU</div>
          <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{server.cpu.toFixed(1)}%</div>
        </div>
        <div style={{ background: '#0d0d0d', borderRadius: 6, padding: '8px 10px' }}>
          <div style={{ color: '#555', fontSize: 11, marginBottom: 2 }}>RAM</div>
          <div style={{ fontFamily: 'monospace', fontSize: 13 }}>
            {formatBytes(Math.round(server.ram_used / 1024 / 1024))} / {server.ram_max}MB
          </div>
        </div>
        <div style={{ background: '#0d0d0d', borderRadius: 6, padding: '8px 10px' }}>
          <div style={{ color: '#555', fontSize: 11, marginBottom: 2 }}>UPTIME</div>
          <div style={{ fontFamily: 'monospace', fontSize: 13 }}>{server.uptime || '—'}</div>
        </div>
      </div>

      {server.ram_max > 0 && (
        <div style={{ marginBottom: 12 }}>
          <div style={{ background: '#1a1a1a', borderRadius: 4, height: 4, overflow: 'hidden' }}>
            <div style={{
              width: `${Math.min(ramPct, 100)}%`,
              height: '100%',
              background: ramPct > 80 ? '#ef4444' : ramPct > 60 ? '#eab308' : '#c11414',
              borderRadius: 4,
              transition: 'width 500ms',
            }} />
          </div>
        </div>
      )}

      <div style={{ display: 'flex', gap: 6 }} onClick={e => e.stopPropagation()}>
        {server.status !== 'running' ? (
          <button
            onClick={() => onAction('start')}
            style={{
              flex: 1,
              background: '#c11414',
              color: 'white',
              border: 'none',
              borderRadius: 6,
              padding: '7px 0',
              cursor: 'pointer',
              fontSize: 13,
              fontWeight: 500,
              transition: 'background 150ms',
            }}
            onMouseOver={e => (e.currentTarget.style.background = '#e51d1d')}
            onMouseOut={e => (e.currentTarget.style.background = '#c11414')}
          >
            ▶ Start
          </button>
        ) : (
          <>
            <button
              onClick={() => onAction('restart')}
              style={{
                flex: 1,
                background: 'rgba(234,179,8,0.15)',
                color: '#eab308',
                border: '1px solid rgba(234,179,8,0.3)',
                borderRadius: 6,
                padding: '7px 0',
                cursor: 'pointer',
                fontSize: 13,
                fontWeight: 500,
                transition: 'background 150ms',
              }}
              onMouseOver={e => (e.currentTarget.style.background = 'rgba(234,179,8,0.25)')}
              onMouseOut={e => (e.currentTarget.style.background = 'rgba(234,179,8,0.15)')}
            >
              ↺ Restart
            </button>
            <button
              onClick={() => onAction('stop')}
              style={{
                flex: 1,
                background: 'rgba(239,68,68,0.12)',
                color: '#ef4444',
                border: '1px solid rgba(239,68,68,0.25)',
                borderRadius: 6,
                padding: '7px 0',
                cursor: 'pointer',
                fontSize: 13,
                fontWeight: 500,
                transition: 'background 150ms',
              }}
              onMouseOver={e => (e.currentTarget.style.background = 'rgba(239,68,68,0.22)')}
              onMouseOut={e => (e.currentTarget.style.background = 'rgba(239,68,68,0.12)')}
            >
              ■ Stop
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export default function Home() {
  const [servers, setServers] = useState<Server[]>([])
  const [selected, setSelected] = useState<Server | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  const showToast = (msg: string) => {
    setToast(msg)
    setTimeout(() => setToast(null), 3000)
  }

  const fetchServers = useCallback(async () => {
    try {
      const res = await fetch(`${API}/api/servers`)
      if (!res.ok) throw new Error('daemon unreachable')
      const data = await res.json()
      setServers(data)
      setError(null)
    } catch {
      setError('Cannot connect to OPD daemon. Is it running?')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchServers()
    const interval = setInterval(fetchServers, 3000)
    return () => clearInterval(interval)
  }, [fetchServers])

  const handleAction = async (serverId: string, action: string) => {
    try {
      const res = await fetch(`${API}/api/servers/${serverId}/${action}`, { method: 'POST' })
      const data = await res.json()
      if (data.error) showToast(`Error: ${data.error}`)
      else showToast(`${action} sent to ${serverId}`)
      setTimeout(fetchServers, 500)
    } catch {
      showToast('Failed to send command')
    }
  }

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      height: '100dvh',
      background: '#0a0a0a',
      color: '#f0f0f0',
    }}>
      {/* Header */}
      <header style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '0 24px',
        height: 56,
        borderBottom: '1px solid rgba(193,20,20,0.25)',
        background: '#0d0d0d',
        flexShrink: 0,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {/* SVG Logo */}
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <rect width="28" height="28" rx="6" fill="#c11414" />
            <path d="M7 14L14 7L21 14L14 21L7 14Z" fill="white" fillOpacity="0.9" />
            <circle cx="14" cy="14" r="3" fill="#c11414" />
          </svg>
          <span style={{ fontWeight: 700, fontSize: 16, letterSpacing: '-0.3px' }}>OPD Panel</span>
          <span style={{
            background: 'rgba(193,20,20,0.15)',
            color: '#c11414',
            border: '1px solid rgba(193,20,20,0.3)',
            borderRadius: 4,
            padding: '2px 7px',
            fontSize: 11,
            fontFamily: 'monospace',
            letterSpacing: '0.5px',
          }}>BETA</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span style={{ color: '#555', fontSize: 12, fontFamily: 'monospace' }}>
            {servers.length} server{servers.length !== 1 ? 's' : ''}
          </span>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            color: error ? '#ef4444' : '#22c55e',
            fontSize: 12,
          }}>
            <span className={`status-dot ${error ? 'crashed' : 'running'}`} />
            {error ? 'daemon offline' : 'daemon online'}
          </div>
        </div>
      </header>

      {/* Main layout */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Left sidebar — server list */}
        <aside style={{
          width: 300,
          flexShrink: 0,
          borderRight: '1px solid rgba(255,255,255,0.06)',
          display: 'flex',
          flexDirection: 'column',
          background: '#0d0d0d',
        }}>
          <div style={{
            padding: '14px 16px 10px',
            borderBottom: '1px solid rgba(255,255,255,0.04)',
            fontSize: 11,
            color: '#555',
            letterSpacing: '0.8px',
            textTransform: 'uppercase',
            fontWeight: 600,
          }}>Servers</div>

          <div style={{ flex: 1, overflowY: 'auto', padding: '10px 12px', display: 'flex', flexDirection: 'column', gap: 8 }}>
            {loading && (
              <div style={{ color: '#555', fontSize: 13, padding: '20px 0', textAlign: 'center' }}>Loading...</div>
            )}
            {error && (
              <div style={{
                background: 'rgba(239,68,68,0.08)',
                border: '1px solid rgba(239,68,68,0.2)',
                borderRadius: 8,
                padding: 14,
                color: '#ef4444',
                fontSize: 13,
              }}>
                ⚠ {error}
              </div>
            )}
            {!loading && !error && servers.length === 0 && (
              <div style={{ color: '#555', fontSize: 13, padding: '20px 0', textAlign: 'center', lineHeight: 1.8 }}>
                No servers found.<br />
                <span style={{ fontSize: 12 }}>Run <code style={{ color: '#c11414' }}>opd create</code> to add one.</span>
              </div>
            )}
            {servers.map(s => (
              <ServerCard
                key={s.id}
                server={s}
                selected={selected?.id === s.id}
                onSelect={() => setSelected(s)}
                onAction={(action) => handleAction(s.id, action)}
              />
            ))}
          </div>
        </aside>

        {/* Right panel — logs */}
        <main style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          background: '#0a0a0a',
        }}>
          {selected ? (
            <>
              <div style={{
                padding: '12px 20px',
                borderBottom: '1px solid rgba(255,255,255,0.05)',
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                flexShrink: 0,
              }}>
                <StatusBadge status={selected.status} />
                <span style={{ fontWeight: 600, fontSize: 15 }}>{selected.name}</span>
                <span style={{ color: '#555', fontSize: 12, fontFamily: 'monospace' }}>:{selected.port}</span>
                <span style={{ marginLeft: 'auto', color: '#555', fontSize: 12 }}>Live logs</span>
              </div>
              <div style={{ flex: 1, padding: '12px 16px', overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
                <LogViewer serverId={selected.id} serverName={selected.name} />
              </div>
            </>
          ) : (
            <div style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#333',
              gap: 12,
            }}>
              <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
                <rect width="48" height="48" rx="12" fill="rgba(193,20,20,0.08)" />
                <path d="M16 24L24 16L32 24L24 32L16 24Z" stroke="#c11414" strokeWidth="1.5" fill="none" />
                <circle cx="24" cy="24" r="4" stroke="#c11414" strokeWidth="1.5" fill="none" />
              </svg>
              <p style={{ fontSize: 14 }}>Select a server to view logs</p>
              <p style={{ fontSize: 12, color: '#222' }}>Click any server card on the left</p>
            </div>
          )}
        </main>
      </div>

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed',
          bottom: 24,
          right: 24,
          background: '#181818',
          border: '1px solid rgba(193,20,20,0.4)',
          borderRadius: 8,
          padding: '10px 18px',
          fontSize: 13,
          color: '#f0f0f0',
          boxShadow: '0 4px 20px rgba(0,0,0,0.5)',
          zIndex: 1000,
          animation: 'fadeIn 0.2s ease-in-out',
        }}>
          {toast}
        </div>
      )}
    </div>
  )
}
