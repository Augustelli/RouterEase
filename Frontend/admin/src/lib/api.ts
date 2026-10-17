export type Policy={id:string;name:string;description:string;enabled:boolean;default_action:'ALLOW'|'BLOCK';created_at:string;updated_at:string}
export type Rule={id:string;policy_id:string;domain:string;action:'ALLOW'|'BLOCK';enabled:boolean;created_at:string}
export type Router={id:string;name:string;status:string}; export type Device={id:string;name:string;router_id:string}; export type Group={id:string;name:string;description:string;router_id:string}
const base=import.meta.env.VITE_API_BASE_URL ?? '/routerease/dns'
export async function api<T>(path:string, token:string, init:RequestInit={}) :Promise<T>{const r=await fetch(`${base}${path}`,{...init,headers:{Authorization:`Bearer ${token}`,...(init.headers||{})}});if(!r.ok)throw new Error((await r.text())||`Request failed (${r.status})`);return r.status===204?undefined as T: r.json()}
export const policies=(t:string)=>api<Policy[]>('/api/v1/policies',t)
export const rules=(t:string,id:string)=>api<Rule[]>(`/api/v1/policies/${id}/rules`,t)
