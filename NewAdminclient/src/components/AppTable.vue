<template>
  <el-table
    :data="data"
    border
    stripe
    class="app-table"
    v-loading="loading"
    :header-cell-style="headerCellStyle"
    :cell-style="cellStyle"
  >
    <template v-for="column in columns">
      <el-table-column
        v-if="column.type !== 'action'"
        :key="column.key || column.title"
        :prop="column.key"
        :label="column.title"
        :align="column.align || 'center'"
        :min-width="column.minWidth"
        :width="column.width"
        show-overflow-tooltip
      >
        <template slot-scope="scope">
          <table-render
            v-if="column.render"
            :render="column.render"
            :row="scope.row"
            :index="scope.$index"
          />
          <span v-else>{{ scope.row[column.key] }}</span>
        </template>
      </el-table-column>
      <el-table-column
        v-else
        :key="column.title"
        :label="column.title"
        :align="column.align || 'center'"
        :min-width="column.minWidth"
        :width="column.width"
      >
        <template slot-scope="scope">
          <div :class="['app-table__actions', { 'app-table__actions--wrap': column.wrapActions }]">
            <el-button
              v-for="button in column.buttons"
              :key="button.label"
              type="text"
              size="small"
              :class="{ 'app-table__action-button--compact': column.wrapActions }"
              @click="button.onClick(scope.row, scope.$index)"
            >
              {{ button.label }}
            </el-button>
          </div>
        </template>
      </el-table-column>
    </template>
  </el-table>
</template>

<script>
const TableRender = {
  name: "TableRender",
  functional: true,
  props: {
    render: {
      type: Function,
      required: true,
    },
    row: {
      type: Object,
      required: true,
    },
    index: {
      type: Number,
      required: true,
    },
  },
  render(h, ctx) {
    return ctx.props.render(h, {
      row: ctx.props.row,
      index: ctx.props.index,
    });
  },
};

export default {
  name: "AppTable",
  components: {
    TableRender,
  },
  props: {
    data: {
      type: Array,
      default: () => [],
    },
    columns: {
      type: Array,
      default: () => [],
    },
    loading: {
      type: Boolean,
      default: false,
    },
  },
  methods: {
    headerCellStyle() {
      return {
        background: "#f7f9fc",
        color: "#526176",
        padding: "11px 0",
      };
    },
    cellStyle() {
      return {
        padding: "9px 0",
      };
    },
  },
};
</script>

<style scoped>
.app-table {
  width: 100%;
}

.app-table__actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.app-table__actions--wrap {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px 8px;
}

.app-table /deep/ .el-button--text {
  font-weight: 600;
}

.app-table__action-button--compact {
  min-width: 56px;
  margin: 0;
  padding: 3px 0;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.08);
  color: #1d4ed8;
}
</style>
