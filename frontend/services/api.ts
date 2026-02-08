// API Service Layer for GovHash Admin Portal

const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://api.govhash.org';

// Store admin password in sessionStorage after login
let adminPassword = '';

export const setAdminPassword = (password: string) => {
  adminPassword = password;
  sessionStorage.setItem('admin_auth', password);
};

export const getAdminPassword = (): string => {
  if (adminPassword) return adminPassword;
  return sessionStorage.getItem('admin_auth') || '';
};

export const clearAdminPassword = () => {
  adminPassword = '';
  sessionStorage.removeItem('admin_auth');
};

// Helper to make authenticated requests
const fetchWithAuth = async (url: string, options: RequestInit = {}) => {
  const headers = {
    'Content-Type': 'application/json',
    'X-Admin-Password': getAdminPassword(),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE_URL}${url}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
};

// Client Management APIs
export const clientAPI = {
  list: async () => {
    const data = await fetchWithAuth('/admin/clients/list');
    return data.clients || [];
  },

  register: async (clientData: {
    name: string;
    tier: string;
    max_daily_tx: number;
    public_key?: string;
    allowed_ips?: string[];
    site_origin?: string;
  }) => {
    return await fetchWithAuth('/admin/clients/register', {
      method: 'POST',
      body: JSON.stringify(clientData),
    });
  },

  updateSecurity: async (
    clientId: string,
    security: {
      tier?: string;
      require_signature?: boolean;
      allowed_ips?: string[];
      grace_period_hours?: number;
    }
  ) => {
    return await fetchWithAuth(`/admin/clients/${clientId}/security`, {
      method: 'PATCH',
      body: JSON.stringify(security),
    });
  },

  activate: async (clientId: string) => {
    return await fetchWithAuth(`/admin/clients/${clientId}/activate`, {
      method: 'POST',
    });
  },

  deactivate: async (clientId: string) => {
    return await fetchWithAuth(`/admin/clients/${clientId}/deactivate`, {
      method: 'POST',
    });
  },
};

// System Health APIs
export const healthAPI = {
  check: async () => {
    const response = await fetch(`${API_BASE_URL}/health`);
    return response.json();
  },

  stats: async () => {
    const response = await fetch(`${API_BASE_URL}/admin/stats`);
    return response.json();
  },

  trainStatus: async () => {
    return await fetchWithAuth('/admin/emergency/status');
  },
};

// Emergency Controls
export const emergencyAPI = {
  stopTrain: async () => {
    return await fetchWithAuth('/admin/emergency/stop-train', {
      method: 'POST',
    });
  },

  status: async () => {
    return await fetchWithAuth('/admin/emergency/status');
  },
};

// Auth helper
export const authAPI = {
  verify: async (password: string): Promise<boolean> => {
    try {
      setAdminPassword(password);
      await clientAPI.list();
      return true;
    } catch (error) {
      clearAdminPassword();
      return false;
    }
  },

  logout: () => {
    clearAdminPassword();
  },
};
