
import React, { useState, useEffect, useMemo } from 'react';
import { Client, ClientTier, SystemStats, HealthCheckResponse } from './types';
import { TIER_CONFIG, Icons } from './constants';
import { 
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar 
} from 'recharts';

// Mock Data
const INITIAL_CLIENTS: Client[] = [
  {
    id: 'cl_8f237d',
    name: 'NotaryHash Enterprise',
    api_key: 'gh_live_a1b2c3d4e5f6',
    tier: ClientTier.ENTERPRISE,
    max_daily_tx: 100000,
    require_signature: true,
    allowed_ips: ['192.168.1.1'],
    public_key: '04a6e5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6',
    created_at: '2023-10-15T10:00:00Z',
    status: 'active',
    current_day_tx: 45230,
    grace_period_hours: 48
  },
  {
    id: 'cl_23a10b',
    name: 'AKUA Pilot App',
    api_key: 'gh_test_k9j8h7g6f5d4',
    tier: ClientTier.PILOT,
    max_daily_tx: 10000,
    require_signature: false,
    allowed_ips: ['127.0.0.1'],
    created_at: '2023-11-20T14:30:00Z',
    status: 'active',
    current_day_tx: 1205
  }
];

const MOCK_STATS: SystemStats = {
  total_tx_24h: 125430,
  active_clients: 42,
  utxo_count: 5200,
  health_status: 'operational',
  avg_broadcast_latency: 142
};

const CHART_DATA = [
  { time: '00:00', tx: 4500 },
  { time: '04:00', tx: 3200 },
  { time: '08:00', tx: 8900 },
  { time: '12:00', tx: 12500 },
  { time: '16:00', tx: 9800 },
  { time: '20:00', tx: 7400 },
  { time: '23:59', tx: 5100 },
];

const SidebarItem = ({ icon: Icon, label, active, onClick }: { icon: any, label: string, active: boolean, onClick: () => void }) => (
  <button
    onClick={onClick}
    className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg transition-colors ${
      active ? 'bg-indigo-600 text-white shadow-lg' : 'text-slate-400 hover:bg-slate-800 hover:text-white'
    }`}
  >
    <Icon />
    <span className="font-medium">{label}</span>
  </button>
);

const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'clients' | 'health'>('dashboard');
  const [clients, setClients] = useState<Client[]>(INITIAL_CLIENTS);
  const [stats, setStats] = useState<SystemStats>(MOCK_STATS);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [adminPassword, setAdminPassword] = useState('');
  const [isAuthorized, setIsAuthorized] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  // Filtered Clients
  const filteredClients = useMemo(() => {
    return clients.filter(c => 
      c.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
      c.id.includes(searchQuery)
    );
  }, [clients, searchQuery]);

  const handleRegisterClient = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const tier = formData.get('tier') as ClientTier;
    
    const newClient: Client = {
      id: `cl_${Math.random().toString(36).substr(2, 6)}`,
      name: formData.get('name') as string,
      api_key: `gh_${tier}_${Math.random().toString(36).substr(2, 12)}`,
      tier: tier,
      max_daily_tx: Number(formData.get('max_daily_tx')),
      require_signature: TIER_CONFIG[tier].reqSig,
      allowed_ips: (formData.get('allowed_ips') as string).split(',').map(i => i.trim()),
      public_key: formData.get('public_key') as string || undefined,
      created_at: new Date().toISOString(),
      status: 'active',
      current_day_tx: 0,
      grace_period_hours: tier !== ClientTier.PILOT ? 48 : undefined
    };

    setClients([...clients, newClient]);
    setShowAddModal(false);
  };

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    if (adminPassword) {
      setIsAuthorized(true);
    }
  };

  if (!isAuthorized) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 p-4 sm:p-6">
        <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 sm:p-8 shadow-2xl">
          <div className="flex flex-col items-center mb-8">
            <div className="w-16 h-16 bg-indigo-600 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-indigo-500/20">
              <Icons.Security />
            </div>
            <h1 className="text-xl sm:text-2xl font-bold text-white text-center">GovHash Admin</h1>
            <p className="text-slate-400 text-sm mt-1 text-center">Authentication required for AKUA Broadcaster infrastructure</p>
          </div>
          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1">Admin Password</label>
              <input
                type="password"
                value={adminPassword}
                onChange={(e) => setAdminPassword(e.target.value)}
                placeholder="Enter password..."
                className="w-full px-4 py-3 bg-slate-800 border border-slate-700 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                required
              />
            </div>
            <button className="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-3 rounded-xl transition-all">
              Authenticate Access
            </button>
          </form>
          <p className="mt-6 text-[10px] sm:text-xs text-slate-500 text-center leading-relaxed">
            Authorized access only. All actions are logged and subject to government-standard security audits.
          </p>
        </div>
      </div>
    );
  }

  const navigateTo = (tab: 'dashboard' | 'clients' | 'health') => {
    setActiveTab(tab);
    setIsSidebarOpen(false);
  };

  return (
    <div className="flex h-screen bg-slate-50 relative">
      {/* Mobile Sidebar Overlay */}
      {isSidebarOpen && (
        <div 
          className="fixed inset-0 bg-slate-950/50 backdrop-blur-sm z-40 lg:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={`
        fixed inset-y-0 left-0 w-64 bg-slate-950 text-white flex flex-col p-4 z-50 transition-transform duration-300 transform
        ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full'}
        lg:relative lg:translate-x-0
      `}>
        <div className="flex items-center justify-between lg:justify-start space-x-3 px-4 py-6 mb-8 border-b border-slate-800">
          <div className="flex items-center space-x-3">
            <div className="w-8 h-8 bg-indigo-600 rounded flex items-center justify-center">
              <Icons.Security />
            </div>
            <span className="text-xl font-bold tracking-tight">GovHash.org</span>
          </div>
          <button className="lg:hidden text-slate-400" onClick={() => setIsSidebarOpen(false)}>
            ✕
          </button>
        </div>

        <nav className="flex-1 space-y-2">
          <SidebarItem 
            icon={Icons.Dashboard} 
            label="Dashboard" 
            active={activeTab === 'dashboard'} 
            onClick={() => navigateTo('dashboard')} 
          />
          <SidebarItem 
            icon={Icons.Clients} 
            label="Client Mgmt" 
            active={activeTab === 'clients'} 
            onClick={() => navigateTo('clients')} 
          />
          <SidebarItem 
            icon={Icons.Health} 
            label="System Health" 
            active={activeTab === 'health'} 
            onClick={() => navigateTo('health')} 
          />
        </nav>

        <div className="p-4 mt-auto">
          <div className="p-4 bg-slate-900 rounded-xl border border-slate-800">
            <p className="text-xs text-slate-400 mb-1">Infrastructure</p>
            <p className="text-sm font-semibold text-white">BSV Network Mainnet</p>
            <div className="flex items-center mt-2">
              <div className="w-2 h-2 rounded-full bg-emerald-500 mr-2"></div>
              <span className="text-xs text-emerald-500 font-medium">Synced: 825,412</span>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto w-full">
        <header className="h-20 bg-white border-b border-slate-200 px-4 sm:px-8 flex items-center justify-between sticky top-0 z-30 shadow-sm">
          <div className="flex items-center space-x-4">
            <button 
              className="lg:hidden p-2 text-slate-600 hover:bg-slate-100 rounded-lg"
              onClick={() => setIsSidebarOpen(true)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-800 leading-tight">
                {activeTab === 'dashboard' && 'Operations Overview'}
                {activeTab === 'clients' && 'Client Management'}
                {activeTab === 'health' && 'System Infrastructure Health'}
              </h2>
              <p className="hidden sm:block text-sm text-slate-500">Managing global AKUA Broadcaster nodes</p>
            </div>
          </div>
          <div className="flex items-center space-x-3 sm:space-x-4">
            <div className="hidden sm:block text-right">
              <p className="text-sm font-semibold text-slate-800">Admin User</p>
              <p className="text-xs text-slate-500">Security Clearance Level 4</p>
            </div>
            <div className="w-8 h-8 sm:w-10 sm:h-10 bg-slate-100 rounded-full border border-slate-200 overflow-hidden flex items-center justify-center">
              <img src="https://picsum.photos/seed/admin/40/40" alt="Avatar" />
            </div>
          </div>
        </header>

        <div className="p-4 sm:p-8">
          {activeTab === 'dashboard' && (
            <div className="space-y-6 sm:space-y-8">
              {/* Stats Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
                {[
                  { label: '24h Tx Volume', value: stats.total_tx_24h.toLocaleString(), icon: Icons.Key, trend: '+12.5%' },
                  { label: 'Active Clients', value: stats.active_clients, icon: Icons.Clients, trend: '+3' },
                  { label: 'UTXO Pool', value: stats.utxo_count.toLocaleString(), icon: Icons.Settings, trend: 'Optimal' },
                  { label: 'Avg Latency', value: `${stats.avg_broadcast_latency}ms`, icon: Icons.Health, trend: 'Stable' },
                ].map((s, i) => (
                  <div key={i} className="bg-white p-4 sm:p-6 rounded-2xl border border-slate-200 shadow-sm">
                    <div className="flex justify-between items-start mb-4">
                      <div className="p-2 sm:p-3 bg-slate-50 rounded-xl text-indigo-600">
                        <s.icon />
                      </div>
                      <span className={`text-[10px] sm:text-xs font-bold px-2 py-1 rounded-full ${s.trend.startsWith('+') ? 'bg-emerald-50 text-emerald-600' : 'bg-slate-50 text-slate-600'}`}>
                        {s.trend}
                      </span>
                    </div>
                    <p className="text-slate-500 text-xs sm:text-sm font-medium">{s.label}</p>
                    <h3 className="text-xl sm:text-2xl font-bold text-slate-900 mt-1">{s.value}</h3>
                  </div>
                ))}
              </div>

              {/* Chart Section */}
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 sm:gap-8">
                <div className="lg:col-span-2 bg-white p-4 sm:p-8 rounded-2xl border border-slate-200 shadow-sm">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 sm:mb-8 gap-4">
                    <div>
                      <h3 className="text-base sm:text-lg font-bold text-slate-800">Transaction Throughput</h3>
                      <p className="text-xs sm:text-sm text-slate-500">Broadcasting performance across last 24 hours</p>
                    </div>
                    <select className="bg-slate-50 border border-slate-200 rounded-lg px-3 py-1.5 text-xs sm:text-sm outline-none w-full sm:w-auto">
                      <option>Last 24 Hours</option>
                      <option>Last 7 Days</option>
                    </select>
                  </div>
                  <div className="h-60 sm:h-72">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={CHART_DATA}>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                        <XAxis dataKey="time" axisLine={false} tickLine={false} tick={{fill: '#94a3b8', fontSize: 10}} />
                        <YAxis axisLine={false} tickLine={false} tick={{fill: '#94a3b8', fontSize: 10}} />
                        <Tooltip 
                          contentStyle={{borderRadius: '12px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)'}}
                        />
                        <Line 
                          type="monotone" 
                          dataKey="tx" 
                          stroke="#4f46e5" 
                          strokeWidth={3} 
                          dot={{r: 4, fill: '#4f46e5', strokeWidth: 2, stroke: '#fff'}}
                          activeDot={{r: 6}}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </div>

                <div className="bg-white p-4 sm:p-8 rounded-2xl border border-slate-200 shadow-sm">
                  <h3 className="text-base sm:text-lg font-bold text-slate-800 mb-6">Tier Distribution</h3>
                  <div className="space-y-4 sm:space-y-6">
                    {Object.entries(TIER_CONFIG).map(([key, config]) => {
                      const count = clients.filter(c => c.tier === key).length;
                      const percentage = (count / clients.length) * 100;
                      return (
                        <div key={key}>
                          <div className="flex justify-between text-xs sm:text-sm mb-2">
                            <span className="font-semibold text-slate-700 capitalize">{key}</span>
                            <span className="text-slate-500">{count} clients</span>
                          </div>
                          <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
                            <div 
                              className={`h-full ${config.color.split(' ')[0]}`} 
                              style={{ width: `${percentage}%` }}
                            ></div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                  <div className="mt-8 pt-6 border-t border-slate-100">
                    <h4 className="text-xs sm:text-sm font-bold text-slate-800 mb-3 uppercase tracking-wider">Security Alerts</h4>
                    <div className="space-y-3">
                      <div className="flex items-center space-x-3 text-[10px] sm:text-xs bg-amber-50 text-amber-700 p-3 rounded-lg border border-amber-100">
                        <Icons.Security />
                        <span className="flex-1">Client cl_23a10b approaching rate limit</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'clients' && (
            <div className="space-y-4 sm:space-y-6">
              <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
                <div className="relative flex-1">
                  <input
                    type="text"
                    placeholder="Search clients..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 text-sm"
                  />
                  <div className="absolute left-3 top-3 text-slate-400">
                    <Icons.Dashboard />
                  </div>
                </div>
                <button 
                  onClick={() => setShowAddModal(true)}
                  className="bg-indigo-600 hover:bg-indigo-500 text-white font-semibold px-6 py-2.5 rounded-xl flex items-center justify-center space-x-2 transition-all text-sm"
                >
                  <Icons.Clients />
                  <span>Register Client</span>
                </button>
              </div>

              <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse min-w-[700px]">
                    <thead>
                      <tr className="bg-slate-50 border-b border-slate-200">
                        <th className="px-6 py-4 text-[10px] sm:text-xs font-bold text-slate-500 uppercase">Client Details</th>
                        <th className="px-6 py-4 text-[10px] sm:text-xs font-bold text-slate-500 uppercase">Security Tier</th>
                        <th className="px-6 py-4 text-[10px] sm:text-xs font-bold text-slate-500 uppercase">Daily Usage</th>
                        <th className="px-6 py-4 text-[10px] sm:text-xs font-bold text-slate-500 uppercase">Security Status</th>
                        <th className="px-6 py-4 text-[10px] sm:text-xs font-bold text-slate-500 uppercase text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {filteredClients.map(client => (
                        <tr key={client.id} className="hover:bg-slate-50 transition-colors">
                          <td className="px-6 py-4">
                            <p className="font-bold text-slate-900 text-sm">{client.name}</p>
                            <p className="text-[10px] text-slate-500 font-mono mt-0.5">{client.id}</p>
                          </td>
                          <td className="px-6 py-4">
                            <span className={`px-2.5 py-1 rounded-full text-[10px] font-bold capitalize ${TIER_CONFIG[client.tier].color}`}>
                              {client.tier}
                            </span>
                          </td>
                          <td className="px-6 py-4">
                            <div className="w-32">
                              <div className="flex justify-between text-[10px] mb-1">
                                <span className="font-bold">{client.current_day_tx.toLocaleString()}</span>
                                <span className="text-slate-400">/{client.max_daily_tx.toLocaleString()}</span>
                              </div>
                              <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
                                <div 
                                  className="h-full bg-indigo-500 rounded-full"
                                  style={{ width: `${(client.current_day_tx / client.max_daily_tx) * 100}%` }}
                                ></div>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex flex-col space-y-1">
                              <div className="flex items-center space-x-2">
                                <div className={`w-2 h-2 rounded-full ${client.require_signature ? 'bg-emerald-500' : 'bg-amber-400'}`}></div>
                                <span className="text-xs text-slate-600 font-medium whitespace-nowrap">{client.require_signature ? 'ECDSA Enforced' : 'Key-Only'}</span>
                              </div>
                              {client.allowed_ips.length > 0 && (
                                <span className="text-[10px] text-slate-400 truncate max-w-[120px]">IP Lock: {client.allowed_ips[0]}</span>
                              )}
                            </div>
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button className="text-indigo-600 hover:text-indigo-800 text-sm font-bold">Edit</button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'health' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8">
              <div className="bg-white p-4 sm:p-8 rounded-2xl border border-slate-200 shadow-sm">
                <h3 className="text-base sm:text-lg font-bold text-slate-800 mb-6 flex items-center">
                  <span className="p-2 bg-indigo-50 text-indigo-600 rounded-lg mr-3"><Icons.Health /></span>
                  Node Infrastructure Status
                </h3>
                <div className="space-y-4 sm:space-y-6">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between p-4 bg-slate-50 rounded-xl gap-2">
                    <div>
                      <p className="font-bold text-slate-800 text-sm sm:text-base">GovHash Primary API</p>
                      <p className="text-xs sm:text-sm text-slate-500">v1.2.4-stable</p>
                    </div>
                    <span className="w-fit px-3 py-1 bg-emerald-100 text-emerald-700 rounded-full text-[10px] font-bold uppercase tracking-wider">Online</span>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-4 border border-slate-100 rounded-xl">
                      <p className="text-[10px] text-slate-400 uppercase font-bold mb-1">UTXO Confirmed</p>
                      <p className="text-lg sm:text-xl font-bold text-slate-900">4,821</p>
                    </div>
                    <div className="p-4 border border-slate-100 rounded-xl">
                      <p className="text-[10px] text-slate-400 uppercase font-bold mb-1">UTXO Splitting</p>
                      <p className="text-lg sm:text-xl font-bold text-indigo-600">379</p>
                    </div>
                  </div>
                </div>
              </div>

              <div className="bg-white p-4 sm:p-8 rounded-2xl border border-slate-200 shadow-sm">
                <h3 className="text-base sm:text-lg font-bold text-slate-800 mb-6 flex items-center">
                  <span className="p-2 bg-slate-50 text-slate-600 rounded-lg mr-3"><Icons.Security /></span>
                  Security Audit Trail
                </h3>
                <div className="space-y-4">
                  {[
                    { event: 'Tier Upgrade', client: 'NotaryHash', time: '12m ago' },
                    { event: 'New Registration', client: 'AKUA Pilot', time: '4h ago' },
                    { event: 'Suspicious IP Block', client: 'Global Auth', time: '6h ago' },
                    { event: 'Admin Login', client: 'SystemRoot', time: '1d ago' },
                  ].map((log, i) => (
                    <div key={i} className="flex items-start justify-between py-3 border-b border-slate-50 last:border-0 gap-4">
                      <div className="flex-1">
                        <p className="text-xs sm:text-sm font-bold text-slate-800">{log.event}</p>
                        <p className="text-[10px] sm:text-xs text-slate-500">{log.client}</p>
                      </div>
                      <span className="text-[9px] sm:text-[10px] font-medium text-slate-400 uppercase whitespace-nowrap">{log.time}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </main>

      {/* Register Client Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/50 backdrop-blur-sm p-4">
          <div className="bg-white rounded-2xl sm:rounded-3xl w-full max-w-2xl max-h-[90vh] shadow-2xl overflow-y-auto animate-in fade-in zoom-in duration-200">
            <div className="p-6 sm:p-8 border-b border-slate-100 bg-slate-50 flex justify-between items-center sticky top-0 z-10">
              <div>
                <h3 className="text-lg sm:text-xl font-bold text-slate-900">Register New Client</h3>
                <p className="text-xs sm:text-sm text-slate-500">Configure security tier and broadcast limits</p>
              </div>
              <button 
                onClick={() => setShowAddModal(false)}
                className="text-slate-400 hover:text-slate-600 p-2 hover:bg-slate-200 rounded-lg transition-colors"
              >
                ✕
              </button>
            </div>
            <form onSubmit={handleRegisterClient} className="p-6 sm:p-8 space-y-4 sm:space-y-6">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6">
                <div className="sm:col-span-2">
                  <label className="block text-xs sm:text-sm font-bold text-slate-700 mb-1.5 sm:mb-2">Client Name</label>
                  <input 
                    name="name"
                    required
                    placeholder="e.g. Acme Blockchain Solutions"
                    className="w-full px-4 py-2.5 sm:py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 transition-all text-sm"
                  />
                </div>
                <div>
                  <label className="block text-xs sm:text-sm font-bold text-slate-700 mb-1.5 sm:mb-2">Security Tier</label>
                  <select 
                    name="tier"
                    required
                    className="w-full px-4 py-2.5 sm:py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 transition-all text-sm"
                  >
                    <option value={ClientTier.PILOT}>Pilot (Testing)</option>
                    <option value={ClientTier.ENTERPRISE}>Enterprise (Commercial)</option>
                    <option value={ClientTier.GOVERNMENT}>Government (Institutional)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs sm:text-sm font-bold text-slate-700 mb-1.5 sm:mb-2">Daily Tx Limit</label>
                  <input 
                    name="max_daily_tx"
                    type="number"
                    defaultValue={10000}
                    className="w-full px-4 py-2.5 sm:py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 transition-all text-sm"
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs sm:text-sm font-bold text-slate-700 mb-1.5 sm:mb-2">Allowed IPs (Comma separated)</label>
                  <input 
                    name="allowed_ips"
                    placeholder="127.0.0.1, 10.0.0.1"
                    className="w-full px-4 py-2.5 sm:py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 transition-all text-sm"
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs sm:text-sm font-bold text-slate-700 mb-1.5 sm:mb-2">ECDSA Public Key (Optional for Pilot)</label>
                  <textarea 
                    name="public_key"
                    rows={2}
                    placeholder="04a6..."
                    className="w-full px-4 py-2.5 sm:py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500 transition-all font-mono text-[10px] sm:text-xs"
                  ></textarea>
                </div>
              </div>
              <div className="pt-4 flex flex-col sm:flex-row justify-end gap-3 sm:gap-4">
                <button 
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="w-full sm:w-auto px-6 py-2.5 sm:py-3 text-slate-600 font-bold hover:bg-slate-50 rounded-xl transition-all text-sm"
                >
                  Cancel
                </button>
                <button 
                  type="submit"
                  className="w-full sm:w-auto bg-indigo-600 hover:bg-indigo-500 text-white font-bold px-8 py-2.5 sm:py-3 rounded-xl shadow-lg shadow-indigo-500/20 transition-all text-sm"
                >
                  Generate Credentials
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default App;
