
export enum ClientTier {
  PILOT = 'pilot',
  ENTERPRISE = 'enterprise',
  GOVERNMENT = 'government'
}

export interface Client {
  id: string;
  name: string;
  api_key: string;
  tier: ClientTier;
  max_daily_tx: number;
  require_signature: boolean;
  allowed_ips: string[];
  public_key?: string;
  created_at: string;
  status: 'active' | 'suspended';
  current_day_tx: number;
  grace_period_hours?: number;
}

export interface SystemStats {
  total_tx_24h: number;
  active_clients: number;
  utxo_count: number;
  health_status: 'operational' | 'degraded' | 'critical';
  avg_broadcast_latency: number;
}

export interface HealthCheckResponse {
  status: string;
  utxoStats: {
    total: number;
    confirmed: number;
    unconfirmed: number;
    splitting: number;
  };
  nodeInfo: {
    version: string;
    protocol: string;
  };
}
