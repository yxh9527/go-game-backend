import config from "@/config";

/** 把后台注单行 / fgServer 注单列表项统一成详情弹窗可用结构。 */

export const safeNumber = (value) => {
  if (value === null || value === undefined || value === "") return 0;
  return Number(value) || 0;
};

export const formatAmount = (value) => {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "string" && Number.isNaN(Number(value))) return value;
  return safeNumber(value).toFixed(2);
};

export const formatPlayedTime = (value) => {
  if (!value && value !== 0) return "";
  if (typeof value === "string" && value.includes("-")) return value;
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) return String(value || "");
  const ms = numeric < 1e12 ? numeric * 1000 : numeric;
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) return String(value);
  const pad = (n) => String(n).padStart(2, "0");
  return (
    date.getFullYear() +
    "-" +
    pad(date.getMonth() + 1) +
    "-" +
    pad(date.getDate()) +
    " " +
    pad(date.getHours()) +
    ":" +
    pad(date.getMinutes()) +
    ":" +
    pad(date.getSeconds())
  );
};

export const parseMaybeJson = (value) => {
  let current = value;
  for (let i = 0; i < 3; i += 1) {
    if (typeof current !== "string") return current;
    const text = current.trim();
    if (!text) return current;
    if (text.charAt(0) !== "{" && text.charAt(0) !== "[") return current;
    try {
      current = JSON.parse(text);
    } catch (error) {
      return value;
    }
  }
  return current;
};

const normalizeOssBaseUrl = (url) => {
  if (!url) return "";
  return url.endsWith("/") ? url : url + "/";
};

/** 与 fgServer recordService.buildRecordAssetUrl 对齐：只给 /global 相对路径补 OSS。 */
export const resolveAssetUrl = (value, ossUrl) => {
  if (value === null || value === undefined || value === "") return "";
  if (typeof value !== "string") return value;
  const text = value.trim();
  if (!text) return "";
  if (/^(?:https?:)?\/\//i.test(text) || text.indexOf("data:") === 0) {
    return text;
  }

  let assetPath = "";
  if (text.indexOf("/global/") === 0) assetPath = text.slice(1);
  else if (text.indexOf("global/") === 0) assetPath = text;
  if (!assetPath) return text;

  const base = normalizeOssBaseUrl(ossUrl || config.ossUrl || "");
  return base ? base + assetPath : "/" + assetPath;
};

/** 详情里图片路径可能藏在任意嵌套结构，递归只改资源字符串。 */
export const applyRecordAssetBaseUrl = (value, ossUrl) => {
  if (typeof value === "string") return resolveAssetUrl(value, ossUrl);
  if (Array.isArray(value)) {
    return value.map((item) => applyRecordAssetBaseUrl(item, ossUrl));
  }
  if (value && typeof value === "object") {
    const next = {};
    Object.keys(value).forEach((key) => {
      next[key] = applyRecordAssetBaseUrl(value[key], ossUrl);
    });
    return next;
  }
  return value;
};

export const buildSlotCardImageUrl = (gameId, symbolId, size) => {
  const id = Math.max(Number(symbolId) || 0, 0);
  const padded = String(id).padStart(2, "0");
  return (
    "/global/game/game_FG/" +
    Number(gameId) +
    "/" +
    (size || "40x40") +
    "/card_" +
    padded +
    ".png?v=1.23"
  );
};

const looksLikeDetailInfo = (value) =>
  Boolean(
    value &&
      typeof value === "object" &&
      (value.grids ||
        value.specific_bet_info ||
        value.result_show_url ||
        value.result_grid_url ||
        value.type_id != null ||
        value.game_id != null ||
        value.total_bet),
  );

/**
 * 对齐 fgServer clientApiRpcProd.mapEsToDetailRow：
 * 形态 A：log 直接是 { styleV, info }
 * 形态 B：log 是完整 SlotSpinRecord，详情在 detail.info
 */
const extractRecordDetail = (value) => {
  const parsed = parseMaybeJson(value);
  if (!parsed || typeof parsed !== "object") return null;
  if (parsed.info && typeof parsed.info === "object") {
    return parsed;
  }
  const nested = parseMaybeJson(parsed.detail);
  if (nested && typeof nested === "object" && nested.info && typeof nested.info === "object") {
    return nested;
  }
  if (looksLikeDetailInfo(parsed)) {
    return { info: parsed };
  }
  return null;
};

export const resolveDetailPayload = (row) => {
  if (!row || typeof row !== "object") {
    return { detail: null, info: null };
  }
  const sources = [row.detail, row.log, row.info ? { info: row.info } : null];
  for (let i = 0; i < sources.length; i += 1) {
    if (sources[i] == null || sources[i] === "") continue;
    const detail = extractRecordDetail(sources[i]);
    if (detail && detail.info && typeof detail.info === "object") {
      return { detail: detail, info: detail.info };
    }
  }
  return { detail: null, info: null };
};

const fruitKaijiangUrl = (gameId, winPosition) =>
  "/global/game/fruit/zh_CN/" + gameId + "/kaijiang/" + winPosition + ".png?v=1.23";

const fruitXiazhuUrl = (gameId, position) =>
  "/global/game/fruit/zh_CN/" + gameId + "/xiazhu/" + position + ".png?v=1.23";

/** 对齐 recordService.buildSlotLogDetailsResponse 对旧 Fruit 记录的补齐。 */
const patchFruitRecordInfo = (rawInfo, recordId) => {
  if (!rawInfo || typeof rawInfo !== "object") return rawInfo;
  let info = Object.assign({}, rawInfo);
  const gameId = Number(info.game_id || 0);

  if (gameId === 7026 && Array.isArray(info.specific_bet_info)) {
    const betPositions = Object.keys(info.total_bet || {});
    info.specific_bet_info = info.specific_bet_info.map((betInfo, index) =>
      Object.assign({}, betInfo, {
        bet_area_url:
          betInfo.bet_area_url ||
          (betPositions[index] ? fruitXiazhuUrl(7026, betPositions[index]) : ""),
      }),
    );
  }

  if ([7000, 7002, 7008, 7009, 7026].indexOf(gameId) >= 0 && !info.ju_id) {
    const winPosition = Number((info.reward_result && info.reward_result[0]) || 0);
    const betNum = info.all_bets != null ? info.all_bets : "0";
    const rewardMoney = info.sum_bonus != null ? info.sum_bonus : 0;
    const betPositions = Object.keys(info.total_bet || {});
    info = Object.assign({}, info, {
      ju_id: info.ju_id || recordId,
      reward_result: ["普通"],
      result_show_url: winPosition > 0 ? [fruitKaijiangUrl(gameId, winPosition)] : [],
      specific_bet_info: (info.specific_bet_info || []).map((betInfo, index) =>
        Object.assign({}, betInfo, {
          bet_area_url:
            betInfo.bet_area_url ||
            fruitXiazhuUrl(gameId, betPositions[index] != null ? betPositions[index] : index + 1),
          bet_nums: betInfo.bet_nums != null ? betInfo.bet_nums : 0,
          reward_rate: Number(betInfo.reward_money) > 0 ? betInfo.reward_rate : 0,
          num: betInfo.num != null ? betInfo.num : 0,
          index: betInfo.index != null ? betInfo.index : "",
          grid: betInfo.grid || [],
        }),
      ),
      total_bet: { bet_num: betNum, reward_money: rewardMoney },
      title_type: info.title_type != null ? info.title_type : 1,
      level: info.level != null ? info.level : 0,
      player_name: info.player_name || "",
    });
  }

  if (gameId === 7011 && !(info.specific_bet_info && info.specific_bet_info.length)) {
    const elements = (info.reward_result || [])
      .map(Number)
      .filter(Number.isFinite)
      .slice(0, 16);
    const grid = [];
    for (let row = 0; row < 4; row += 1) {
      grid.push(
        elements.slice(row * 4, (row + 1) * 4).map((element) =>
          fruitXiazhuUrl(7011, element === 6 ? 16 : element),
        ),
      );
    }
    const legacyTotal = info.total_bet || {};
    const betNum = info.all_bets != null ? info.all_bets : "0";
    const rewardMoney = info.sum_bonus != null ? info.sum_bonus : 0;
    const betValue = Number(betNum);
    const rewardRate = betValue > 0 ? (Number(rewardMoney) / betValue) * 100 : 0;
    info = Object.assign({}, info, {
      ju_id: recordId,
      reward_result: ["普通"],
      specific_bet_info: [
        {
          bet_area_url: 0,
          bet_num: betNum,
          bet_nums: Number(legacyTotal.bet_num || 0),
          reward_rate: rewardRate,
          reward_money: rewardMoney,
          num: 0,
          index: 1,
          grid: grid,
        },
      ],
      total_bet: { bet_num: betNum, reward_money: rewardMoney },
      title_type: 1,
      level: 1,
      player_name: "",
    });
  }

  if (gameId === 7027) {
    const currentBet = (info.specific_bet_info && info.specific_bet_info[0]) || {};
    const legacyTotal = info.total_bet || {};
    const level =
      info.num != null
        ? info.num
        : currentBet.num != null
          ? currentBet.num
          : Number(legacyTotal.level || 0);
    const rewardRate = Number(currentBet.reward_rate || 0);
    const betNum =
      info.bet_num != null
        ? info.bet_num
        : currentBet.bet_num != null
          ? currentBet.bet_num
          : info.all_bets != null
            ? info.all_bets
            : "0";
    const rewardMoney =
      info.reward_money != null
        ? info.reward_money
        : currentBet.reward_money != null
          ? currentBet.reward_money
          : info.sum_bonus != null
            ? info.sum_bonus
            : 0;
    const riskLevels = ["", "低", "中", "高"];
    const risk = info.risk || riskLevels[Number(legacyTotal.risk || 0)] || "";
    info = Object.assign({}, info, {
      ju_id: info.ju_id || recordId,
      reward_result: ["普通"],
      specific_bet_info: [
        Object.assign({}, currentBet, {
          bet_area_url: currentBet.bet_area_url || 0,
          bet_num: betNum,
          bet_nums: currentBet.bet_nums != null ? currentBet.bet_nums : 0,
          reward_rate: rewardRate,
          reward_money: rewardMoney,
          num: level,
          index: currentBet.index != null ? currentBet.index : "",
          grid: currentBet.grid || [],
        }),
      ],
      total_bet: { bet_num: betNum, reward_money: rewardMoney },
      title_type: info.title_type != null ? info.title_type : 1,
      level: info.level != null ? info.level : 0,
      bet_num: betNum,
      risk: risk,
      num: level,
      rate: info.rate != null ? info.rate : (rewardRate / 100).toFixed(2),
      reward_money: rewardMoney,
      player_name: info.player_name || "",
    });
  }

  return info;
};

const normalizeBonusGameInfo = (info) => {
  if (!info || typeof info !== "object") return info;
  const raw = info.bonus_game_info;
  const hasObjectShape =
    raw && typeof raw === "object" && !Array.isArray(raw) && Object.prototype.hasOwnProperty.call(raw, "bonus_info");
  if (hasObjectShape) return info;
  return Object.assign({}, info, {
    bonus_game_info: {
      bonus_info: Array.isArray(raw) ? raw : [],
      xiao_all_bonus: info.xiao_all_bonus != null ? info.xiao_all_bonus : info.sum_bonus != null ? info.sum_bonus : 0,
    },
  });
};

const convertNumericCells = (value, gameId, size, ossUrl) => {
  if (typeof value === "number" || /^\d+$/.test(String(value || ""))) {
    return resolveAssetUrl(buildSlotCardImageUrl(gameId, value, size), ossUrl);
  }
  if (Array.isArray(value)) {
    return value.map((item) => convertNumericCells(item, gameId, size, ossUrl));
  }
  return value;
};

const enrichDetailInfo = (rawInfo, gameId, ossUrl, recordId) => {
  if (!rawInfo || typeof rawInfo !== "object") return null;
  const id = Number((rawInfo.game_id != null ? rawInfo.game_id : gameId) || 0);
  let info = patchFruitRecordInfo(Object.assign({}, rawInfo, { game_id: id || rawInfo.game_id }), recordId);
  info = normalizeBonusGameInfo(info);

  if (Array.isArray(info.grids)) {
    info.grids = convertNumericCells(info.grids, id, "40x40", ossUrl);
  }
  if (Array.isArray(info.lines_info)) {
    info.lines_info = info.lines_info.map((line) => {
      if (!line || typeof line !== "object") return line;
      const next = Object.assign({}, line);
      if (next.symbol) next.symbol = convertNumericCells(next.symbol, id, "20x20", ossUrl);
      if (Array.isArray(next.grid_data)) {
        next.grid_data = convertNumericCells(next.grid_data, id, "40x40", ossUrl);
      }
      return next;
    });
  }
  if (Array.isArray(info.m_list_info)) {
    info.m_list_info = info.m_list_info.map((line) => {
      if (!line || typeof line !== "object") return line;
      const next = Object.assign({}, line);
      if (next.symbol) next.symbol = convertNumericCells(next.symbol, id, "20x20", ossUrl);
      if (Array.isArray(next.grid_data)) {
        next.grid_data = convertNumericCells(next.grid_data, id, "40x40", ossUrl);
      }
      return next;
    });
  }
  if (Array.isArray(info.specific_bet_info)) {
    info.specific_bet_info = info.specific_bet_info.map((bet) => {
      if (!bet || typeof bet !== "object") return bet;
      const next = Object.assign({}, bet);
      if (Array.isArray(next.grid)) {
        next.grid = convertNumericCells(next.grid, id, "40x40", ossUrl);
      }
      return next;
    });
  }

  info.grids = info.grids || [];
  info.scatter_info = info.scatter_info || [];
  info.lines_info = info.lines_info || [];
  info.m_list_info = info.m_list_info || [];
  info.little_game_list = info.little_game_list || [];
  info.choose_list = info.choose_list || [];
  info.conin_info = info.conin_info || [];
  info.grid_show = info.grid_show || [];
  info.type_id = info.type_id != null ? info.type_id : 1;
  info.times = info.times != null ? info.times : "1";
  info.jp_bonus = info.jp_bonus != null ? info.jp_bonus : 0;
  info.is_all_lines = info.is_all_lines != null ? info.is_all_lines : 0;
  info.line_bets = info.line_bets != null ? info.line_bets : info.all_bets != null ? info.all_bets : "0";
  if (info.xiao_all_bonus == null) {
    info.xiao_all_bonus =
      (info.bonus_game_info && info.bonus_game_info.xiao_all_bonus) || info.sum_bonus || 0;
  }

  return applyRecordAssetBaseUrl(info, ossUrl);
};

const pickFirst = function () {
  for (let i = 0; i < arguments.length; i += 1) {
    const value = arguments[i];
    if (value !== null && value !== undefined && value !== "") return value;
  }
  return "";
};

const hasChinese = (value) => /[\u4e00-\u9fff]/.test(String(value || ""));

export const resolveChineseGameName = (row, gameOptions) => {
  const source = row || {};
  const gameId = Number(source.gameId || source.game_id || 0);
  const symbol = String(source.symbol || "");
  const options = Array.isArray(gameOptions) ? gameOptions : [];
  const hit =
    options.find((item) => Number(item.number) === gameId) ||
    options.find((item) => {
      const code = item.confName || item.symbol || "";
      return code && (code === symbol || code === source.gameName);
    });
  if (hit && (hit.nameZH || hit.NameZH)) return hit.nameZH || hit.NameZH;
  if (hit && hasChinese(hit.name)) return hit.name;

  const spin = parseMaybeJson(source.detail) || parseMaybeJson(source.log) || {};
  const fromLog = spin.listItem && spin.listItem.game_name;
  if (hasChinese(fromLog)) return fromLog;
  if (hasChinese(source.gameName)) return source.gameName;
  if (hasChinese(source.game_name)) return source.game_name;
  return pickFirst(source.gameName, source.game_name, symbol, gameId || "");
};

export const normalizeSettlementRow = (row, ossUrl) => {
  const source = row || {};
  const resolved = resolveDetailPayload(source);
  const rawInfo = resolved.info;
  const gameId = Number(
    pickFirst(
      rawInfo && rawInfo.game_id,
      source.gameId,
      source.game_id,
      0,
    ),
  );
  const recordId = String(
    pickFirst(
      source.roundID,
      source.id,
      rawInfo && rawInfo.ju_id,
      source.officeNumber,
    ),
  );
  const info = enrichDetailInfo(rawInfo, gameId, ossUrl || config.ossUrl, recordId);
  const totalBet = info && info.total_bet ? info.total_bet : null;

  return {
    raw: source,
    detail: resolved.detail,
    info: info,
    gameId: gameId,
    gameName: pickFirst(
      resolveChineseGameName(source),
      info && info.game_name,
    ),
    recordId: String(
      pickFirst(
        source.roundID,
        source.id,
        info && info.ju_id,
        source.officeNumber,
      ),
    ),
    time: pickFirst(info && info.time, formatPlayedTime(source.playedDate), source.time),
    allBets: pickFirst(
      info && info.all_bets,
      totalBet && totalBet.bet_num,
      source.bet,
      source.all_bets,
    ),
    allBonus: pickFirst(
      info && info.sum_bonus,
      totalBet && totalBet.reward_money,
      source.win,
      source.all_bonus,
    ),
    jpBonus: pickFirst(info && info.jp_bonus, source.jp_bonus),
    lineBets: pickFirst(info && info.line_bets),
    typeText: pickFirst(info && info.type, source.game_type),
    currency: pickFirst(source.currency, source.symbol),
    playerName: pickFirst(info && info.player_name, source.nickName, source.account),
  };
};

export const isFruitGameId = (gameId) => {
  const id = Number(gameId);
  return id >= 7000 && id < 8000;
};

export const isSlotDetail = (info) => {
  if (!info || typeof info !== "object") return false;
  if (Array.isArray(info.grids) && info.grids.length > 0) return true;
  // 有 type_id / lines_info 也按 slots 详情处理，避免缺 grids 时误判。
  if (info.type_id != null && !isFruitGameId(info.game_id)) return true;
  return false;
};

export const isFruitDetail = (info, gameId) => {
  if (!info || typeof info !== "object") return false;
  if (Array.isArray(info.grids) && info.grids.length > 0) return false;
  if (isFruitGameId(gameId || info.game_id)) return true;
  return Boolean(
    (Array.isArray(info.specific_bet_info) && info.specific_bet_info.length) ||
      (Array.isArray(info.result_show_url) && info.result_show_url.length) ||
      (Array.isArray(info.result_grid_url) && info.result_grid_url.length) ||
      info.total_bet,
  );
};

export const asImageList = (value) => {
  if (!value && value !== 0) return [];
  if (typeof value === "string") return value ? [value] : [];
  if (!Array.isArray(value)) return [];
  return value.filter((item) => typeof item === "string" && item);
};

/** result_grid_url 可能是一维图片数组，也可能是二维盘面。 */
export const asImageMatrix = (value) => {
  if (!Array.isArray(value) || !value.length) return [];
  if (Array.isArray(value[0])) {
    return value
      .map((row) =>
        Array.isArray(row) ? row.filter((item) => typeof item === "string" && item) : [],
      )
      .filter((row) => row.length);
  }
  const flat = value.filter((item) => typeof item === "string" && item);
  return flat.length ? [flat] : [];
};

const scatterSymbols = (scatter) => {
  const num = Math.max(Number(scatter && scatter.num) || 0, 0);
  const img = scatter && scatter.img_url;
  if (Array.isArray(img)) {
    return img.filter((item) => typeof item === "string" && item);
  }
  if (typeof img !== "string" || !img) return [];
  const count = num > 0 ? num : 1;
  const out = [];
  for (let i = 0; i < count; i += 1) out.push(img);
  return out;
};

/** 对齐记录页：scatter_info 追加进中奖线，展示为「特殊玩法」。 */
export const pickLineInfos = (info) => {
  if (!info || typeof info !== "object") return [];
  const isAllLines = Number(info.is_all_lines || 0);
  let source = info.lines_info;
  if (isAllLines === 2 || isAllLines === 3) {
    source = info.m_list_info;
  } else if (isAllLines === 1) {
    source =
      info.m_list_info && info.m_list_info.length
        ? info.m_list_info
        : info.lines_info;
  }
  const lines = Array.isArray(source) ? source.slice() : [];
  const scatters = Array.isArray(info.scatter_info) ? info.scatter_info : [];
  scatters.forEach((scatter) => {
    if (!scatter || typeof scatter !== "object") return;
    const symbols = scatterSymbols(scatter);
    lines.push({
      special: true,
      bets: scatter.bets,
      times: scatter.times,
      bonus: scatter.bonus,
      line_id: "特殊玩法",
      line_shape_url: "",
      direction: "",
      symbol: symbols,
      img_url: symbols,
      ts: 0,
    });
  });
  return lines;
};
