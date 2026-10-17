CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE routers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  router_id uuid NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
  mac_address text NOT NULL,
  last_known_ip inet,
  last_seen timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(router_id, mac_address)
);
CREATE INDEX devices_router_ip_idx ON devices(router_id, last_known_ip);

CREATE TABLE device_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  router_id uuid NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
  description text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(router_id, name)
);
CREATE TABLE device_group_members (
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  group_id uuid NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
  PRIMARY KEY(device_id, group_id)
);

CREATE TABLE dns_policies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 128),
  description text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT true,
  default_action text NOT NULL CHECK (default_action IN ('ALLOW', 'BLOCK')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE dns_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  policy_id uuid NOT NULL REFERENCES dns_policies(id) ON DELETE CASCADE,
  domain text NOT NULL,
  action text NOT NULL CHECK (action IN ('ALLOW', 'BLOCK')),
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(policy_id, domain)
);
CREATE INDEX dns_rules_policy_idx ON dns_rules(policy_id);

CREATE TABLE policy_assignments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  policy_id uuid NOT NULL REFERENCES dns_policies(id) ON DELETE CASCADE,
  device_id uuid REFERENCES devices(id) ON DELETE CASCADE,
  group_id uuid REFERENCES device_groups(id) ON DELETE CASCADE,
  priority integer NOT NULL DEFAULT 100 CHECK (priority >= 0 AND priority <= 100000),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((device_id IS NOT NULL AND group_id IS NULL) OR (device_id IS NULL AND group_id IS NOT NULL)),
  UNIQUE(policy_id, device_id),
  UNIQUE(policy_id, group_id)
);
CREATE INDEX policy_assignments_device_idx ON policy_assignments(device_id, priority);
CREATE INDEX policy_assignments_group_idx ON policy_assignments(group_id, priority);

CREATE TABLE router_api_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  router_id uuid NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
  key_hash text NOT NULL,
  scopes text[] NOT NULL DEFAULT ARRAY['dns:query', 'inventory:write'],
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  revoked_at timestamptz,
  last_used_at timestamptz
);
CREATE INDEX router_api_keys_router_idx ON router_api_keys(router_id);

CREATE TABLE device_leases (
  router_id uuid NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  ip inet NOT NULL,
  observed_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  PRIMARY KEY(router_id, ip),
  UNIQUE(router_id, device_id)
);
CREATE INDEX device_leases_lookup_idx ON device_leases(router_id, ip, expires_at);

CREATE TABLE policy_decision_audits (
  id bigserial PRIMARY KEY,
  router_id uuid NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
  device_id uuid REFERENCES devices(id) ON DELETE SET NULL,
  decision text NOT NULL CHECK (decision IN ('ALLOW', 'BLOCK', 'UNAVAILABLE')),
  reason text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
