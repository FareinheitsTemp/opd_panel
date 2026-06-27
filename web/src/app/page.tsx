'use client'

import { useEffect, useState, useRef, useCallback } from 'react'

const API = 'http://127.0.0.1:51201'

interface ServerEntry {
  id: string; name: string; status: string; port: number
  pid: number; ram_used: number; ram_max: number; cpu: number
  uptime: string; jar: string; dir: string
  motd: string; max_players: number; gamemode: string
  difficulty: string; auto_restart: boolean; java_flags: string[]
}

interface DiskInfo { path: string; label: string; free_gb: number }

interface Plugin { name: string; size: number }

function fmtRam(b: number) { return b ? (b/1024/1024).toFixed(0)+' MB' : '0 MB' }
function fmtSize(b: number) {
  if (b > 1024*1024) return (b/1024/1024).toFixed(1)+' MB'
  if (b > 1024) return (b/1024).toFixed(0)+' KB'
  return b+' B'
}

function StatusDot({ status }: { status: string }) {
  const c = status==='running'?'#22c55e':status==='starting'?'#f59e0b':status==='stopping'?'#f97316':'#6b7280'
  return <span style={{ display:'inline-block',width:8,height:8,borderRadius:'50%',background:c,
    boxShadow:status==='running'?`0 0 6px ${c}`:'none',marginRight:8,flexShrink:0 }} />
}

function btn(bg: string, c: string, small = false): React.CSSProperties {
  return { background:bg, color:c, border:`1px solid ${c}33`, borderRadius:6,
    padding:small?'4px 10px':'6px 14px', cursor:'pointer', fontSize:small?12:13, fontWeight:500 }
}

// ---- Add Server Modal ----
function AddServerModal({ onClose, onCreated }: { onClose:()=>void, onCreated:()=>void }) {
  const [disks, setDisks] = useState<DiskInfo[]>([])
  const [currentRoot, setCurrentRoot] = useState('')
  const [form, setForm] = useState({ id:'', name:'', port:'25565', ram_min_mb:'1024', ram_max_mb:'4096', jar:'server.jar', servers_root:'' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [created, setCreated] = useState<string|null>(null)

  useEffect(() => {
    fetch(`${API}/api/disks`).then(r=>r.json()).then(d=>{
      setDisks(d.disks||[])
      setCurrentRoot(d.current_root||'')
      setForm(f=>({...f, servers_root: d.current_root||''}))
    }).catch(()=>{})
  }, [])

  const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement|HTMLSelectElement>) =>
    setForm(f=>({...f,[k]:e.target.value}))

  async function submit(e: React.FormEvent) {
    e.preventDefault(); setError(''); setLoading(true)
    try {
      const res = await fetch(`${API}/api/servers`, {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({
          id: form.id.trim(), name: form.name.trim()||form.id.trim(),
          port: parseInt(form.port), ram_min_mb: parseInt(form.ram_min_mb),
          ram_max_mb: parseInt(form.ram_max_mb), jar: form.jar.trim()||'server.jar',
          servers_root: form.servers_root || undefined,
        }),
      })
      const data = await res.json()
      if (!res.ok) { setError(data.error||'Unknown error'); return }
      setCreated(data.dir); onCreated()
    } catch(e: unknown) {
      setError(e instanceof Error ? e.message : 'Connection failed')
    } finally { setLoading(false) }
  }

  const inp: React.CSSProperties = { width:'100%', padding:'8px 12px', background:'#1a1a1a',
    border:'1px solid #333', borderRadius:6, color:'#e0e0e0', fontSize:14, outline:'none' }
  const lbl: React.CSSProperties = { display:'block', fontSize:12, color:'#888', marginBottom:4, fontWeight:500 }

  return (
    <div style={{ position:'fixed',inset:0,background:'rgba(0,0,0,0.75)',display:'flex',
      alignItems:'center',justifyContent:'center',zIndex:1000 }}
      onClick={e=>e.target===e.currentTarget&&onClose()}>
      <div style={{ background:'#111',border:'1px solid #2a2a2a',borderRadius:12,padding:28,width:480,maxWidth:'95vw' }}>
        <div style={{ display:'flex',justifyContent:'space-between',alignItems:'center',marginBottom:20 }}>
          <h2 style={{ color:'#e0e0e0',fontSize:18,fontWeight:600 }}>Add Server</h2>
          <button onClick={onClose} style={{ color:'#666',fontSize:20,cursor:'pointer',background:'none',border:'none' }}>×</button>
        </div>

        {created ? (
          <div>
            <div style={{ color:'#22c55e',marginBottom:12,fontSize:15 }}>✔ Server created!</div>
            <div style={{ background:'#1a1a1a',borderRadius:6,padding:'10px 14px',color:'#aaa',fontSize:13,marginBottom:16 }}>
              Place <code style={{color:'#e0e0e0'}}>{form.jar}</code> in:<br/>
              <code style={{color:'#c084fc',wordBreak:'break-all'}}>{created}</code>
            </div>
            <button onClick={onClose} style={{ width:'100%',padding:'9px 0',background:'#7f1d1d',color:'#fff',border:'none',borderRadius:6,cursor:'pointer',fontSize:14,fontWeight:600 }}>Close</button>
          </div>
        ) : (
          <form onSubmit={submit}>
            <div style={{ display:'grid',gap:14 }}>
              <div>
                <label style={lbl}>Storage Location</label>
                <select style={{...inp}} value={form.servers_root} onChange={e=>setForm(f=>({...f,servers_root:e.target.value}))}>
                  <option value={currentRoot}>{currentRoot} (current)</option>
                  {disks.filter(d=>d.path!==currentRoot).map(d=>(
                    <option key={d.path} value={d.path+'opd\\servers'}>
                      {d.label} — {(d.free_gb??0).toFixed(1)} GB free → {d.path}opd\servers
                    </option>
                  ))}
                </select>
              </div>
              <div style={{ display:'grid',gridTemplateColumns:'1fr 1fr',gap:12 }}>
                <div><label style={lbl}>Server ID *</label>
                  <input style={inp} value={form.id} onChange={set('id')} placeholder="survival" required /></div>
                <div><label style={lbl}>Display Name</label>
                  <input style={inp} value={form.name} onChange={set('name')} placeholder="Survival" /></div>
              </div>
              <div style={{ display:'grid',gridTemplateColumns:'1fr 1fr',gap:12 }}>
                <div><label style={lbl}>Port</label>
                  <input style={inp} type="number" value={form.port} onChange={set('port')} min={1} max={65535} /></div>
                <div><label style={lbl}>Jar filename</label>
                  <input style={inp} value={form.jar} onChange={set('jar')} placeholder="server.jar" /></div>
              </div>
              <div style={{ display:'grid',gridTemplateColumns:'1fr 1fr',gap:12 }}>
                <div><label style={lbl}>Min RAM (MB)</label>
                  <input style={inp} type="number" value={form.ram_min_mb} onChange={set('ram_min_mb')} min={128} /></div>
                <div><label style={lbl}>Max RAM (MB)</label>
                  <input style={inp} type="number" value={form.ram_max_mb} onChange={set('ram_max_mb')} min={256} /></div>
              </div>
            </div>
            {error && <div style={{ color:'#f87171',fontSize:13,marginTop:12 }}>✗ {error}</div>}
            <button type="submit" disabled={loading} style={{ marginTop:18,width:'100%',padding:'10px 0',
              background:loading?'#4a0a0a':'#7f1d1d',color:loading?'#888':'#fff',
              border:'none',borderRadius:6,cursor:loading?'not-allowed':'pointer',fontSize:14,fontWeight:600 }}>
              {loading?'Creating...':'Create Server'}
            </button>
          </form>
        )}
      </div>
    </div>
  )
}

// ---- Settings Panel ----
function SettingsPanel({ server, onSaved }: { server: ServerEntry, onSaved: ()=>void }) {
  const [form, setForm] = useState({
    name: server.name, port: String(server.port),
    ram_min_mb: '1024', ram_max_mb: String(server.ram_max/1024/1024||server.ram_max||4096),
    motd: server.motd||'', max_players: String(server.max_players||20),
    gamemode: server.gamemode||'survival', difficulty: server.difficulty||'normal',
    auto_restart: server.auto_restart||false, jar: server.jar||'server.jar',
    java_flags: (server.java_flags||[]).join(' '),
  })
  const [saved, setSaved] = useState(false)
  const [err, setErr] = useState('')

  const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement|HTMLSelectElement|HTMLTextAreaElement>) =>
    setForm(f=>({...f,[k]:e.target.value}))
  const setCheck = (k: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm(f=>({...f,[k]:e.target.checked}))

  async function save() {
    setErr('')
    try {
      const res = await fetch(`${API}/api/servers/${server.id}/settings`, {
        method:'PUT', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({
          name: form.name, port: parseInt(form.port),
          ram_max_mb: parseInt(form.ram_max_mb),
          motd: form.motd, max_players: parseInt(form.max_players),
          gamemode: form.gamemode, difficulty: form.difficulty,
          auto_restart: form.auto_restart, jar: form.jar,
          java_flags: form.java_flags.trim() ? form.java_flags.trim().split(/\s+/) : [],
        }),
      })
      const data = await res.json()
      if (!res.ok) { setErr(data.error); return }
      setSaved(true); setTimeout(()=>setSaved(false),2000); onSaved()
    } catch(e: unknown) { setErr(e instanceof Error ? e.message : 'Failed') }
  }

  const inp: React.CSSProperties = { width:'100%',padding:'7px 10px',background:'#1a1a1a',
    border:'1px solid #2a2a2a',borderRadius:6,color:'#e0e0e0',fontSize:13,outline:'none' }
  const lbl: React.CSSProperties = { fontSize:11,color:'#666',display:'block',marginBottom:3,fontWeight:500 }
  const row: React.CSSProperties = { display:'grid',gridTemplateColumns:'1fr 1fr',gap:10 }

  return (
    <div style={{ padding:'16px 24px', overflowY:'auto', maxHeight:'100%' }}>
      <div style={{ maxWidth:560 }}>
        <h3 style={{ color:'#e0e0e0',fontSize:15,fontWeight:600,marginBottom:16 }}>Server Settings</h3>

        <div style={{ display:'grid',gap:12 }}>
          <div style={row}>
            <div><label style={lbl}>Name</label><input style={inp} value={form.name} onChange={set('name')} /></div>
            <div><label style={lbl}>Port</label><input style={inp} type="number" value={form.port} onChange={set('port')} /></div>
          </div>
          <div style={row}>
            <div><label style={lbl}>Jar filename</label><input style={inp} value={form.jar} onChange={set('jar')} /></div>
            <div><label style={lbl}>Max RAM (MB)</label><input style={inp} type="number" value={form.ram_max_mb} onChange={set('ram_max_mb')} /></div>
          </div>
          <div><label style={lbl}>MOTD</label><input style={inp} value={form.motd} onChange={set('motd')} placeholder="A Minecraft Server" /></div>
          <div style={row}>
            <div><label style={lbl}>Max Players</label><input style={inp} type="number" value={form.max_players} onChange={set('max_players')} /></div>
            <div><label style={lbl}>Gamemode</label>
              <select style={inp} value={form.gamemode} onChange={set('gamemode')}>
                <option>survival</option><option>creative</option><option>adventure</option><option>spectator</option>
              </select></div>
          </div>
          <div style={row}>
            <div><label style={lbl}>Difficulty</label>
              <select style={inp} value={form.difficulty} onChange={set('difficulty')}>
                <option>peaceful</option><option>easy</option><option>normal</option><option>hard</option>
              </select></div>
            <div style={{ display:'flex',alignItems:'center',gap:8,paddingTop:18 }}>
              <input type="checkbox" id="ar" checked={form.auto_restart} onChange={setCheck('auto_restart')} />
              <label htmlFor="ar" style={{ color:'#aaa',fontSize:13,cursor:'pointer' }}>Auto-restart on crash</label>
            </div>
          </div>
          <div><label style={lbl}>Extra JVM Flags</label>
            <input style={inp} value={form.java_flags} onChange={set('java_flags')} placeholder="-XX:+UseG1GC -XX:MaxGCPauseMillis=200" /></div>
          <div style={{ color:'#555',fontSize:11 }}>Server folder: <code style={{color:'#aaa'}}>{server.dir}</code></div>
        </div>

        {err && <div style={{ color:'#f87171',fontSize:13,marginTop:10 }}>✗ {err}</div>}
        <button onClick={save} style={{ marginTop:16,...btn('#166534','#22c55e') }}>
          {saved ? '✔ Saved!' : 'Save Settings'}
        </button>
      </div>
    </div>
  )
}

// ---- Plugins Panel ----
function PluginsPanel({ server }: { server: ServerEntry }) {
  const [plugins, setPlugins] = useState<Plugin[]>([])
  const [uploading, setUploading] = useState(false)
  const [err, setErr] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  const load = useCallback(async () => {
    const r = await fetch(`${API}/api/servers/${server.id}/plugins`)
    const d = await r.json()
    setPlugins(Array.isArray(d) ? d : [])
  }, [server.id])

  useEffect(() => { load() }, [load])

  async function upload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]; if (!file) return
    setUploading(true); setErr('')
    try {
      const fd = new FormData(); fd.append('plugin', file)
      const res = await fetch(`${API}/api/servers/${server.id}/plugins`, { method:'POST', body:fd })
      const d = await res.json()
      if (!res.ok) { setErr(d.error); return }
      load()
    } catch(e: unknown) { setErr(e instanceof Error ? e.message : 'Upload failed') }
    finally { setUploading(false); if(fileRef.current) fileRef.current.value='' }
  }

  async function del(name: string) {
    if (!confirm(`Delete ${name}?`)) return
    await fetch(`${API}/api/servers/${server.id}/plugins/${name}`, { method:'DELETE' })
    load()
  }

  return (
    <div style={{ padding:'16px 24px' }}>
      <div style={{ display:'flex',justifyContent:'space-between',alignItems:'center',marginBottom:16 }}>
        <h3 style={{ color:'#e0e0e0',fontSize:15,fontWeight:600 }}>Plugins ({plugins.length})</h3>
        <label style={{ ...btn('#1e1b4b','#818cf8'),cursor:'pointer',display:'inline-block' }}>
          {uploading ? 'Uploading...' : '+ Upload .jar'}
          <input ref={fileRef} type="file" accept=".jar" style={{ display:'none' }} onChange={upload} />
        </label>
      </div>
      {err && <div style={{ color:'#f87171',fontSize:13,marginBottom:10 }}>✗ {err}</div>}
      {plugins.length === 0 ? (
        <div style={{ color:'#444',fontSize:13,padding:'24px 0',textAlign:'center' }}>
          No plugins yet. Upload a .jar file to get started.
        </div>
      ) : (
        <div style={{ display:'flex',flexDirection:'column',gap:6 }}>
          {plugins.map(p=>(
            <div key={p.name} style={{ display:'flex',alignItems:'center',justifyContent:'space-between',
              padding:'8px 12px',background:'#1a1a1a',borderRadius:6,border:'1px solid #2a2a2a' }}>
              <div>
                <div style={{ color:'#e0e0e0',fontSize:13,fontWeight:500 }}>🔌 {p.name}</div>
                <div style={{ color:'#555',fontSize:11 }}>{fmtSize(p.size)}</div>
              </div>
              <button onClick={()=>del(p.name)} style={btn('#7f1d1d','#f87171',true)}>Delete</button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ---- Main ----
export default function Home() {
  const [servers, setServers] = useState<ServerEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [connErr, setConnErr] = useState<string|null>(null)
  const [selected, setSelected] = useState<string|null>(null)
  const [tab, setTab] = useState<'console'|'settings'|'plugins'>('console')
  const [logs, setLogs] = useState<string[]>([])
  const [cmd, setCmd] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const sseRef = useRef<EventSource|null>(null)

  const fetchServers = useCallback(async () => {
    try {
      const r = await fetch(`${API}/api/servers`)
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setServers(Array.isArray(data) ? data : [])
      setConnErr(null)
    } catch(e: unknown) {
      setConnErr(e instanceof Error ? e.message : 'Connection failed')
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchServers(); const t=setInterval(fetchServers,3000); return ()=>clearInterval(t) }, [fetchServers])
  useEffect(() => { logsEndRef.current?.scrollIntoView({behavior:'smooth'}) }, [logs])

  function selectServer(id: string) {
    if (!id) return
    setSelected(id); setLogs([]); setTab('console')
    sseRef.current?.close()
    const es = new EventSource(`${API}/ws/logs/${id}`)
    es.onmessage = e => setLogs(prev=>[...prev.slice(-500), e.data])
    sseRef.current = es
  }

  async function action(id: string, act: string) {
    await fetch(`${API}/api/servers/${id}/${act}`, {method:'POST'})
    setTimeout(fetchServers, 500)
  }

  async function sendCmd(id: string) {
    if (!cmd.trim()) return
    await fetch(`${API}/api/servers/${id}/command`, {
      method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({cmd}),
    })
    setCmd('')
  }

  const sel = servers.find(s=>s.id===selected) ?? null

  const cpu = sel?.cpu ?? 0
  const ramUsed = sel?.ram_used ?? 0
  const ramMax = sel?.ram_max ?? 0

  const stats: [string, string][] = sel ? [
    ['RAM', `${fmtRam(ramUsed)} / ${fmtRam(ramMax)}`],
    ['CPU', `${cpu.toFixed(1)}%`],
    ['Uptime', sel.uptime || '—'],
    ['PID', sel.pid ? String(sel.pid) : '—'],
  ] : []

  const actionBtns = sel ? [
    sel.status!=='running'&&sel.status!=='starting' ? { key:'start', label:'▶ Start', bg:'#166534', color:'#22c55e' } : null,
    sel.status==='running'||sel.status==='starting' ? { key:'restart', label:'↺ Restart', bg:'#78350f', color:'#f59e0b' } : null,
    sel.status==='running'||sel.status==='starting' ? { key:'stop', label:'■ Stop', bg:'#7f1d1d', color:'#f87171' } : null,
  ].filter(Boolean) as { key: string; label: string; bg: string; color: string }[] : []

  const tabStyle = (t: string): React.CSSProperties => ({
    padding:'8px 16px', cursor:'pointer', fontSize:13, fontWeight:500,
    border:'none', background:'none',
    color: tab===t ? '#e0e0e0' : '#555',
    borderBottom: tab===t ? '2px solid #dc2626' : '2px solid transparent',
  })

  return (
    <div style={{ minHeight:'100vh',background:'#0a0a0a',color:'#e0e0e0',
      fontFamily:"'Inter',system-ui,sans-serif",display:'flex',flexDirection:'column' }}>

      {/* Header */}
      <header style={{ borderBottom:'1px solid #1e1e1e',padding:'14px 24px',
        display:'flex',alignItems:'center',justifyContent:'space-between',background:'#0d0d0d' }}>
        <div style={{ display:'flex',alignItems:'center',gap:10 }}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
            <rect x="2" y="2" width="9" height="9" rx="1.5" fill="#dc2626"/>
            <rect x="13" y="2" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".6"/>
            <rect x="2" y="13" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".6"/>
            <rect x="13" y="13" width="9" height="9" rx="1.5" fill="#dc2626" opacity=".3"/>
          </svg>
          <span style={{ fontWeight:700,fontSize:16,letterSpacing:'-0.02em' }}>opd panel</span>
        </div>
        <button onClick={()=>setShowAdd(true)} style={{ background:'#7f1d1d',color:'#fff',border:'none',
          borderRadius:6,padding:'7px 14px',cursor:'pointer',fontSize:13,fontWeight:600,
          display:'flex',alignItems:'center',gap:6 }}>
          <span style={{ fontSize:18,lineHeight:1 }}>+</span> Add Server
        </button>
      </header>

      <div style={{ display:'flex',flex:1,overflow:'hidden' }}>
        {/* Sidebar */}
        <aside style={{ width:260,borderRight:'1px solid #1e1e1e',overflowY:'auto',
          background:'#0d0d0d',display:'flex',flexDirection:'column' }}>
          <div style={{ padding:'12px 16px 8px',fontSize:11,color:'#555',fontWeight:600,
            letterSpacing:'0.08em',textTransform:'uppercase' }}>
            Servers {servers.length>0&&`(${servers.length})`}
          </div>
          {loading && <div style={{ padding:'20px 16px',color:'#555',fontSize:13 }}>Connecting...</div>}
          {connErr && <div style={{ padding:'12px 16px',color:'#f87171',fontSize:12 }}>✗ {connErr}<br/><span style={{color:'#555'}}>Is daemon running?</span></div>}
          {!loading&&!connErr&&servers.length===0&&(
            <div style={{ padding:'16px',color:'#555',fontSize:13 }}>
              No servers yet.<br/>
              <button onClick={()=>setShowAdd(true)} style={{ marginTop:8,color:'#dc2626',background:'none',border:'none',cursor:'pointer',fontSize:13,padding:0 }}>+ Add your first server</button>
            </div>
          )}
          {servers.map(srv=>(
            <button key={srv.id} onClick={()=>selectServer(srv.id)} style={{
              display:'flex',alignItems:'center',width:'100%',padding:'10px 16px',
              background:selected===srv.id?'#1a1a1a':'none',border:'none',
              borderLeft:selected===srv.id?'2px solid #dc2626':'2px solid transparent',
              cursor:'pointer',textAlign:'left',transition:'background 0.1s',
            }}>
              <StatusDot status={srv.status}/>
              <div style={{ overflow:'hidden' }}>
                <div style={{ color:'#e0e0e0',fontSize:14,fontWeight:500,whiteSpace:'nowrap',overflow:'hidden',textOverflow:'ellipsis' }}>{srv.name}</div>
                <div style={{ color:'#555',fontSize:11 }}>{srv.id} · :{srv.port}</div>
              </div>
            </button>
          ))}
        </aside>

        {/* Main */}
        <main style={{ flex:1,overflow:'hidden',display:'flex',flexDirection:'column' }}>
          {!sel ? (
            <div style={{ flex:1,display:'flex',alignItems:'center',justifyContent:'center',color:'#333',flexDirection:'column',gap:8 }}>
              <span style={{ fontSize:40 }}>⬛</span>
              <span style={{ fontSize:14 }}>Select a server</span>
            </div>
          ) : (
            <div style={{ display:'flex',flexDirection:'column',height:'100%' }}>
              {/* Server Header */}
              <div style={{ padding:'14px 24px',borderBottom:'1px solid #1e1e1e',
                display:'flex',alignItems:'center',justifyContent:'space-between',flexShrink:0 }}>
                <div style={{ display:'flex',alignItems:'center',gap:10 }}>
                  <StatusDot status={sel.status}/>
                  <div>
                    <div style={{ fontWeight:600,fontSize:16 }}>{sel.name}</div>
                    <div style={{ color:'#555',fontSize:12 }}>{sel.id} · :{sel.port} · {sel.status}</div>
                  </div>
                </div>
                <div style={{ display:'flex',gap:8 }}>
                  {actionBtns.map(b=>(
                    <button key={b.key} onClick={()=>action(sel.id, b.key)} style={btn(b.bg, b.color)}>
                      {b.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Stats bar */}
              <div style={{ display:'grid',gridTemplateColumns:'repeat(4,1fr)',gap:1,
                borderBottom:'1px solid #1e1e1e',background:'#1e1e1e',flexShrink:0 }}>
                {stats.map(([label, value])=>(
                  <div key={label} style={{ background:'#0d0d0d',padding:'10px 16px' }}>
                    <div style={{ color:'#555',fontSize:11,marginBottom:3 }}>{label}</div>
                    <div style={{ color:'#e0e0e0',fontSize:14,fontWeight:600,fontVariantNumeric:'tabular-nums' }}>{value}</div>
                  </div>
                ))}
              </div>

              {/* Tabs */}
              <div style={{ display:'flex',borderBottom:'1px solid #1e1e1e',flexShrink:0 }}>
                {(['console','settings','plugins'] as const).map(t=>(
                  <button key={t} onClick={()=>setTab(t)} style={tabStyle(t)}>
                    {t.charAt(0).toUpperCase()+t.slice(1)}
                  </button>
                ))}
              </div>

              {/* Tab content */}
              <div style={{ flex:1,overflow:'auto',display:'flex',flexDirection:'column' }}>
                {tab==='console' && (
                  <>
                    <div style={{ flex:1,overflowY:'auto',background:'#060606',
                      fontFamily:'monospace',fontSize:12,padding:'12px 16px',lineHeight:1.6 }}>
                      {logs.length===0 ? (
                        <span style={{ color:'#333' }}>No logs yet.</span>
                      ) : logs.map((l,i)=>(
                        <div key={i} style={{ color:l.includes('ERROR')||l.includes('WARN')?'#f87171':'#9ca3af' }}>{l}</div>
                      ))}
                      <div ref={logsEndRef}/>
                    </div>
                    {sel.status==='running'&&(
                      <div style={{ padding:'10px 16px',borderTop:'1px solid #1e1e1e',display:'flex',gap:8,flexShrink:0 }}>
                        <span style={{ color:'#555',fontFamily:'monospace',fontSize:13,alignSelf:'center' }}>{'>'}</span>
                        <input value={cmd} onChange={e=>setCmd(e.target.value)}
                          onKeyDown={e=>e.key==='Enter'&&sendCmd(sel.id)}
                          placeholder="say Hello!"
                          style={{ flex:1,background:'none',border:'none',outline:'none',color:'#e0e0e0',fontFamily:'monospace',fontSize:13 }}/>
                        <button onClick={()=>sendCmd(sel.id)} style={btn('#1e1e1e','#555')}>Send</button>
                      </div>
                    )}
                  </>
                )}
                {tab==='settings' && <SettingsPanel server={sel} onSaved={fetchServers}/>}
                {tab==='plugins' && <PluginsPanel server={sel}/>}
              </div>
            </div>
          )}
        </main>
      </div>

      {showAdd&&<AddServerModal onClose={()=>setShowAdd(false)} onCreated={()=>{fetchServers();setShowAdd(false)}}/>}
    </div>
  )
}
