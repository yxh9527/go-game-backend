<template>
  <div>
    <Card>
      <Button type="primary" style="margin-bottom: 10px" to="/site-message-add"
        >添加消息</Button
      >
      <tables ref="tables" v-model="tableData" :columns="columns" />
      <div class="pageStyle">
        <Page
          :total="pageData.current"
          :current="pageData.page"
          :page-size="pageData.pageSize"
          :page-size-opts="pageData.pageOpts"
          show-elevator
          show-sizer
          show-total
          title="输入页数后，按Enter键跳转页面"
          @on-change="changePage"
          @on-page-size-change="changePageSize"
        />
      </div>
    </Card>
  </div>
</template>
<script>
import Tables from "_c/tables";
import { getMsgList } from "@/api/data";
import { setting } from "@/config";
export default {
  name: "gameMessage",
  components: {
    Tables,
  },
  inject: ["handleLogOut"],
  data() {
    return {
      labelList: [
        {
          label: "邮件类型",
          value: "",
          option: [],
        },
      ],

      columns: [
        { title: "消息标题", key: "title", width: 200, align: "center" },
        { title: "消息内容", key: "info", align: "center" },
        {
          title: "发布时间",
          key: "createTime",
          width: 180,
          align: "center",
          render(h, params) {
            return (
              <span>
                {new Date(params.row.createTime * 1000).toLocaleString(
                  "chinese",
                  { hour12: false }
                )}
              </span>
            );
          },
        },
        {
          title: "消息类型",
          key: "msgType",
          render(h, params) {
            return params.row.msgType == 1 ? (
              <span>管理消息</span>
            ) : (
              <span>{params.row.msgType}</span>
            );
          },
        },
        {
          title: "备注",
          key: "remarks",
          align: "center",
          render(h, params) {
            return <span>{params.row.remarks || "-"}</span>;
          },
        },
        { title: "接收人", key: "account" },
      ],
      tableData: [],
      editData: [],
      pageData: {
        current: 0,
        page: setting.page,
        pageSize: setting.pageSize,
        pageOpts: setting.pageOpts,
      },
    };
  },
  methods: {
    handleDelete(params) {},
    handleSearch() {
      let Data = [
        { page: this.pageData.page },
        { pageSize: this.pageData.pageSize },
      ];
      if (this.startTime) {
        Data.push({
          startTime: this.startTime
            .toLocaleString("chinese", { hour12: false })
            .replace(/\//g, "-"),
        });
      }
      if (this.endTime) {
        Data.push({
          endTime: this.endTime
            .toLocaleString("chinese", { hour12: false })
            .replace(/\//g, "-"),
        });
      }
      getMsgList(Data)
        .then((res) => {
          this.tableData = [];
          if (res.data.code == 200) {
            this.tableData.push(...res.data.data.data);
            this.pageData.current = res.data.data.total;
          } else {
            // 判断响应状态是否为Token失效，如果失效则执行退出函数并刷新页面。
            this.$nextTick(() => {
              if (setting.arrStatus.indexOf(res.data.code) != -1) {
                this.$Message.error(
                  res.data.code + " ：&nbsp;" + res.data.msg + "请重新登录"
                );
                this.handleLogOut();
              } else {
                this.$Message.error(res.data.code + " ：&nbsp;" + res.data.msg);
              }
            });
          }
        })
        .catch((err) => {
          this.$Message.error("服务器未响应，请重新登录或联系管理员");
        });
    },
    changePage(index) {
      this.pageData.page = index;
      this.handleSearch();
    },
    changePageSize(index) {
      this.pageData.pageSize = index;
      this.handleSearch();
    },
  },
  mounted() {
    this.handleSearch();
  },
};
</script>

<style>
.pageStyle {
  margin-top: 20px;
  text-align: right;
}
</style>
