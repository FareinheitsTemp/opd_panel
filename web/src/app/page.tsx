'use client'

import { useEffect, useState, useRef, useCallback } from 'react'

const API = 'http://127.0.0.1:51201'

interface ServerEntry {
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

function fmtRam(bytes: number) {
  if (!bytes) return '0 MB'
  return (bytes / 1024 / 1024).toFixed(0) + ' MB'
}

function StatusDot({ status }: { status: string }) {
  const color =
    status === 'running' ? '#22c55e' :
    status === 'starting' ? '#f59e0b' :
    status === 'stopping' ? '#f97316' : '#6b7280'
  return (
    <span style={{
      display: 'inline-block', width: 8, height: 8,
      borderRadius: '50%', background: color,
      boxShadow: status === 'running' ? `0 0 6px ${color}` : 'none',
      marginRight: 8, flexShrink: 0,
    }} />
  )
}

interface AddServerModalProps {
  onClose: () => void
  onCreated: () => void
}

function AddServerModal({ onClose, onCreated }: AddServerModalProps) {
  const [form, setForm] = useState({
    id: '', name: '', port: '25565',
    ram_min_mb: '1024', ram_max_mb: '4096', jar: 'server.jar',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [created, setCreated] = useState<string | null>(null)

  const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm(f => ({ ...f, [k]: e.target.value }))

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch(`${API}/api/servers/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: form.id.trim(),
          name: form.name.trim() || form.id.trim(),
          port: parseInt(form.port),
          ram_min_mb: parseInt(form.ram_min_mb),
          ram_max_mb: parseInt(form.ram_max_mb),
          jar: form.jar.trim() || 'server.jar',
        }),
      })
      const data = await res.json()
      if (!res.ok) { setError(data.error || 'Unknown error'); return }
      setCreated(data.dir)
      onCreated()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Connection failed')
    } finally {
      setLoading(false)
    }
  }

  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '8px 12px',
    background: '#1a1a1a', border: '1px solid #333',
    borderRadius: 6, color: '#e0e0e0', fontSize: 14,
    outline: 'none',
  }
  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: 12,
    color: '#888', marginBottom: 4, fontWeight: 500,
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 1000,
    }} onClick={e => e.target === e.currentTarget && onClose()}>
      <div style={{
        background: '#111', border: '1px solid #2a2a2a',
        borderRadius: 12, padding: 28, width: 440, maxWidth: '95vw',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <h2 style={{ color: '#e0e0e0', fontSize: 18, fontWeight: 600 }}>Add Server</h2>
          <button onClick={onClose} style={{ color: '#666', fontSize: 20, cursor: 'pointer', background: 'none', border: 'none' }}>×</button>
        </div>

        {created ? (
          <div>
            <div style={{ color: '#22c55e', marginBottom: 12, fontSize: 15 }}>✔ Server created!</div>
            <div style={{ background: '#1a1a1a', borderRadius: 6, padding: '10px 14px', color: '#aaa', fontSize: 13, marginBottom: 16 }}>
              Place your <code style={{ color: '#e0e0e0' }}>{form.jar}</code> in:<br />
              <code style={{ color: '#c084fc', wordBreak: 'break-all' }}>{created}</code>
            </div>
            <button onClick={onClose} style={{
              width: '100%', padding: '9px 0',
              background: '#7f1d1d', color: '#fff', border: 'none',
              borderRadius: 6, cursor: 'pointer', fontSize: 14, fontWeight: 600,
            }}>Close</button>
          </div>
        ) : (
          <form onSubmit={submit}>
            <div style={{ display: 'grid', gap: 14 }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <label style={labelStyle}>Server ID *</label>
                  <input style={inputStyle} value={form.id} onChange={set('id')}
                    placeholder="survival" required pattern="[a-zA-Z0-9_-]+" />
                </div>
                <div>
                  <label style={labelStyle}>Display Name</label>
                  <input style={inputStyle} value={form.name} onChange={set('name')} placeholder="Survival" />
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <label style={labelStyle}>Port</label>
                  <input style={inputStyle} type="number" value={form.port} onChange={set('port')} min={1} max={65535} />
                </div>
                <div>
                  <label style={labelStyle}>Jar filename</label>
                  <input style={inputStyle} value={form.jar} onChange={set('jar')} placeholder="server.jar" />
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <label style={labelStyle}>Min RAM (MB)</label>
                  <input style={inputStyle} type="number" value={form.ram_min_mb} onChange={set('ram_min_mb')} min={128} />
                </div>
                <div>
                  <label style={labelStyle}>Max RAM (MB)</label>
                  <input style={inputStyle} type="number" value={form.ram_max_mb} onChange={set('ram_max_mb')} min={256} />
                </div>
              </div>
            </div>

            {error && <div style={{ color: '#f87171', fontSize: 13, marginTop: 12 }}>✗ {error}</div>}

            <button type="submit" disabled={loading} style={{
              marginTop: 18, width: '100%', padding: '10px 0',
              background: loading ? '#4a0a0a' : '#7f1d1d',
              color: loading ? '#888' : '#fff',
              border: 'none', borderRadius: 6, cursor: loading ? 'not-allowed' : 'pointer',
              fontSize: 14, fontWeight: 600, transition: 'background 0.15s',
            }}>
              {loading ? 'Creating...' : 'Create Server'}
            </button>
          </form>
        )}
      </div>
    </div>
  )
}

export default function Home() {
  const [servers, setServers] = useState<ServerEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [logs, setLogs] = useState<string[]>([])
  const [cmd, setCmd] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const sseRef = useRef<EventSource | null>(null)

  const fetchServers = useCallback(async () => {
    try {
      const r = await fetch(`${API}/api/servers`)
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setServers(data)
      setError(null)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Connection failed')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchServers()
    const t = setInterval(fetchServers, 3000)
    return () => clearInterval(t)
  }, [fetchServers])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  function selectServer(id: string) {
    setSelected(id)
    setLogs([])
    sseRef.current?.close()
    const es = new EventSource(`${API}/ws/logs/${id}`)
    es.onmessage = e => setLogs(prev => [...prev.slice(-500), e.data])
    sseRef.current = es
  }

  async function action(id: string, act: string) {
    await fetch(`${API}/api/servers/${id}/${act}`, { method: 'POST' })
    setTimeout(fetchServers, 500)
  }

  async function sendCmd(id: string) {
    if (!cmd.trim()) return
    await fetch(`${API}/api/servers/${id}/command`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cmd }),
    })
    setCmd('')
  }

  const sel = servers.find(s => s.id === selected)

  return (
    <div style={{
      minHeight: '100vh', background: '#0a0a0a',
      color: '#e0e0e0', fontFamily: "'Inter', system-ui, sans-serif",
      display: 'flex', flexDirection: 'column',
    }}>
      {/* Header */}
      <header style={{
        borderBottom: '1px solid #1e1e1e', padding: '14px 24px',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        background: '#0d0d0d',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
            <rect x="2" y="2" width="9" height="9" rx="1.5" fill="#dc2626"/>
            <rect x="13" y="2" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".6"/>
            <rect x="2" y="13" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".6"/>
            <rect x="13" y="13" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".3"/>
          </svg>
          <span style={{ fontWeight: 700, fontSize: 16, letterSpacing: '-0.02em' }}>opd panel</span>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          style={{
            background: '#7f1d1d', color: '#fff', border: 'none',
            borderRadius: 6, padding: '7px 14px', cursor: 'pointer',
            fontSize: 13, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
          }}
        >
          <span style={{ fontSize: 18, lineHeight: 1 }}>+</span> Add Server
        </button>
      </header>

      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Sidebar */}
        <aside style={{
          width: 280, borderRight: '1px solid #1e1e1e',
          overflowY: 'auto', background: '#0d0d0d',
          display: 'flex', flexDirection: 'column',
        }}>
          <div style={{ padding: '12px 16px 8px', fontSize: 11, color: '#555', fontWeight: 600, letterSpacing: '0.08em', textTransform: 'uppercase' }}>
            Servers {servers.length > 0 && `(${servers.length})`}
          </div>

          {loading && (
            <div style={{ padding: '20px 16px', color: '#555', fontSize: 13 }}>Connecting to daemon...</div>
          )}
          {error && (
            <div style={{ padding: '12px 16px', color: '#f87171', fontSize: 12 }}>
              ✗ {error}<br />
              <span style={{ color: '#555' }}>Is opd daemon running?</span>
            </div>
          )}
          {!loading && !error && servers.length === 0 && (
            <div style={{ padding: '16px', color: '#555', fontSize: 13 }}>
              No servers yet.<br />
              <button onClick={() => setShowAdd(true)} style={{
                marginTop: 8, color: '#dc2626', background: 'none',
                border: 'none', cursor: 'pointer', fontSize: 13, padding: 0,
              }}>+ Add your first server</button>
            </div>
          )}

          {servers.map(srv => (
            <button key={srv.id} onClick={() => selectServer(srv.id)}
              style={{
                display: 'flex', alignItems: 'center', width: '100%',
                padding: '10px 16px', background: selected === srv.id ? '#1a1a1a' : 'none',
                border: 'none', borderLeft: selected === srv.id ? '2px solid #dc2626' : '2px solid transparent',
                cursor: 'pointer', textAlign: 'left', transition: 'background 0.1s',
              }}
            >
              <StatusDot status={srv.status} />
              <div style={{ overflow: 'hidden' }}>
                <div style={{ color: '#e0e0e0', fontSize: 14, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {srv.name}
                </div>
                <div style={{ color: '#555', fontSize: 11 }}>{srv.id} · :{srv.port}</div>
              </div>
            </button>
          ))}
        </aside>

        {/* Main */}
        <main style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
          {!sel ? (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#333', flexDirection: 'column', gap: 8 }}>
              <span style={{ fontSize: 40 }}>⬛</span>
              <span style={{ fontSize: 14 }}>Select a server</span>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
              {/* Server Header */}
              <div style={{ padding: '16px 24px', borderBottom: '1px solid #1e1e1e', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <StatusDot status={sel.status} />
                  <div>
                    <div style={{ fontWeight: 600, fontSize: 16 }}>{sel.name}</div>
                    <div style={{ color: '#555', fontSize: 12 }}>{sel.id} · port {sel.port} · {sel.status}</div>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  {sel.status !== 'running' && sel.status !== 'starting' && (
                    <button onClick={() => action(sel.id, 'start')} style={btnStyle('#166534', '#22c55e')}>Start</button>
                  )}
                  {(sel.status === 'running' || sel.status === 'starting') && (
                    <button onClick={() => action(sel.id, 'restart')} style={btnStyle('#78350f', '#f59e0b')}>Restart</button>
                  )}
                  {(sel.status === 'running' || sel.status === 'starting') && (
                    <button onClick={() => action(sel.id, 'stop')} style={btnStyle('#7f1d1d', '#f87171')}>Stop</button>
                  )}
                </div>
              </div>

              {/* Stats */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 1, borderBottom: '1px solid #1e1e1e', background: '#1e1e1e' }}>
                {[
                  ['RAM Used', fmtRam(sel.ram_used)],
                  ['RAM Max', fmtRam(sel.ram_max)],
                  ['CPU', `${sel.cpu.toFixed(1)}%`],
                  ['Uptime', sel.uptime || '—'],
                ].map(([label, val]) => (
                  <div key={label} style={{ background: '#0d0d0d', padding: '12px 16px' }}>
                    <div style={{ color: '#555', fontSize: 11, marginBottom: 4 }}>{label}</div>
                    <div style={{ color: '#e0e0e0', fontSize: 15, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{val}</div>
                  </div>
                ))}
              </div>

              {/* Logs */}
              <div style={{ flex: 1, overflowY: 'auto', background: '#060606', fontFamily: 'monospace', fontSize: 12, padding: '12px 16px', lineHeight: 1.6 }}>
                {logs.length === 0 ? (
                  <span style={{ color: '#333' }}>No logs yet. Start the server or select a running server.</span>
                ) : (
                  logs.map((l, i) => <div key={i} style={{ color: l.includes('ERROR') || l.includes('WARN') ? '#f87171' : '#9ca3af' }}>{l}</div>)
                )}
                <div ref={logsEndRef} />
              </div>

              {/* Console input */}
              {sel.status === 'running' && (
                <div style={{ padding: '10px 16px', borderTop: '1px solid #1e1e1e', display: 'flex', gap: 8 }}>
                  <span style={{ color: '#555', fontFamily: 'monospace', fontSize: 13, alignSelf: 'center' }}>{'>'}</span>
                  <input
                    value={cmd}
                    onChange={e => setCmd(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && sendCmd(sel.id)}
                    placeholder="say Hello, world!"
                    style={{
                      flex: 1, background: 'none', border: 'none', outline: 'none',
                      color: '#e0e0e0', fontFamily: 'monospace', fontSize: 13,
                    }}
                  />
                  <button onClick={() => sendCmd(sel.id)} style={btnStyle('#1e1e1e', '#555')}>Send</button>
                </div>
              )}
            </div>
          )}
        </main>
      </div>

      {showAdd && (
        <AddServerModal
          onClose={() => setShowAdd(false)}
          onCreated={() => { fetchServers(); }}
        />
      )}
    </div>
  )
}

function btnStyle(bg: string, color: string): React.CSSProperties {
  return {
    background: bg, color, border: `1px solid ${color}33`,
    borderRadius: 6, padding: '6px 14px', cursor: 'pointer',
    fontSize: 13, fontWeight: 500,
  }
}
