<template>
  <div class="user-avatar-dropdown">
    <Button
      type="dashed"
      @click="message"
      style="margin-right: 10px; padding: 0px 10px"
    >
      消息中心
      <!-- <Badge style="margin-left: 10px" :count="messageUnread"></Badge> -->
    </Button>
    <Dropdown @on-click="handleClick">
      <Badge :dot="!!messageUnreadCount">
        <Avatar :src="userAvatar" />
      </Badge>
      <Icon :size="18" type="md-arrow-dropdown"></Icon>
      <DropdownMenu slot="list">
        <DropdownItem name="logout">退出登录</DropdownItem>
      </DropdownMenu>
    </Dropdown>
  </div>
</template>

<script>
import "./user.less";
import { mapActions } from "vuex";
import { getUnreadCount } from "@/api/user";
export default {
  name: "User",
  props: {
    userAvatar: {
      type: String,
      default: "",
    },
    messageUnreadCount: {
      type: Number,
      default: 0,
    },
  },
  data() {
    return {
      messageUnread: 0,
    };
  },
  methods: {
    ...mapActions(["handleLogOut"]),
    logout() {
      this.handleLogOut().then(() => {
        this.$router.push({
          name: "login",
        });
      });
    },
    message() {
      this.$router.push({
        name: "message_page",
      });
    },
    handleClick(name) {
      switch (name) {
        case "logout":
          this.logout();
          break;
        case "message":
          this.message();
          break;
      }
    },
  },
  mounted() {
    getUnreadCount().then((res) => {
      this.messageUnread = 0;
      if (res.data.code == 200) {
        res.data.data.map((item) => {
          if (item.isRead == 0) {
            this.messageUnread++;
          }
        });
      }
    });
  },
};
</script>
