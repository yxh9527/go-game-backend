package dao

// Terminal sold keys can expire only after durable market ownership was
// confirmed. Active tickets and legacy unconfirmed ownership stay protected.
const scratchRetirementScript = `
local function scratchTerminal(state)
 local raw=redis.call('HGET',KEYS[3],ARGV[1])
 if not raw then error('SCRATCH_ROUND_MISSING') end
 local round=cjson.decode(raw);local ticket=round.scratch
 if round.state~=state or type(ticket)~='table' then error('SCRATCH_NOT_TERMINAL') end
 if tonumber(round.agent)~=tonumber(ARGV[4]) or tonumber(round.gameId)~=tonumber(ARGV[5]) or tonumber(ticket.userId)~=tonumber(ARGV[6]) or ticket.currency~=ARGV[7] or ticket.serialFull~=ARGV[8] or ticket.roundId~=ARGV[1] then error('SCRATCH_OWNER_MISMATCH') end
 if ticket.mirrored~=true then error('SCRATCH_NOT_MIRRORED') end
 if ticket.soldMirrored~=true then error('SCRATCH_SOLD_NOT_MIRRORED') end
 local completed=type(round.settlement)=='table' and tonumber(round.settlement.completedAt) or nil
 if not completed or completed<=0 then completed=tonumber(round.updatedAt) end
 local now=tonumber(ARGV[9])
 if not completed or completed<=0 or completed>now then error('SCRATCH_COMPLETION_TIME_INVALID') end
 local sold=redis.call('GET',KEYS[2])
 if sold and sold~=ARGV[1] then error('SCRATCH_SERIAL_OWNER_CHANGED') end
 return round,ticket,completed+3600-now
end
local function retireScratchSold(remaining)
 if remaining<=0 then redis.call('DEL',KEYS[2]);return end
 local ttl=redis.call('PTTL',KEYS[2]);local bound=remaining*1000
 if ttl==-1 or ttl>bound then redis.call('PEXPIRE',KEYS[2],bound) end
end
`

const dropScratchPendingScript = scratchRetirementScript + `
local round,ticket,remaining=scratchTerminal('VOIDED')
if ticket.refunded~=true then return redis.error_reply('SCRATCH_NOT_REFUNDED') end
redis.call('SREM',KEYS[1],ARGV[1]);retireScratchSold(remaining)
return 1
`
