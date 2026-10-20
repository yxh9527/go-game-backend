<template>
  <div v-if="state == 0">
    <Card style="padding: 30px 0">
      <Row>
        <i-col span="8" offset="8">
          <Step :state="state" :data="stepList"></Step>
          <h3 style="margin: 20px 0">消息基本内容</h3>
          <Divider />
          <Form label-position="right" :label-width="120">
            <FormItem
              v-for="item of itemCreate"
              :key="item.lable"
              :label="item.lable"
            >
              <i-input
                v-if="item.type == 'text'"
                type="text"
                :maxlength="50"
                v-model="item.value"
                :placeholder="'请输入' + item.lable"
              ></i-input>

              <Select
                v-if="item.type == 'select'"
                v-model="item.value"
                style="width: 100%"
              >
                <Option
                  v-for="items in item.option"
                  :value="items.value"
                  :key="items.label"
                  >{{ items.label }}</Option
                >
              </Select>

              <Select
                v-if="item.type == 'list'"
                v-model="item.value"
                style="width: 100%"
              >
                <Option
                  v-for="items in item.option"
                  :value="items.id"
                  :key="items.account"
                  >{{ items.account }}</Option
                >
              </Select>

              <i-input
                v-if="item.type == 'textarea'"
                type="textarea"
                v-model="item.value"
                maxlength="150"
                show-word-limit
                :placeholder="'请输入' + item.lable"
                :rows="5"
              ></i-input>
              <Badge v-if="item.required" style="transform: translateY(-5px)">
                <Icon
                  type="ios-medical"
                  title="必填项"
                  style="margin-left: 20px"
                  slot="count"
                  color="#ed4014"
                  size="12"
                />
              </Badge>
            </FormItem>
            <FormItem>
              <Button type="primary" size="large" @click="submit"
                >创建消息</Button
              >
              <Button style="margin-left: 8px" size="large" @click="resForm"
                >重置</Button
              >
            </FormItem>
          </Form>
        </i-col>
      </Row>
    </Card>
  </div>
  <div v-else>
    <Card style="padding: 30px 0">
      <Row>
        <i-col span="8" offset="8">
          <Step :state="state" :data="stepList"></Step>
          <Card style="padding: 15px">
            <h3 style="width: 120px; text-align: right">基本信息：</h3>
            <List>
              <ListItem
                class="label-style"
                v-for="item of itemCreate"
                :key="item.key"
                :title="item.lable"
              >
                <label style="width: 120px; text-align: right"
                  >{{ item.lable }}：</label
                >
                {{ item.value }}
              </ListItem>
            </List>
          </Card>
          <div style="text-align: center; margin-top: 30px">
            <Button type="success" size="large" to="/site-message">完成</Button>
            <!-- <Button style="margin-left: 15px" size="large" @click="change">修改</Button> -->
          </div>
        </i-col>
      </Row>
    </Card>
  </div>
</template>

<script>
import Step from "_c/step";
import { addMsg, getMsgReceiveList } from "@/api/data";
export default {
  name: "game_add",
  components: {
    Step,
  },
  data() {
    return {
      state: this.$route.meta.state,
      stepList: [
        { title: "填写消息基本内容", content: "填写基本信息" },
        { title: "创建成功", content: "创建成功" },
      ],
      itemCreate: [
        {
          lable: "消息标题",
          type: "text",
          key: "title",
          value: "",
          required: true,
        },
        {
          lable: "消息类型",
          type: "select",
          key: "msgType",
          value: 1,
          option: [{ label: "管理消息", value: 1 }],
          required: true,
        },
        {
          lable: "收件人",
          type: "list",
          key: "receiveIds",
          value: "",
          option: [],
          required: true,
        },
        {
          lable: "消息内容",
          type: "textarea",
          key: "info",
          value: "",
          required: true,
        },
        { lable: "备注", type: "text", key: "remarks", value: "" },
      ],
    };
  },
  methods: {
    resForm() {
      this.itemCreate.forEach((item) => {
        item.value = "";
      });
    },
    next() {
      this.state += 1;
    },
    change() {
      this.state = 0;
    },
    submit() {
      const Data = [];
      this.itemCreate.map((item) => {
        Data.push({
          [item.key]: item.value,
        });
      });
      addMsg(Data).then((res) => {
        if (res.data.code == 200) {
          this.state += 1;
        }
      });
    },
  },
  mounted() {
    getMsgReceiveList().then((res) => {
      this.itemCreate.forEach((item) => {
        if (item.key == "receiveIds") {
          item.option.push(...res.data.data);
        }
      });
    });
  },
};
</script>

<style></style>
