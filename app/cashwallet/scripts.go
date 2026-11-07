package cashwallet

// Apply preflights every key and every operation before its first write. Lua
// errors do not roll back Redis writes, so mutations must remain in the tail.
const Apply = `
local req = cjson.decode(ARGV[1])
local function typ(key, expected)
  local t = redis.call('TYPE', key).ok
  if t ~= 'none' and t ~= expected then error('CASH_KEY_TYPE') end
end
typ(KEYS[1], 'set')
local accounts = {}
local MAX = 9007199254740991
local function integer(n) return type(n) == 'number' and n % 1 == 0 and math.abs(n) <= MAX end
for i, a in ipairs(req.accounts) do
  local pk, ck = KEYS[2*i], KEYS[2*i+1]
  typ(pk, 'hash'); typ(ck, 'string')
  if redis.call('HGET',pk,'id') ~= tostring(a.userId) then return redis.error_reply('CASH_PLAYER_MISSING') end
  if redis.call('HGET',pk,'agent_id') ~= tostring(a.agentId) or redis.call('HGET',pk,'currency_type') ~= a.currency then
    return redis.error_reply('CASH_ACCOUNT_MISMATCH')
  end
  local balance = tonumber(redis.call('HGET',pk,'currency'))
  if not integer(balance) or balance < 0 then return redis.error_reply('CASH_BALANCE_INVALID') end
  local revision=tonumber(redis.call('HGET',pk,'cash_revision') or '0')
  if not integer(revision) or revision < 0 or revision >= MAX or revision ~= a.revision then return redis.error_reply('CASH_REVISION') end
  local raw = redis.call('GET',ck)
  local carry = raw and cjson.decode(raw) or nil
  if carry and (not integer(carry.amount) or carry.amount < 0 or carry.amount > balance) then
    return redis.error_reply('CASH_CARRY_INVALID')
  end
  accounts[i] = {balance=balance, carry=carry, account=a, pk=pk, ck=ck, revision=revision, dirty=false, touched=false}
end
for _, op in ipairs(req.steps) do
  local a = accounts[op.account]
    local c = a.carry
    local n = op.amount or 0
    if not integer(n) then return redis.error_reply('CASH_AMOUNT_INVALID') end
    if op.kind == 'reserve' then
      if n < 0 or (c and (c.token ~= op.token or c.tableKey ~= op.tableKey)) then return redis.error_reply('CASH_CARRY_CONFLICT') end
      if not c then c={amount=0, token=op.token, tableKey=op.tableKey, revision=0}; a.carry=c end
      if a.balance - c.amount < n then return redis.error_reply('CASH_INSUFFICIENT') end
      c.amount=c.amount+n; c.revision=c.revision+1; a.touched=true
    elseif op.kind == 'carryDelta' or op.kind == 'renew' then
      if not c or c.token ~= op.token or c.tableKey ~= op.tableKey then return redis.error_reply('CASH_CARRY_EXPIRED') end
      if c.revision ~= op.revision then return redis.error_reply('CASH_REVISION') end
      if op.kind == 'carryDelta' then
        if c.amount+n < 0 or a.balance+n < 0 or not integer(c.amount+n) or not integer(a.balance+n) then return redis.error_reply('CASH_INSUFFICIENT') end
        c.amount=c.amount+n; a.balance=a.balance+n; c.revision=c.revision+1; a.dirty=true
      end
      a.touched=true
    elseif op.kind == 'release' then
      if c and c.token == op.token and c.tableKey == op.tableKey then
        if c.revision ~= op.revision then return redis.error_reply('CASH_REVISION') end
        if n < 0 or n > c.amount then return redis.error_reply('CASH_AMOUNT_INVALID') end
        c.amount=c.amount-n; c.revision=c.revision+1; a.touched=true
        if c.amount == 0 and not op.keepOwner then a.carry=nil end
      end
    elseif op.kind == 'debit' or op.kind == 'credit' then
      if n < 0 then return redis.error_reply('CASH_AMOUNT_INVALID') end
      local delta = op.kind == 'debit' and -n or n
      if a.balance+delta < (c and c.amount or 0) or not integer(a.balance+delta) then return redis.error_reply('CASH_INSUFFICIENT') end
      a.balance=a.balance+delta; a.dirty=true
    else return redis.error_reply('CASH_OPERATION_INVALID') end
end
local result = {id=req.id, accounts={}}
for i,a in ipairs(accounts) do
  result.accounts[i]={userId=a.account.userId,agentId=a.account.agentId,currency=a.account.currency,revision=a.revision+1,balance=a.balance,carry=a.carry or cjson.null,ttlMs=a.carry and (a.touched and 300000 or redis.call('PTTL',a.ck)) or 0}
end
local encoded = cjson.encode(result)
-- No validation or fallible arithmetic below this line.
for _,a in ipairs(accounts) do
  if a.dirty then
    redis.call('HSET',a.pk,'currency',string.format('%.0f',a.balance))
    redis.call('SADD',KEYS[1],tostring(a.account.userId))
  end
  redis.call('HSET',a.pk,'cash_revision',string.format('%.0f',a.revision+1))
  redis.call('EXPIRE',a.pk,1200)
  if a.touched then
    if a.carry then redis.call('SET',a.ck,cjson.encode(a.carry),'PX',300000) else redis.call('DEL',a.ck) end
  end
end
return encoded
`

// Update guards legacy withdrawals in their existing integer currency units.
// String arithmetic preserves int64 precision for pre-existing large accounts.
const LegacyUpdate = `
if redis.call('HEXISTS',KEYS[1],'id') == 0 then return false end
local current=redis.call('HGET',KEYS[1],'currency'); if not current then return false end
local delta=ARGV[1]
local function less(a,b) if #a ~= #b then return #a < #b end return a < b end
local function add(a,b)
  local out,carry='',0;local i,j=#a,#b
  while i>0 or j>0 or carry>0 do
    local n=(i>0 and tonumber(a:sub(i,i)) or 0)+(j>0 and tonumber(b:sub(j,j)) or 0)+carry
    out=tostring(n%10)..out;carry=math.floor(n/10);i=i-1;j=j-1
  end
  return out
end
local carryKey='gogame:cash-carry:'..(redis.call('HGET',KEYS[1],'agent_id') or '0')..':'..(redis.call('HGET',KEYS[1],'id') or '')..':'..(redis.call('HGET',KEYS[1],'currency_type') or 'CNY')
local raw=redis.call('GET',carryKey);local reserved=raw and cjson.decode(raw).amount or 0
if type(reserved)~='number' or reserved<0 or reserved%1~=0 or reserved>9007199254740991 then return redis.error_reply('CASH_CARRY_INVALID') end
local revision=tonumber(redis.call('HGET',KEYS[1],'cash_revision') or '0')
if not revision or revision<0 or revision%1~=0 or revision>=9007199254740991 then return redis.error_reply('CASH_REVISION') end
if delta:sub(1,1)=='-' then
  if current:sub(1,1)=='-' or less(current,add(delta:sub(2),string.format('%.0f',reserved))) then return -1 end
elseif current:sub(1,1)=='-' and less(delta,current:sub(2)) then return -1 end
redis.call('HINCRBY',KEYS[1],'currency',delta)
redis.call('HSET',KEYS[1],'cash_revision',string.format('%.0f',revision+1))
redis.pcall('HINCRBY',KEYS[1],'exp',ARGV[2])
redis.call('EXPIRE',KEYS[1],1200)
redis.pcall('SADD','dirty_list_imp',redis.call('HGET',KEYS[1],'id'))
return redis.call('HGET',KEYS[1],'currency')
`
