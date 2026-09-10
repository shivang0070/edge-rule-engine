import { useState, useEffect } from 'react';
import { Activity, ShieldCheck, PauseCircle, PlayCircle, Clock, Trash2, Edit, X, Plus, Code } from 'lucide-react';

const STATE_VARIABLES = [
  { id: 'occupancy.total', label: 'Total Occupancy' },
  { id: 'occupancy.person', label: 'Person Occupancy' },
  { id: 'occupancy.vehicle', label: 'Vehicle Occupancy' },
  { id: 'staffCount', label: 'Staff Count' },
  { id: 'dwell.person.avg', label: 'Avg Person Dwell (s)' },
  { id: 'dwell.person.min', label: 'Min Person Dwell (s)' },
  { id: 'dwell.person.max', label: 'Max Person Dwell (s)' },
  { id: 'dwell.vehicle.avg', label: 'Avg Vehicle Dwell (s)' },
  { id: 'dwell.vehicle.min', label: 'Min Vehicle Dwell (s)' },
  { id: 'dwell.vehicle.max', label: 'Max Vehicle Dwell (s)' },
  { id: 'window.uniquePersons', label: 'Unique Persons (Rolling Window)' },
  { id: 'window.uniqueVehicles', label: 'Unique Vehicles (Rolling Window)' },
];

const EVENT_VARIABLES = [
  { id: 'window.entranceCount', label: 'Entrance Count' },
  { id: 'window.exitCount', label: 'Exit Count' },
  { id: 'window.count', label: 'Total Events' },
];

const OPERATORS = [
  { id: '>', label: 'Greater Than (>)' },
  { id: '>=', label: 'Greater or Equal (>=)' },
  { id: '<', label: 'Less Than (<)' },
  { id: '<=', label: 'Less or Equal (<=)' },
  { id: '==', label: 'Equals (==)' },
  { id: '!=', label: 'Not Equals (!=)' },
];

export default function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [health, setHealth] = useState<any>(null);
  const [rules, setRules] = useState<any[]>([]);
  const [executions, setExecutions] = useState<any[]>([]);
  const [syslogs, setSyslogs] = useState<any[]>([]);
  
  // Modal state
  const [editingRule, setEditingRule] = useState<any>(null);
  const [builderConditions, setBuilderConditions] = useState<{variable:string, operator:string, value:string}[] | null>(null);
  const [forceRaw, setForceRaw] = useState(false);

  useEffect(() => {
    fetchHealth();
    fetchRules();
    fetchExecutions();

    const evtSource = new EventSource('/api/v1/stream');
    evtSource.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.event === 'execution') {
          setExecutions((prev: any) => [parsed.data, ...prev].slice(0, 100));
        } else if (parsed.event === 'syslog') {
          setSyslogs((prev: any) => [parsed.data, ...prev].slice(0, 300));
        }
      } catch (e) {}
    };
    return () => evtSource.close();
  }, []);

  const fetchHealth = () => fetch('/health').then(r => r.json()).then(setHealth).catch(console.error);
  const fetchRules = () => fetch('/api/v1/rules').then(r => r.json()).then(setRules).catch(console.error);
  const fetchExecutions = () => fetch('/api/v1/executions').then(r => r.json()).then(setExecutions).catch(console.error);

  const toggleRule = async (rule: any) => {
    const updated = { ...rule, enabled: !rule.enabled };
    await fetch(`/api/v1/rules/${rule.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updated)
    });
    fetchRules();
  };

  const deleteRule = async (id: string) => {
    if (!window.confirm("Are you sure you want to permanently delete this rule?")) return;
    await fetch(`/api/v1/rules/${id}`, { method: 'DELETE' });
    fetchRules();
  };

  // Smart parser to reverse-engineer Expr strings into UI components
  const parseExpression = (expr: string) => {
    if (!expr) return [{ variable: 'occupancy.person', operator: '>', value: '3' }];
    const conditions = [];
    const parts = expr.split('&&');
    for (let p of parts) {
      const match = p.trim().match(/^([\w.]+)\s*(>=|<=|>|<|==|!=)\s*([\d.]+)$/);
      if (match) {
        conditions.push({ variable: match[1], operator: match[2], value: match[3] });
      } else {
        return null; // Expression is too complex for visual builder
      }
    }
    return conditions.length ? conditions : null;
  };

  const openEditModal = (rule: any) => {
    setEditingRule(JSON.parse(JSON.stringify(rule)));
    setForceRaw(false);
    
    const parsed = parseExpression(rule.condition.expression);
    if (parsed) {
      setBuilderConditions(parsed);
    } else {
      setBuilderConditions(null);
    }
  };

  const updateBuilderCondition = (index: number, field: string, val: string) => {
    if (!builderConditions) return;
    const newConds = [...builderConditions];
    newConds[index] = { ...newConds[index], [field]: val };
    setBuilderConditions(newConds);
    
    // Sync to underlying expression instantly
    const expr = newConds.map(c => `${c.variable} ${c.operator} ${c.value}`).join(' && ');
    setEditingRule((prev: any) => ({...prev, condition: {...prev.condition, expression: expr}}));
  };

  const addBuilderCondition = () => {
    if (!builderConditions) return;
    const newConds = [...builderConditions, { variable: 'staffCount', operator: '==', value: '0' }];
    setBuilderConditions(newConds);
    const expr = newConds.map(c => `${c.variable} ${c.operator} ${c.value}`).join(' && ');
    setEditingRule((prev: any) => ({...prev, condition: {...prev.condition, expression: expr}}));
  };

  const removeBuilderCondition = (index: number) => {
    if (!builderConditions || builderConditions.length <= 1) return;
    const newConds = builderConditions.filter((_, i) => i !== index);
    setBuilderConditions(newConds);
    const expr = newConds.map(c => `${c.variable} ${c.operator} ${c.value}`).join(' && ');
    setEditingRule((prev: any) => ({...prev, condition: {...prev.condition, expression: expr}}));
  };

  const submitEdit = async () => {
    try {
      const payload = { ...editingRule };
      if (payload.window && !payload.window.durationSeconds) delete payload.window;
      if (payload.sustain && !payload.sustain.durationSeconds) delete payload.sustain;

      const res = await fetch(`/api/v1/rules/${payload.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      if (!res.ok) throw new Error("Server rejected the update");
      setEditingRule(null);
      fetchRules();
    } catch (e: any) {
      alert("Failed to update rule: " + e.message);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col font-sans">
      <header className="bg-white border-b px-6 py-4 flex items-center justify-between shadow-sm z-10 relative">
        <div className="flex items-center gap-2">
          <Activity className="text-blue-600" />
          <h1 className="text-xl font-bold text-slate-800">Edge Rule Engine</h1>
        </div>
        {health && (
          <div className="flex gap-4 text-sm font-medium text-slate-500">
            <span className="flex items-center gap-1"><ShieldCheck size={16} className="text-green-500"/> {health.status.toUpperCase()}</span>
            <span>Uptime: {health.uptime}</span>
          </div>
        )}
      </header>

      <main className="flex-1 p-6 max-w-7xl w-full mx-auto relative">
        <div className="flex gap-4 mb-6">
          <button onClick={() => setActiveTab('dashboard')} className={`px-4 py-2 font-medium rounded-md transition-colors ${activeTab === 'dashboard' ? 'bg-blue-600 text-white shadow-sm' : 'bg-white text-slate-600 border hover:bg-slate-50'}`}>Dashboard & Live</button>
          <button onClick={() => setActiveTab('rules')} className={`px-4 py-2 font-medium rounded-md transition-colors ${activeTab === 'rules' ? 'bg-blue-600 text-white shadow-sm' : 'bg-white text-slate-600 border hover:bg-slate-50'}`}>Active Rules</button>
          <button onClick={() => setActiveTab('history')} className={`px-4 py-2 font-medium rounded-md transition-colors ${activeTab === 'history' ? 'bg-blue-600 text-white shadow-sm' : 'bg-white text-slate-600 border hover:bg-slate-50'}`}>Execution History</button>
        </div>

        {/* --- TABS CODE OMITTED FOR BREVITY, IDENTICAL TO BEFORE --- */}
        {activeTab === 'dashboard' && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="col-span-1 bg-white p-6 rounded-lg shadow-sm border flex flex-col gap-6">
              <div>
                <h2 className="text-lg font-bold mb-4 text-slate-800">System Health</h2>
                <div className="space-y-4">
                  <div>
                    <div className="text-sm text-slate-500 font-medium">Active Rules Loaded</div>
                    <div className="text-4xl font-light text-slate-800">{health?.activeRules || 0}</div>
                  </div>
                  <div>
                    <div className="text-sm text-slate-500 font-medium">Recent Executions Tracked</div>
                    <div className="text-4xl font-light text-slate-800">{executions.length}</div>
                  </div>
                </div>
              </div>
            </div>
            <div className="col-span-2 bg-slate-900 rounded-lg shadow-sm border p-4 text-emerald-400 font-mono text-sm h-[500px] flex flex-col">
              <div className="mb-4 text-slate-400 border-b border-slate-700 pb-2 flex justify-between items-center">
                <span># Live System Terminal</span>
                <span className="text-xs text-slate-500 bg-slate-800 px-2 py-1 rounded">{syslogs.length} events</span>
              </div>
              <div className="flex-1 overflow-y-auto">
                <div className="flex flex-col-reverse">
                  {syslogs.map((log, i) => (
                    <div key={i} className="mb-1.5 leading-tight text-xs flex gap-2 hover:bg-slate-800/50 p-0.5 rounded">
                      <span className="text-slate-500 shrink-0">[{new Date(log.ts * 1000).toISOString().split('T')[1].replace('Z', '')}]</span>
                      <span className={`shrink-0 w-10 ${log.level === 'error' ? 'text-red-400 font-bold' : log.level === 'warn' ? 'text-amber-400 font-bold' : log.level === 'debug' ? 'text-slate-500' : 'text-blue-400 font-bold'}`}>
                        {log.level?.toUpperCase()}
                      </span>
                      <div className="break-all">
                        <span className={log.msg?.includes('TRIGGERED') ? 'text-emerald-400 font-bold' : 'text-slate-300'}>{log.msg}</span>{' '}
                        <span className="text-slate-500">
                          {Object.entries(log).filter(([k]) => !['ts', 'level', 'msg'].includes(k)).map(([k,v]) => `${k}=${v}`).join(' ')}
                        </span>
                      </div>
                    </div>
                  ))}
                  {syslogs.length === 0 && <div className="text-slate-500 text-center mt-10 italic">Waiting for incoming logs...</div>}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'rules' && (
          <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
            <table className="w-full text-left">
              <thead className="bg-slate-50 border-b">
                <tr>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Rule Name</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Target (Cam / ROI)</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Source</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Condition</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rules.map(rule => (
                  <tr key={rule.id} className={rule.enabled ? 'hover:bg-slate-50 transition-colors' : 'bg-slate-50 opacity-70'}>
                    <td className="p-4 font-medium text-slate-800">{rule.name} <div className="text-xs font-normal text-slate-400 mt-1">{rule.id}</div></td>
                    <td className="p-4 text-sm text-slate-600">
                      <div className="font-medium text-slate-800">{rule.scope.cameraId}</div>
                      <div className="text-xs">{rule.scope.roiId || <span className="italic text-slate-400">Any ROI</span>}</div>
                    </td>
                    <td className="p-4"><span className="px-2 py-1 bg-slate-100 border text-slate-600 rounded text-xs font-bold uppercase tracking-wider">{rule.source}</span></td>
                    <td className="p-4">
                      <div className="font-mono text-sm text-blue-600 max-w-xs truncate" title={rule.condition.expression}>{rule.condition.expression}</div>
                      {rule.sustain?.durationSeconds > 0 && (
                        <div className="text-xs font-semibold text-amber-600 mt-1 bg-amber-50 inline-block px-1.5 py-0.5 rounded border border-amber-200">
                          sustained continuously for {rule.sustain.durationSeconds}s
                        </div>
                      )}
                      {rule.window?.durationSeconds > 0 && (
                        <div className="text-xs font-semibold text-purple-600 mt-1 bg-purple-50 inline-block px-1.5 py-0.5 rounded border border-purple-200">
                          rolling {rule.window.durationSeconds}s window
                        </div>
                      )}
                    </td>
                    <td className="p-4 text-right space-x-2">
                      <button onClick={() => toggleRule(rule)} className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${rule.enabled ? 'bg-amber-100 text-amber-700 hover:bg-amber-200' : 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200'}`}>
                        {rule.enabled ? <><PauseCircle size={16}/> Pause</> : <><PlayCircle size={16}/> Resume</>}
                      </button>
                      <button onClick={() => openEditModal(rule)} className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium bg-slate-100 text-slate-700 hover:bg-slate-200 transition-colors">
                        <Edit size={16}/> Edit
                      </button>
                      <button onClick={() => deleteRule(rule.id)} className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium bg-red-100 text-red-700 hover:bg-red-200 transition-colors">
                        <Trash2 size={16}/>
                      </button>
                    </td>
                  </tr>
                ))}
                {rules.length === 0 && <tr><td colSpan={5} className="p-8 text-center text-slate-500">No rules loaded.</td></tr>}
              </tbody>
            </table>
          </div>
        )}

        {activeTab === 'history' && (
          <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
            <table className="w-full text-left">
              <thead className="bg-slate-50 border-b">
                <tr>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Date & Time</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Rule ID</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Status</th>
                  <th className="p-4 font-semibold text-slate-600 text-sm">Context / Reason</th>
                </tr>
              </thead>
              <tbody className="divide-y text-sm">
                {executions.map(ex => (
                  <tr key={ex.id} className="hover:bg-slate-50 transition-colors">
                    <td className="p-4 text-slate-600 whitespace-nowrap"><Clock size={14} className="inline mr-1.5 text-slate-400"/>{new Date(ex.triggeredAt * 1000).toLocaleString()}</td>
                    <td className="p-4 font-medium text-slate-800">{ex.ruleId}</td>
                    <td className="p-4">
                      <span className={`px-2 py-1 rounded border text-xs font-bold tracking-wider ${ex.status === 'success' ? 'bg-green-50 border-green-200 text-green-700' : 'bg-red-50 border-red-200 text-red-700'}`}>
                        {ex.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="p-4 font-mono text-xs text-slate-600 truncate max-w-sm">{ex.error || ex.context}</td>
                  </tr>
                ))}
                {executions.length === 0 && <tr><td colSpan={4} className="p-8 text-center text-slate-500">No executions yet.</td></tr>}
              </tbody>
            </table>
          </div>
        )}
      </main>

      {/* Edit Rule Modal */}
      {editingRule && (
        <div className="fixed inset-0 bg-slate-900/60 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
          <div className="bg-white rounded-xl shadow-2xl w-full max-w-3xl overflow-hidden flex flex-col max-h-[90vh]">
            <div className="p-4 border-b bg-slate-50 flex justify-between items-center">
              <h3 className="font-bold text-lg text-slate-800">Edit Rule: {editingRule.name}</h3>
              <button onClick={() => setEditingRule(null)} className="text-slate-400 hover:text-slate-700"><X size={20}/></button>
            </div>
            
            <div className="p-6 overflow-y-auto flex-1 space-y-6">
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-1">Rule Name</label>
                  <input type="text" value={editingRule.name} onChange={e => setEditingRule({...editingRule, name: e.target.value})} className="w-full border rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"/>
                </div>
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-1">Data Source</label>
                  <select value={editingRule.source} onChange={e => setEditingRule({...editingRule, source: e.target.value})} className="w-full border rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none">
                    <option value="state">State (Polling)</option>
                    <option value="event">Event (Discrete)</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-1">Target Camera ID</label>
                  <input type="text" value={editingRule.scope.cameraId} onChange={e => setEditingRule({...editingRule, scope: {...editingRule.scope, cameraId: e.target.value}})} className="w-full border rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"/>
                </div>
                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-1">Target ROI ID <span className="text-slate-400 font-normal">(Optional)</span></label>
                  <input type="text" value={editingRule.scope.roiId || ''} onChange={e => setEditingRule({...editingRule, scope: {...editingRule.scope, roiId: e.target.value}})} className="w-full border rounded-md px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" placeholder="Leave blank for any ROI"/>
                </div>
              </div>

              <hr className="border-slate-100" />

              {/* No Code Logic Builder */}
              <div>
                <div className="flex justify-between items-end mb-2">
                  <label className="block text-sm font-bold text-slate-800">Trigger Conditions (No-Code Builder)</label>
                  <button onClick={() => setForceRaw(!forceRaw)} className="text-xs flex items-center gap-1 text-blue-600 hover:text-blue-800">
                    <Code size={14}/> {forceRaw || !builderConditions ? 'Use Visual Builder' : 'Use Raw Expr Mode'}
                  </button>
                </div>
                
                {builderConditions && !forceRaw ? (
                  <div className="space-y-3 bg-blue-50 p-4 rounded-lg border border-blue-100">
                    {builderConditions.map((cond, i) => (
                      <div key={i} className="flex items-center gap-2">
                        {i > 0 && <span className="font-bold text-blue-800 text-xs mr-1 w-6">AND</span>}
                        {i === 0 && <span className="font-bold text-blue-800 text-xs mr-1 w-6">IF</span>}
                        <select value={cond.variable} onChange={(e) => updateBuilderCondition(i, 'variable', e.target.value)} className="flex-1 border border-slate-300 rounded px-2 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500">
                          {(editingRule.source === 'event' ? EVENT_VARIABLES : STATE_VARIABLES).map(v => <option key={v.id} value={v.id}>{v.label} ({v.id})</option>)}
                        </select>
                        <select value={cond.operator} onChange={(e) => updateBuilderCondition(i, 'operator', e.target.value)} className="w-32 border border-slate-300 rounded px-2 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500">
                          {OPERATORS.map(o => <option key={o.id} value={o.id}>{o.label}</option>)}
                        </select>
                        <input type="text" value={cond.value} onChange={(e) => updateBuilderCondition(i, 'value', e.target.value)} className="w-24 border border-slate-300 rounded px-2 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-blue-500 text-center font-mono" placeholder="Value"/>
                        {builderConditions.length > 1 && (
                          <button onClick={() => removeBuilderCondition(i)} className="text-slate-400 hover:text-red-500 ml-1"><X size={18}/></button>
                        )}
                      </div>
                    ))}
                    <button onClick={addBuilderCondition} className="mt-2 text-sm flex items-center gap-1 text-blue-700 font-medium hover:text-blue-900 bg-white px-2 py-1 rounded border border-blue-200 shadow-sm">
                      <Plus size={16}/> Add AND Condition
                    </button>
                    <div className="mt-3 pt-3 border-t border-blue-200/50 flex items-center gap-2">
                      <span className="text-xs text-blue-600 font-medium">Generated Expression:</span>
                      <code className="text-xs bg-white px-2 py-1 rounded border border-blue-100 text-slate-700">{editingRule.condition.expression}</code>
                    </div>
                  </div>
                ) : (
                  <div className="bg-slate-50 p-4 rounded-lg border border-slate-200">
                    <p className="text-xs text-amber-600 font-medium mb-2">Visual builder unavailable for complex expressions. Using raw mode.</p>
                    <input type="text" value={editingRule.condition.expression} onChange={e => setEditingRule({...editingRule, condition: {...editingRule.condition, expression: e.target.value}})} className="w-full border rounded-md px-3 py-2 text-sm font-mono text-blue-600 focus:ring-2 focus:ring-blue-500 focus:outline-none shadow-inner"/>
                  </div>
                )}
              </div>

              <div className={`grid gap-4 bg-slate-50 p-4 rounded-lg border border-slate-100 ${editingRule.source === 'event' || editingRule.condition.expression.includes('window.unique') ? (editingRule.source === 'state' ? 'grid-cols-3' : 'grid-cols-2') : 'grid-cols-2'}`}>
                
                {(editingRule.source === 'event' || editingRule.condition.expression.includes('window.unique')) && (
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-1">Rolling Window (Secs)</label>
                    <input type="number" value={editingRule.window?.durationSeconds || ''} onChange={e => setEditingRule({...editingRule, window: { durationSeconds: parseInt(e.target.value) || undefined }})} className="w-full border rounded-md px-3 py-2 text-sm shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" placeholder="e.g. 120 for 2 mins"/>
                    <p className="text-[10px] text-slate-400 mt-1">Accumulates data over this duration</p>
                  </div>
                )}

                {editingRule.source === 'state' && (
                  <div>
                    <label className="block text-sm font-semibold text-slate-700 mb-1">Sustain Duration (Secs)</label>
                    <input type="number" value={editingRule.sustain?.durationSeconds || ''} onChange={e => setEditingRule({...editingRule, sustain: { durationSeconds: parseInt(e.target.value) || undefined }})} className="w-full border rounded-md px-3 py-2 text-sm shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none" placeholder="e.g. 180 for 3 mins"/>
                    <p className="text-[10px] text-slate-400 mt-1">Must remain true continuously</p>
                  </div>
                )}

                <div>
                  <label className="block text-sm font-semibold text-slate-700 mb-1">Trigger Cooldown (Secs)</label>
                  <input type="number" value={editingRule.trigger.cooldownSeconds} onChange={e => setEditingRule({...editingRule, trigger: {...editingRule.trigger, cooldownSeconds: parseInt(e.target.value) || 0}})} className="w-full border rounded-md px-3 py-2 text-sm shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"/>
                  <p className="text-[10px] text-slate-400 mt-1">Silences duplicate alerts</p>
                </div>
              </div>

              <div>
                <label className="block text-sm font-semibold text-slate-700 mb-1">Action Payload Type</label>
                <input type="text" value={editingRule.action.type} onChange={e => setEditingRule({...editingRule, action: {...editingRule.action, type: e.target.value}})} className="w-full border rounded-md px-3 py-2 text-sm shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"/>
              </div>
            </div>

            <div className="p-4 border-t bg-slate-50 flex justify-end gap-3 items-center">
              <div className="flex items-center gap-2 flex-1 ml-2">
                <input type="checkbox" id="enabled-checkbox" checked={editingRule.enabled} onChange={e => setEditingRule({...editingRule, enabled: e.target.checked})} className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"/>
                <label htmlFor="enabled-checkbox" className="text-sm font-medium text-slate-700 cursor-pointer">Rule Active</label>
              </div>
              <button onClick={() => setEditingRule(null)} className="px-5 py-2 border rounded-md text-slate-600 hover:bg-slate-100 font-medium text-sm transition-colors">Cancel</button>
              <button onClick={submitEdit} className="px-5 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium text-sm shadow-sm transition-colors">Save Rule Changes</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
