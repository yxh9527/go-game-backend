<template>
  <div class="record-dialog" :class="{ embedded: embedded }">
    <template v-if="normalized.info">
      <div class="record-hero">
        <div class="record-hero__main">
          <div class="record-kicker">{{ heroType }}</div>
          <div class="record-title">{{ heroTitle }}</div>
          <div class="record-sub">
            <span>局号 {{ heroRound }}</span>
            <span v-if="heroTime">{{ heroTime }}</span>
            <span v-if="heroPlayer">{{ heroPlayer }}</span>
          </div>
        </div>
        <div class="record-hero__stats">
          <div class="hero-stat">
            <div class="hero-stat__label">投注</div>
            <div class="hero-stat__value">{{ formatAmount(normalized.allBets) }}</div>
          </div>
          <div class="hero-stat">
            <div class="hero-stat__label">派彩</div>
            <div class="hero-stat__value" :class="payoutClass">
              {{ formatAmount(normalized.allBonus) }}
            </div>
          </div>
        </div>
      </div>

      <div v-if="metaChips.length" class="record-chips">
        <div v-for="item in metaChips" :key="'chip-' + item.label" class="record-chip">
          <span class="record-chip__label">{{ item.label }}</span>
          <span class="record-chip__value">{{ item.value }}</span>
        </div>
      </div>

      <template v-if="mode === 'slot'">
        <section class="record-stage">
          <div class="stage-head">
            <div class="stage-title">开奖盘面</div>
            <div v-if="slotColumns.length" class="stage-note">{{ slotColumns.length }} 轴</div>
          </div>
          <div v-if="slotColumns.length" class="reel-wrap">
            <div class="reel-board" :style="slotBoardStyle">
              <div
                v-for="(col, colIndex) in slotColumns"
                :key="'col-' + colIndex"
                class="reel-col"
              >
                <div
                  v-for="(src, rowIndex) in col"
                  :key="'cell-' + colIndex + '-' + rowIndex"
                  class="reel-cell"
                >
                  <img class="reel-icon" :src="src" alt="" />
                </div>
              </div>
            </div>
          </div>
          <el-alert
            v-else
            title="该 slots 注单没有可用的盘面图片。"
            type="warning"
            :closable="false"
            show-icon
          />
        </section>

        <section v-if="lineInfos.length" class="record-stage">
          <div class="stage-head">
            <div class="stage-title">中奖线</div>
            <div class="stage-note">{{ lineInfos.length }} 条</div>
          </div>
          <div class="line-list">
            <div
              v-for="(line, index) in lineInfos"
              :key="'line-' + index"
              class="line-card"
              :class="{ 'is-special': line.special }"
            >
              <div class="line-head">
                <span class="line-id">{{ lineTitle(line, index) }}</span>
                <span v-if="!line.special && line.direction" class="line-dir">{{ line.direction }}</span>
                <span class="line-times">{{ lineTimes(line) }}</span>
                <span class="line-bonus">+{{ formatAmount(line.bonus) }}</span>
              </div>
              <div class="line-body">
                <img
                  v-if="!line.special && line.line_shape_url"
                  class="line-shape"
                  :src="line.line_shape_url"
                  alt=""
                />
                <div class="line-symbols">
                  <img
                    v-for="(symbol, symbolIndex) in lineSymbols(line)"
                    :key="'symbol-' + index + '-' + symbolIndex"
                    class="line-symbol"
                    :src="symbol"
                    alt=""
                  />
                </div>
                <div v-if="!line.special && Number(line.bets)" class="line-meta">
                  投注 {{ formatAmount(line.bets) }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </template>

      <template v-else-if="mode === 'fruit'">
        <section v-if="rewardTexts.length" class="record-stage">
          <div class="stage-head">
            <div class="stage-title">开奖结果</div>
          </div>
          <div class="reward-tags">
            <span
              v-for="(text, index) in rewardTexts"
              :key="'reward-' + index"
              class="reward-tag"
            >
              {{ text }}
            </span>
          </div>
        </section>

        <section v-if="resultShowImages.length" class="record-stage">
          <div class="stage-head">
            <div class="stage-title">开奖图标</div>
          </div>
          <div class="image-row">
            <img
              v-for="(src, index) in resultShowImages"
              :key="'result-' + index"
              class="result-icon"
              :src="src"
              alt=""
            />
          </div>
        </section>

        <section v-if="resultGridMatrix.length" class="record-stage">
          <div class="stage-head">
            <div class="stage-title">开奖盘面</div>
          </div>
          <div class="reel-wrap">
            <div class="matrix-board">
              <div
                v-for="(row, rowIndex) in resultGridMatrix"
                :key="'result-row-' + rowIndex"
                class="matrix-row"
              >
                <div
                  v-for="(src, colIndex) in row"
                  :key="'result-cell-' + rowIndex + '-' + colIndex"
                  class="reel-cell"
                >
                  <img class="matrix-icon" :src="src" alt="" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-if="specificBets.length" class="record-stage">
          <div class="stage-head">
            <div class="stage-title">下注详情</div>
            <div class="stage-note">{{ specificBets.length }} 项</div>
          </div>
          <div class="bet-list">
            <div
              v-for="(bet, index) in specificBets"
              :key="'bet-' + index"
              class="bet-card"
            >
              <div class="bet-head">
                <img
                  v-if="isImageUrl(bet.bet_area_url)"
                  class="bet-area"
                  :src="bet.bet_area_url"
                  alt=""
                />
                <div class="bet-head-text">
                  <div>下注 {{ formatAmount(bet.bet_num) }}</div>
                  <div class="bet-head-sub">
                    赔率 {{ formatRate(bet.reward_rate) }} ·
                    派彩 {{ formatAmount(bet.reward_money) }}
                  </div>
                </div>
              </div>
              <div v-if="betGridMatrix(bet).length" class="matrix-board compact">
                <div
                  v-for="(row, rowIndex) in betGridMatrix(bet)"
                  :key="'bet-grid-' + index + '-' + rowIndex"
                  class="matrix-row"
                >
                  <div
                    v-for="(src, colIndex) in row"
                    :key="'bet-cell-' + index + '-' + rowIndex + '-' + colIndex"
                    class="reel-cell"
                  >
                    <img class="matrix-icon" :src="src" alt="" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <div v-if="fruitExtraItems.length" class="record-chips">
          <div
            v-for="item in fruitExtraItems"
            :key="'extra-' + item.label"
            class="record-chip"
          >
            <span class="record-chip__label">{{ item.label }}</span>
            <span class="record-chip__value">{{ item.value }}</span>
          </div>
        </div>
      </template>

      <template v-else>
        <el-alert
          title="当前详情结构未识别为 slots/fruits 标准注单，已展示原始摘要。"
          type="warning"
          :closable="false"
          show-icon
        />
        <pre class="raw-json">{{ rawInfoJson }}</pre>
      </template>
    </template>

    <el-empty v-else description="没有可展示的注单详情数据" />
  </div>
</template>

<script>
import config from "@/config";
import {
  asImageList,
  asImageMatrix,
  formatAmount,
  isFruitDetail,
  isSlotDetail,
  normalizeSettlementRow,
  pickLineInfos,
} from "./settlementHelpers";

export default {
  name: "SettlementRecordDialog",
  props: {
    row: {
      type: Object,
      default: null,
    },
    embedded: {
      type: Boolean,
      default: false,
    },
    ossUrl: {
      type: String,
      default: "",
    },
  },
  computed: {
    assetBase() {
      return this.ossUrl || config.ossUrl || "";
    },
    normalized() {
      return normalizeSettlementRow(this.row || {}, this.assetBase);
    },
    mode() {
      const info = this.normalized.info;
      if (isSlotDetail(info)) return "slot";
      if (isFruitDetail(info, this.normalized.gameId)) return "fruit";
      return "unknown";
    },
    heroTitle() {
      return this.normalized.gameName || this.normalized.gameId || "游戏详情";
    },
    heroType() {
      return this.normalized.typeText || (this.mode === "fruit" ? "Fruit" : "Slots");
    },
    heroRound() {
      return this.normalized.recordId || "-";
    },
    heroTime() {
      return this.normalized.time || "";
    },
    heroPlayer() {
      return this.normalized.playerName || "";
    },
    payoutClass() {
      const value = Number(this.normalized.allBonus || 0);
      if (value > 0) return "is-win";
      if (value < 0) return "is-lose";
      return "is-zero";
    },
    metaChips() {
      const item = this.normalized;
      const info = item.info || {};
      const rows = [];
      if (item.gameId) rows.push({ label: "游戏ID", value: String(item.gameId) });
      if (item.lineBets !== "" && item.lineBets != null) {
        rows.push({ label: "线注", value: formatAmount(item.lineBets) });
      }
      if (info.times != null && info.times !== "") {
        rows.push({ label: "倍数", value: String(info.times) });
      }
      if (item.currency) rows.push({ label: "货币", value: item.currency });
      if (item.jpBonus !== "" && item.jpBonus != null && Number(item.jpBonus) !== 0) {
        rows.push({ label: "JP", value: formatAmount(item.jpBonus) });
      }
      return rows;
    },
    slotColumns() {
      const info = this.normalized.info || {};
      return Array.isArray(info.grids) ? info.grids : [];
    },
    slotBoardStyle() {
      const count = Math.max(this.slotColumns.length, 1);
      return {
        gridTemplateColumns: "repeat(" + count + ", 96px)",
      };
    },
    lineInfos() {
      return pickLineInfos(this.normalized.info);
    },
    rewardTexts() {
      const info = this.normalized.info || {};
      const rewards = info.reward_result;
      if (!Array.isArray(rewards)) return [];
      return rewards
        .map(function (item) {
          if (typeof item === "string" || typeof item === "number") return String(item);
          return "";
        })
        .filter(Boolean);
    },
    resultShowImages() {
      const info = this.normalized.info || {};
      return asImageList(info.result_show_url);
    },
    resultGridMatrix() {
      const info = this.normalized.info || {};
      return asImageMatrix(info.result_grid_url);
    },
    specificBets() {
      const info = this.normalized.info || {};
      return Array.isArray(info.specific_bet_info) ? info.specific_bet_info : [];
    },
    fruitExtraItems() {
      const info = this.normalized.info || {};
      const rows = [];
      if (info.bet_num != null && info.bet_num !== "") {
        rows.push({ label: "弹珠数/注数", value: String(info.bet_num) });
      }
      if (info.risk) {
        rows.push({ label: "风险", value: String(info.risk) });
      }
      if (info.num != null && info.num !== "") {
        rows.push({ label: "档位", value: String(info.num) });
      }
      if (info.level != null && Number(info.level) !== 0) {
        rows.push({ label: "等级", value: String(info.level) });
      }
      if (info.rate != null && info.rate !== "") {
        rows.push({ label: "倍率", value: String(info.rate) });
      }
      if (info.reward_money != null && info.reward_money !== "") {
        rows.push({ label: "奖励金额", value: formatAmount(info.reward_money) });
      }
      return rows;
    },
    rawInfoJson() {
      return JSON.stringify(this.normalized.info || {}, null, 2);
    },
  },
  methods: {
    formatAmount: formatAmount,
    asImageList: asImageList,
    betGridMatrix: function (bet) {
      if (!bet || !bet.grid) return [];
      return asImageMatrix(bet.grid);
    },
    isImageUrl: function (value) {
      return (
        typeof value === "string" &&
        (/^(https?:)?\/\//.test(value) || value.indexOf("/") === 0)
      );
    },
    formatRate: function (value) {
      const number = Number(value);
      if (!Number.isFinite(number) || number === 0) return formatAmount(0);
      if (Math.abs(number) >= 10) return (number / 100).toFixed(2);
      return formatAmount(number);
    },
    lineTitle: function (line, index) {
      if (line && line.special) return "特殊玩法";
      return "线 " + ((line && line.line_id) || index + 1);
    },
    lineTimes: function (line) {
      const times = (line && (line.times || line.ts)) || 1;
      return line && line.special ? "X " + times : "x" + times;
    },
    lineSymbols: function (line) {
      if (!line) return [];
      const fromSymbol = asImageList(line.symbol);
      if (fromSymbol.length) return fromSymbol;
      return asImageList(line.img_url);
    },
  },
};
</script>

<style scoped>
.record-dialog {
  min-height: 120px;
  color: var(--text-main);
}

.record-dialog.embedded {
  max-height: calc(100vh - 140px);
  overflow: auto;
  padding: 2px 2px 8px;
}

.record-hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  padding: 18px 20px;
  border-radius: 16px;
  background:
    radial-gradient(circle at 0 0, rgba(37, 99, 235, 0.12), transparent 42%),
    linear-gradient(180deg, #f8fbff 0%, #eef3f9 100%);
  border: 1px solid #dbe4ee;
}

.record-kicker {
  color: #5b6b82;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.record-title {
  margin-top: 6px;
  font-size: 24px;
  font-weight: 800;
  line-height: 1.2;
  letter-spacing: 0.01em;
}

.record-sub {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 8px;
  color: var(--text-sub);
  font-size: 13px;
}

.record-hero__stats {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

.hero-stat {
  min-width: 112px;
  padding: 10px 14px;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.06);
}

.hero-stat__label {
  color: var(--text-faint);
  font-size: 12px;
}

.hero-stat__value {
  margin-top: 4px;
  font-size: 22px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.hero-stat__value.is-win {
  color: #15803d;
}

.hero-stat__value.is-lose,
.hero-stat__value.is-zero {
  color: #dc2626;
}

.record-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.record-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  background: #f4f7fb;
  border: 1px solid #e4ebf2;
}

.record-chip__label {
  color: var(--text-faint);
  font-size: 12px;
}

.record-chip__value {
  color: var(--text-main);
  font-size: 13px;
  font-weight: 700;
}

.record-stage {
  margin-top: 18px;
  padding: 16px 16px 18px;
  border: 1px solid #e4ebf2;
  border-radius: 16px;
  background: #fff;
}

.stage-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 14px;
}

.stage-title {
  font-size: 15px;
  font-weight: 800;
}

.stage-note {
  color: var(--text-faint);
  font-size: 12px;
}

.reel-wrap {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}

.reel-board,
.matrix-board {
  display: grid;
  gap: 10px;
  width: fit-content;
  max-width: 100%;
  padding: 18px;
  border-radius: 18px;
  background:
    radial-gradient(circle at 50% 0, rgba(250, 204, 21, 0.16), transparent 46%),
    linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    0 16px 32px rgba(15, 23, 42, 0.28);
}

.reel-col,
.matrix-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 6px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.55);
}

.matrix-row {
  flex-direction: row;
}

.reel-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96px;
  height: 96px;
  border-radius: 14px;
  background: linear-gradient(180deg, #334155 0%, #1e293b 100%);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
}

.reel-icon,
.matrix-icon,
.result-icon,
.bet-area,
.line-symbol,
.line-shape {
  display: block;
  object-fit: contain;
}

.reel-icon {
  width: 84px;
  height: 84px;
}

.matrix-board.compact {
  margin-top: 12px;
  padding: 10px;
}

.matrix-icon {
  width: 44px;
  height: 44px;
}

.matrix-board.compact .reel-cell {
  width: 52px;
  height: 52px;
}

.image-row {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}

.result-icon {
  width: 64px;
  height: 64px;
  padding: 6px;
  border-radius: 12px;
  background: #0f172a;
}

.reward-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.reward-tag {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 700;
}

.line-list,
.bet-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: min(260px, 32vh);
  overflow-y: auto;
  padding-right: 4px;
}

.line-list::-webkit-scrollbar,
.bet-list::-webkit-scrollbar {
  width: 6px;
}

.line-list::-webkit-scrollbar-thumb,
.bet-list::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: #c5d0dc;
}

.line-list::-webkit-scrollbar-track,
.bet-list::-webkit-scrollbar-track {
  background: transparent;
}

.line-card,
.bet-card {
  padding: 12px 14px;
  border: 1px solid #e8eef4;
  border-radius: 14px;
  background: #f8fafc;
}

.line-head,
.bet-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.line-id,
.line-dir,
.line-times {
  color: var(--text-sub);
  font-size: 13px;
}

.line-bonus {
  margin-left: auto;
  color: #15803d;
  font-size: 15px;
  font-weight: 800;
}

.line-body {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.line-shape {
  width: 80px;
  height: 36px;
}

.line-symbols {
  display: flex;
  gap: 4px;
}

.line-symbol {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: #fff;
}

.line-meta,
.bet-head-sub {
  color: var(--text-sub);
  font-size: 12px;
}

.bet-area {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: #fff;
}

.bet-head-text {
  color: var(--text-main);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.5;
}

.raw-json {
  margin-top: 12px;
  padding: 12px;
  border-radius: 10px;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 12px;
  overflow: auto;
  max-height: 360px;
}

@media (max-width: 768px) {
  .record-hero {
    flex-direction: column;
    align-items: stretch;
  }

  .record-hero__stats {
    width: 100%;
  }

  .hero-stat {
    flex: 1;
  }

  .reel-board {
    grid-template-columns: repeat(auto-fit, 72px) !important;
  }

  .reel-cell {
    width: 72px;
    height: 72px;
  }

  .reel-icon {
    width: 62px;
    height: 62px;
  }
}
</style>
