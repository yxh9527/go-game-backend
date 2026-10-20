<template>
  <div
    class="valid-code disabled-select"
    :style="{ width, height }"
    @click="refreshCode"
  >
    <div class="valid-code__noise valid-code__noise--one"></div>
    <div class="valid-code__noise valid-code__noise--two"></div>
    <span
      v-for="(item, index) in codeList"
      :key="index"
      :style="item.style"
    >
      {{ item.code }}
    </span>
  </div>
</template>

<script>
export default {
  name: "ValidCode",
  model: {
    prop: "value",
    event: "input",
  },
  props: {
    value: {
      type: String,
      default: "",
    },
    width: {
      type: String,
      default: "120px",
    },
    height: {
      type: String,
      default: "40px",
    },
    length: {
      type: Number,
      default: 4,
    },
  },
  data() {
    return {
      codeList: [],
    };
  },
  mounted() {
    this.createCode();
  },
  methods: {
    refreshCode() {
      this.createCode();
    },
    createCode() {
      const chars = "ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz0123456789";
      const charsLen = chars.length;
      const nextCodeList = [];
      for (let index = 0; index < this.length; index += 1) {
        const color = `rgb(${Math.round(Math.random() * 160)}, ${Math.round(Math.random() * 130 + 40)}, ${Math.round(
          Math.random() * 180
        )})`;
        nextCodeList.push({
          code: chars.charAt(Math.floor(Math.random() * charsLen)),
          style: {
            color,
            fontSize: `${16 + Math.floor(Math.random() * 8)}px`,
            transform: `rotate(${Math.floor(Math.random() * 50) - 25}deg)`,
            padding: `0 ${2 + Math.floor(Math.random() * 4)}px`,
          },
        });
      }
      this.codeList = nextCodeList;
      this.$emit(
        "input",
        nextCodeList.map((item) => item.code).join("")
      );
    },
  },
};
</script>

<style scoped>
.valid-code {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(148, 163, 184, 0.24);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(219, 234, 254, 0.9));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.72);
  cursor: pointer;
  user-select: none;
  overflow: hidden;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.valid-code:hover {
  transform: translateY(-1px);
  border-color: rgba(37, 99, 235, 0.28);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.8),
    0 10px 20px rgba(37, 99, 235, 0.12);
}

.valid-code span {
  position: relative;
  z-index: 2;
  display: inline-block;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.valid-code__noise {
  position: absolute;
  inset: auto auto auto 0;
  width: 140%;
  height: 2px;
  background: linear-gradient(90deg, rgba(59, 130, 246, 0), rgba(59, 130, 246, 0.28), rgba(14, 165, 233, 0));
  transform-origin: center;
  opacity: 0.7;
}

.valid-code__noise--one {
  top: 12px;
  left: -12px;
  transform: rotate(8deg);
}

.valid-code__noise--two {
  bottom: 12px;
  left: -8px;
  transform: rotate(-7deg);
}
</style>
