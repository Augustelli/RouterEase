import { describe, expect, it } from 'vitest'
import { ruleSchema } from './Pages'
describe('DNS rule schema',()=>{it('accepts a normalized DNS domain',()=>expect(ruleSchema.parse({domain:'Example.ORG',action:'BLOCK',enabled:true}).domain).toBe('example.org'));it('rejects malformed domains',()=>expect(ruleSchema.safeParse({domain:'badexample',action:'BLOCK',enabled:true}).success).toBe(false))})
